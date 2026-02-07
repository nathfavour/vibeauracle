package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/nathfavour/vibeauracle/tooling"
)

type mockTool struct {
	name string
}

func (m *mockTool) Metadata() tooling.ToolMetadata {
	return tooling.ToolMetadata{Name: m.name}
}

func (m *mockTool) Execute(ctx context.Context, args json.RawMessage) (*tooling.ToolResult, error) {
	return &tooling.ToolResult{Content: "mcp-result"}, nil
}

func TestBridge(t *testing.T) {
	reg := tooling.NewRegistry()
	reg.Register(&mockTool{name: "mcp-tool"})
	
	bridge := NewBridge(reg)
	
	tools := bridge.ListTools()
	if len(tools) != 1 || tools[0].Name != "mcp-tool" {
		t.Errorf("expected 1 tool 'mcp-tool', got %v", tools)
	}
	
	res, err := bridge.Execute(context.Background(), "mcp-tool", nil)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	
	tr, ok := res.(*tooling.ToolResult)
	if !ok || tr.Content != "mcp-result" {
		t.Errorf("unexpected result: %v", res)
	}
}
