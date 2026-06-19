package ipc

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"time"
)

type Request struct {
	Type    string      `json:"type"`
	Method  string      `json:"method"`
	ID      string      `json:"id"`
	Payload interface{} `json:"payload"`
}

type Response struct {
	Type    string          `json:"type"`
	ID      string          `json:"id"`
	Payload json.RawMessage `json:"payload"`
}

type QueryResult struct {
	Content   string `json:"content"`
	Reasoning string `json:"reasoning"`
}

// Query sends a mutation/query payload over the vibeauracle UDS.
func Query(content string) (string, error) {
	conn, err := net.DialTimeout("unix", SocketPath(), 5*time.Second)
	if err != nil {
		return "", fmt.Errorf("vibeauracle uds: %w", err)
	}
	defer conn.Close()

	id := fmt.Sprintf("polygeist-%d", time.Now().UnixNano())
	req := Request{
		Type:   "request",
		Method: "query",
		ID:     id,
		Payload: map[string]string{
			"content": content,
			"intent":  "vibe",
		},
	}
	data, err := json.Marshal(req)
	if err != nil {
		return "", err
	}
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(conn)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)
	if !scanner.Scan() {
		return "", fmt.Errorf("no response from vibeauracle")
	}

	var resp Response
	if err := json.Unmarshal(scanner.Bytes(), &resp); err != nil {
		return "", err
	}
	if resp.Type == "error" {
		return "", fmt.Errorf("%s", string(resp.Payload))
	}

	var result QueryResult
	if err := json.Unmarshal(resp.Payload, &result); err != nil {
		return string(resp.Payload), nil
	}
	if result.Content != "" {
		return result.Content, nil
	}
	return result.Reasoning, nil
}

// Ping checks whether the vibeauracle daemon is reachable.
func Ping() error {
	conn, err := net.DialTimeout("unix", SocketPath(), 2*time.Second)
	if err != nil {
		return err
	}
	defer conn.Close()

	id := fmt.Sprintf("ping-%d", time.Now().UnixNano())
	req := Request{Type: "request", Method: "ping", ID: id}
	data, _ := json.Marshal(req)
	if _, err := conn.Write(append(data, '\n')); err != nil {
		return err
	}
	scanner := bufio.NewScanner(conn)
	if !scanner.Scan() {
		return fmt.Errorf("no pong")
	}
	return nil
}
