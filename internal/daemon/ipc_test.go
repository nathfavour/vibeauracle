package daemon

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"
)

type MockProcessor struct {
	ProcessFunc func(ctx context.Context, req interface{}) (interface{}, error)
}

func (m *MockProcessor) Process(ctx context.Context, req interface{}) (interface{}, error) {
	if m.ProcessFunc != nil {
		return m.ProcessFunc(ctx, req)
	}
	return map[string]interface{}{"content": "mock response"}, nil
}

func (m *MockProcessor) GetSnapshot() (interface{}, error) {
	return map[string]string{"status": "ok"}, nil
}

func (m *MockProcessor) Config() interface{} {
	return map[string]string{"key": "value"}
}

func TestIPCServer(t *testing.T) {
	tmpDir := t.TempDir()
	socketPath := filepath.Join(tmpDir, "test.sock")
	
	p := &MockProcessor{}
	server := NewServer(socketPath, p)
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	go func() {
		if err := server.Start(ctx); err != nil && ctx.Err() == nil {
			fmt.Printf("Server error: %v\n", err)
		}
	}()
	
	// Wait for socket
	time.Sleep(100 * time.Millisecond)
	
	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		t.Fatalf("failed to connect to socket: %v", err)
	}
	defer conn.Close()
	
	// Test Ping
	ping := IPCMessage{
		Type: "request",
		Method: "ping",
		ID: "1",
	}
	data, _ := json.Marshal(ping)
	fmt.Fprintln(conn, string(data))
	
	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}
	
	var resp IPCMessage
	if err := json.Unmarshal([]byte(line), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	
	if resp.Type != "response" || resp.ID != "1" {
		t.Errorf("unexpected response: %+v", resp)
	}
	
	// Test Query
	p.ProcessFunc = func(ctx context.Context, req interface{}) (interface{}, error) {
		return map[string]interface{}{"content": "processed!"}, nil
	}
	
	query := IPCMessage{
		Type: "request",
		Method: "query",
		ID: "2",
		Payload: json.RawMessage(`{"content": "hello"}`),
	}
	data, _ = json.Marshal(query)
	fmt.Fprintln(conn, string(data))
	
	line, _ = reader.ReadString('
')
	json.Unmarshal([]byte(line), &resp)
	
	var payload map[string]string
	json.Unmarshal(resp.Payload, &payload)
	if payload["content"] != "processed!" {
		t.Errorf("expected 'processed!', got %s", payload["content"])
	}
}
