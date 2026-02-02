package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"go.lsp.dev/protocol"
	"go.lsp.dev/uri"
)

// LSPDefinitionTool provides "Go to Definition" capability.
type LSPDefinitionTool struct {
	mgr *LSPManager
}

func NewLSPDefinitionTool(mgr *LSPManager) *LSPDefinitionTool {
	return &LSPDefinitionTool{mgr: mgr}
}

func (t *LSPDefinitionTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "lsp_goto_definition",
		Description: "Find the definition of a symbol at a specific position in a file.",
		Source:      "system",
		Category:    CategoryCoding,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  4,
		Permissions: []Permission{PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Path to the file"},
				"line": {"type": "integer", "description": "Line number (0-indexed)"},
				"character": {"type": "integer", "description": "Character position (0-indexed)"}
			},
			"required": ["path", "line", "character"]
		}`),
	}
}

func (t *LSPDefinitionTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path      string `json:"path"`
		Line      int    `json:"line"`
		Character int    `json:"character"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	lang := t.mgr.MapExtensionToLanguage(filepath.Ext(input.Path))
	if lang == "" {
		return &ToolResult{Status: "error", Content: "Unsupported file type for LSP"}, nil
	}

	session, err := t.mgr.GetSession(ctx, lang)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}

	absPath, _ := filepath.Abs(input.Path)
	params := &protocol.DefinitionParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri.File(absPath),
			},
			Position: protocol.Position{
				Line:      uint32(input.Line),
				Character: uint32(input.Character),
			},
		},
	}

	var result []protocol.Location
	_, err = session.conn.Call(ctx, protocol.MethodTextDocumentDefinition, params, &result)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}

	if len(result) == 0 {
		return &ToolResult{Status: "success", Content: "No definition found"}, nil
	}

	loc := result[0]
	return &ToolResult{
		Status:  "success",
		Content: fmt.Sprintf("Definition found at %s:%d:%d", loc.URI.Filename(), loc.Range.Start.Line, loc.Range.Start.Character),
		Data:    result,
	},
il
}

// LSPReferencesTool provides "Find References" capability.
type LSPReferencesTool struct {
	mgr *LSPManager
}

func NewLSPReferencesTool(mgr *LSPManager) *LSPReferencesTool {
	return &LSPReferencesTool{mgr: mgr}
}

func (t *LSPReferencesTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "lsp_find_references",
		Description: "Find all references to a symbol at a specific position in a file.",
		Source:      "system",
		Category:    CategoryCoding,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  5,
		Permissions: []Permission{PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Path to the file"},
				"line": {"type": "integer", "description": "Line number (0-indexed)"},
				"character": {"type": "integer", "description": "Character position (0-indexed)"}
			},
			"required": ["path", "line", "character"]
		}`),
	}
}

func (t *LSPReferencesTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path      string `json:"path"`
		Line      int    `json:"line"`
		Character int    `json:"character"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	lang := t.mgr.MapExtensionToLanguage(filepath.Ext(input.Path))
	if lang == "" {
		return &ToolResult{Status: "error", Content: "Unsupported file type for LSP"}, nil
	}

	session, err := t.mgr.GetSession(ctx, lang)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}

	absPath, _ := filepath.Abs(input.Path)
	params := &protocol.ReferenceParams{
		TextDocumentPositionParams: protocol.TextDocumentPositionParams{
			TextDocument: protocol.TextDocumentIdentifier{
				URI: uri.File(absPath),
			},
			Position: protocol.Position{
				Line:      uint32(input.Line),
				Character: uint32(input.Character),
			},
		},
		Context: protocol.ReferenceContext{
			IncludeDeclaration: true,
		},
	}

	var result []protocol.Location
	_, err = session.conn.Call(ctx, protocol.MethodTextDocumentReferences, params, &result)
	if err != nil {
		return &ToolResult{Status: "error", Error: err}, err
	}

	if len(result) == 0 {
		return &ToolResult{Status: "success", Content: "No references found"}, nil
	}

	var sb string
	for _, loc := range result {
		sb += fmt.Sprintf("- %s:%d:%d\n", loc.URI.Filename(), loc.Range.Start.Line, loc.Range.Start.Character)
	}

	return &ToolResult{
		Status:  "success",
		Content: sb,
		Data:    result,
	},
il
}
