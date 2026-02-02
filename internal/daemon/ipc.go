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

	"github.com/nathfavour/vibeauracle/brain"
)

// IPCMessage represents a generic message over the UDS
type IPCMessage struct {
	Type    string          `json:"type"`              // request, response, event
	Method  string          `json:"method,omitempty"`  // query, status, config, etc.
	ID      string          `json:"id,omitempty"`      // correlation ID
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Server handles IPC via Unix Domain Socket
type Server struct {
	socketPath string
	brain      *brain.Brain
	mu         sync.RWMutex
	listeners  []net.Listener
}

func NewServer(socketPath string, b *brain.Brain) *Server {
	return &Server{
		socketPath: socketPath,
		brain:      b,
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
				Content string `json:"content"`
			}
			if err := json.Unmarshal(msg.Payload, &payload); err != nil {
				s.sendError(conn, msg.ID, "invalid query payload")
				return
			}

			// Execute query via Brain
			resp, err := s.brain.Process(ctx, brain.Request{
				ID:      msg.ID,
				Content: payload.Content,
			})
			if err != nil {
				s.sendError(conn, msg.ID, err.Error())
				return
			}

			s.sendResponse(conn, msg.ID, map[string]string{"content": resp.Content})

		case "status":
			snapshot, _ := s.brain.GetSnapshot()
			s.sendResponse(conn, msg.ID, snapshot)

		case "config":
			s.sendResponse(conn, msg.ID, s.brain.Config())

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
		Type: "error",
		ID:   id,
		Payload: json.RawMessage(fmt.Sprintf(`{"message": %q}`, errMsg)),
	}
	encoded, _ := json.Marshal(resp)
	fmt.Fprintln(w, string(encoded))
}
