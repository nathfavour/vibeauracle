package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
)

// Processor defines the interface that the daemon expects from the core engine.
// This breaks the circular dependency between 'brain' and 'daemon'.
type Processor interface {
	Process(ctx context.Context, req interface{}) (interface{}, error)
	GetSnapshot() (interface{}, error)
	Config() interface{}
}

// IPCMessage represents a generic message over the UDS
type IPCMessage struct {
	Type    string          `json:"type"`             // request, response, event
	Method  string          `json:"method,omitempty"` // query, status, config, etc.
	ID      string          `json:"id,omitempty"`     // correlation ID
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Server handles IPC via Unix Domain Socket
type Server struct {
	socketPath string
	processor  Processor
	mu         sync.RWMutex
	listeners  []net.Listener
}

func NewServer(socketPath string, p Processor) *Server {
	return &Server{
		socketPath: socketPath,
		processor:  p,
	}
}

// Start launches the UDS server
func (s *Server) Start(ctx context.Context) error {
	// Clean up existing socket if any
	if _, err := os.Stat(s.socketPath); err == nil {
		_ = os.Remove(s.socketPath)
	}

	lis, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("listening on unix socket: %w", err)
	}
	s.listeners = append(s.listeners, lis)

	// Ensure socket is accessible but secure
	_ = os.Chmod(s.socketPath, 0600)

	go func() {
		<-ctx.Done()
		lis.Close()
	}()

	for {
		conn, err := lis.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				fmt.Fprintf(os.Stderr, "Error accepting connection: %v\n", err)
				continue
			}
		}

		go s.handleConnection(ctx, conn)
	}
}

func (s *Server) handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	scanner := bufio.NewScanner(conn)
	for scanner.Scan() {
		var msg IPCMessage
		if err := json.Unmarshal(scanner.Bytes(), &msg); err != nil {
			s.sendError(conn, "", "invalid json")
			continue
		}

		s.handleMessage(ctx, conn, msg)
	}
}

func (s *Server) handleMessage(ctx context.Context, conn net.Conn, msg IPCMessage) {
	switch msg.Type {
	case "request":
		switch msg.Method {
		case "ping":
			s.sendResponse(conn, msg.ID, map[string]string{"status": "pong"})
		case "query":
			var payload struct {
				Content  string `json:"content"`
				Intent   string `json:"intent"`   // Optional override
				Provider string `json:"provider"` // Optional provider (e.g. github-models)
				Model    string `json:"model"`    // Optional model name
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				s.sendError(conn, msg.ID, "invalid query payload")
				return
			}

			// External tools connecting via UDS usually want clean model access.
			// We default to 'vibe' intent if not specified.
			intent := payload.Intent
			if intent == "" {
				intent = "vibe"
			}

			// Execute query via Processor (Brain)
			// We pass a generic map/struct that the Brain can unmarshal or type-assert
			resp, err := s.processor.Process(ctx, map[string]interface{}{
				"id":       msg.ID,
				"content":  payload.Content,
				"intent":   intent,
				"provider": payload.Provider,
				"model":    payload.Model,
			})
			if err != nil {
				s.sendError(conn, msg.ID, err.Error())
				return
			}

			// Handle different response types if necessary
			content := ""
			reasoning := ""
			if r, ok := resp.(interface{ GetContent() string }); ok {
				content = r.GetContent()
			} else if m, ok := resp.(map[string]interface{}); ok {
				content, _ = m["content"].(string)
			}

			if r, ok := resp.(interface{ GetReasoning() string }); ok {
				reasoning = r.GetReasoning()
			} else if m, ok := resp.(map[string]interface{}); ok {
				reasoning, _ = m["reasoning"].(string)
			}

			s.sendResponse(conn, msg.ID, map[string]string{
				"content":   content,
				"reasoning": reasoning,
			})

		case "status":
			snapshot, _ := s.processor.GetSnapshot()
			s.sendResponse(conn, msg.ID, snapshot)

		case "config":
			s.sendResponse(conn, msg.ID, s.processor.Config())

		case "vault_get":
			var payload struct {
				Key string `json:"key"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				s.sendError(conn, msg.ID, "invalid vault_get payload")
				return
			}
			// We need to type assert the processor to something that has a Vault
			// or add it to the Processor interface. For now, we'll try to get it
			// if the processor has a GetVault() method.
			if v, ok := s.processor.(interface{ GetVault() interface{ Get(string) (string, error) } }); ok {
				val, err := v.GetVault().Get(payload.Key)
				if err != nil {
					s.sendError(conn, msg.ID, err.Error())
					return
				}
				s.sendResponse(conn, msg.ID, map[string]string{"value": val})
			} else {
				s.sendError(conn, msg.ID, "vault not accessible via processor")
			}

		case "vault_set":
			var payload struct {
				Key   string `json:"key"`
				Value string `json:"value"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				s.sendError(conn, msg.ID, "invalid vault_set payload")
				return
			}
			if v, ok := s.processor.(interface{ GetVault() interface{ Set(string, string) error } }); ok {
				err := v.GetVault().Set(payload.Key, payload.Value)
				if err != nil {
					s.sendError(conn, msg.ID, err.Error())
					return
				}
				s.sendResponse(conn, msg.ID, map[string]string{"status": "ok"})
			} else {
				s.sendError(conn, msg.ID, "vault not accessible via processor")
			}

		default:
			s.sendError(conn, msg.ID, "unknown method: "+msg.Method)
		}
	default:
		s.sendError(conn, msg.ID, "unsupported message type: "+msg.Type)
	}
}

func (s *Server) sendResponse(w io.Writer, id string, payload interface{}) {
	data, _ := json.Marshal(payload)
	resp := IPCMessage{
		Type:    "response",
		ID:      id,
		Payload: data,
	}
	encoded, _ := json.Marshal(resp)
	fmt.Fprintln(w, string(encoded))
}

func (s *Server) sendError(w io.Writer, id string, errMsg string) {
	resp := IPCMessage{
		Type:    "error",
		ID:      id,
		Payload: json.RawMessage(fmt.Sprintf(`{"message": %q}`, errMsg)),
	}
	encoded, _ := json.Marshal(resp)
	fmt.Fprintln(w, string(encoded))
}
