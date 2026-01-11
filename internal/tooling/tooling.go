package tooling

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nathfavour/vibeauracle/sys"
)

// Tool represents a programmable interface that can be exposed to a model.
type Tool interface {
	Name() string
	Description() string
	Parameters() json.RawMessage // JSON Schema
	Permissions() []Permission
	Execute(ctx context.Context, args json.RawMessage) (interface{}, error)
}

// Permission represents a capability required by a tool.
type Permission string

const (
	PermRead      Permission = "read"
	PermWrite     Permission = "write"
	PermExecute   Permission = "execute"
	PermNetwork   Permission = "network"
	PermSensitive Permission = "sensitive" // Access to passwords, keys, etc.
)

// Registry manages the set of available tools.
type Registry struct {
	tools map[string]Tool
}

func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

func (r *Registry) Register(t Tool) {
	r.tools[t.Name()] = t
}

func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

func (r *Registry) List() []Tool {
	var list []Tool
	for _, t := range r.tools {
		list = append(list, t)
	}
	return list
}

// MCPTool matches the official Model Context Protocol tool definition.
type MCPTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"inputSchema"`
}

// ToMCP converts a tool to an MCP-compliant structure.
func ToMCP(t Tool) MCPTool {
	return MCPTool{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: t.Parameters(),
	}
}

// DefaultRegistry creates a registry populated with core system tools.
func DefaultRegistry(f sys.FS, m *sys.Monitor, guard *SecurityGuard) *Registry {
	r := NewRegistry()

	tools := []Tool{
		NewReadFileTool(f),
		NewWriteFileTool(f),
		NewListFilesTool(f),
		NewTraversalTool(f),
		&ShellExecTool{},
		NewSystemInfoTool(m),
		&FetchURLTool{},
	}

	for _, t := range tools {
		if guard != nil {
			r.Register(WrapWithSecurity(t, guard))
		} else {
			r.Register(t)
		}
	}

	return r
}

// GetPromptDefinitions returns a human-readable or machine-parsable definition
// of all tools to be injected into a model's prompt.
func (r *Registry) GetPromptDefinitions() string {
	var defs string
	for _, t := range r.tools {
		defs += fmt.Sprintf("- %s: %s\n", t.Name(), t.Description())
	}
	return defs
}

