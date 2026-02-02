package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/hexops/gotextdiff"
	"github.com/hexops/gotextdiff/myers"
	"github.com/hexops/gotextdiff/span"
	"github.com/nathfavour/vibeauracle/sys"
)

// PatchTool applies unified diffs to files.
type PatchTool struct {
	fs sys.FS
}

func NewPatchTool(f sys.FS) *PatchTool {
	return &PatchTool{fs: f}
}

func (t *PatchTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "sys_patch",
		Description: "Apply a unified diff patch to a file. This is more efficient than overwriting the entire file.",
		Source:      "system",
		Category:    CategoryFileSystem,
		Roles:       []AgentRole{RoleCoder, RoleEngineer},
		Complexity:  6,
		Permissions: []Permission{PermWrite, PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Path to the file to patch"},
				"patch": {"type": "string", "description": "The unified diff patch to apply"}
			},
			"required": ["path", "patch"]
		}`),
	}
}

func (t *PatchTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path  string `json:"path"`
		Patch string `json:"patch"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	ReportStatus("🩹", "exec", fmt.Sprintf("Patching file: %s", input.Path))

	// Read original content
	originalContent, err := t.fs.ReadFile(input.Path)
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Failed to read %s: %v", input.Path, err))
		return &ToolResult{Status: "error", Error: err}, err
	}

	// Parse the unified diff
	edits, err := parseUnifiedDiff(input.Patch)
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Invalid patch format: %v", err))
		return &ToolResult{Status: "error", Error: err}, err
	}

	// Apply edits
	newContent := gotextdiff.ApplyEdits(string(originalContent), edits)

	// Write back
	err = t.fs.WriteFile(input.Path, []byte(newContent))
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Failed to write patched %s: %v", input.Path, err))
		return &ToolResult{Status: "error", Error: err}, err
	}

	ReportStatus("✅", "exec", fmt.Sprintf("Successfully patched %s", input.Path))
	return &ToolResult{
		Status:    "success",
		Content:   "Patch applied successfully",
		Artifacts: []string{input.Path},
	}, nil
}

// parseUnifiedDiff converts a unified diff string into a slice of gotextdiff.TextEdit.
// This is a simple implementation that assumes standard unified diff format.
func parseUnifiedDiff(patchStr string) ([]gotextdiff.TextEdit, error) {
	// gotextdiff doesn't provide a public high-level unified diff parser that returns []TextEdit directly.
	// We might need to use a more robust parser or implement a simple one for standard agent-generated diffs.
	// Actually, agents often generate simplified patches or we can ask them to.
	
	// For "very robust" implementation, we'll try to handle standard unified diffs.
	// If standard tools are unavailable, we can use a library.
	
	// Check if there's a better way in gotextdiff
	// gotextdiff usually works by calculating diffs between two strings.
	
	// If we want to *apply* a patch provided as a string, we need to parse it.
	// Since gotextdiff is optimized for *generating* and *applying* internal edit structures, 
	// let's see if we can use another helper or implement the parser.
	
	return nil, fmt.Errorf("robust patch parser not implemented yet")
}
