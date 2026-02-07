package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/nathfavour/vibeauracle/sys"
)

type MockTool struct {
	Name string
}

func (m *MockTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name: m.Name,
		Description: "A mock tool",
		Parameters: json.RawMessage(`{"type": "object"}`),
	}
}

func (m *MockTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	return &ToolResult{Status: "success", Content: "done"}, nil
}

func TestRegistry(t *testing.T) {
	reg := NewRegistry()
	reg.Register(&MockTool{Name: "tool1"})
	reg.Register(&MockTool{Name: "tool2"})

	if len(reg.List()) != 2 {
		t.Errorf("expected 2 tools, got %d", len(reg.List()))
	}

	tool, ok := reg.Get("tool1")
	if !ok {
		t.Fatal("tool1 not found")
	}
	if tool.Metadata().Name != "tool1" {
		t.Errorf("expected tool1, got %s", tool.Metadata().Name)
	}

	matches := reg.Search("tool")
	if len(matches) != 2 {
		t.Errorf("expected 2 matches for 'tool', got %d", len(matches))
	}
}

func TestToMCP(t *testing.T) {
	mt := &MockTool{Name: "test-mcp"}
	mcp := ToMCP(mt)

	if mcp.Name != "test-mcp" {
		t.Errorf("expected name test-mcp, got %s", mcp.Name)
	}
	if mcp.Description != "A mock tool" {
		t.Errorf("expected description, got %s", mcp.Description)
	}
}

// MockFS for testing sys tools
type MockFS struct {
	Files map[string][]byte
}

func (m *MockFS) ReadFile(path string) ([]byte, error) {
	content, ok := m.Files[path]
	if !ok {
		return nil, fmt.Errorf("file not found")
	}
	return content, nil
}

func (m *MockFS) WriteFile(path string, content []byte) error {
	m.Files[path] = content
	return nil
}

func (m *MockFS) DeleteFile(path string) error {
	delete(m.Files, path)
	return nil
}

func (m *MockFS) ListFiles(path string) ([]string, error) {
	return []string{}, nil
}

func (m *MockFS) Edit(path string, oldStr, newStr string) error {
	return nil
}

func (m *MockFS) Batch(ops []sys.BatchOp) error {
	return nil
}

func TestReadFileTool(t *testing.T) {
	fs := &MockFS{Files: map[string][]byte{"test.txt": []byte("hello world")}}
	tool := NewReadFileTool(fs)

	args := json.RawMessage(`{"path": "test.txt"}`)
	res, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if res.Content != "hello world" {
		t.Errorf("expected 'hello world', got %s", res.Content)
	}
}

func TestWriteFileTool(t *testing.T) {
	fs := &MockFS{Files: make(map[string][]byte)}
	tool := NewWriteFileTool(fs)

	args := json.RawMessage(`{"path": "new.txt", "content": "vibe"}`)
	_, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if string(fs.Files["new.txt"]) != "vibe" {
		t.Errorf("expected 'vibe', got %s", string(fs.Files["new.txt"]))
	}
}
