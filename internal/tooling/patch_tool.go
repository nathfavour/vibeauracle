package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

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
		Description: "Apply a unified diff patch to a file. This is more efficient than overwriting the entire file. Use standard unified diff format (---, +++, @@, context).",
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

	// Apply the patch
	newContent, err := applyPatch(string(originalContent), input.Patch)
	if err != nil {
		ReportStatus("❌", "exec", fmt.Sprintf("Failed to apply patch to %s: %v", input.Path, err))
		return &ToolResult{Status: "error", Error: err}, err
	}

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

// applyPatch is a robust implementation of unified diff application.
func applyPatch(original, patch string) (string, error) {
	lines := strings.Split(original, "\n")
	patchLines := strings.Split(patch, "\n")

	var resultLines []string
	currentLine := 0

	hunkHeader := regexp.MustCompile(`^@@ -(\d+),?(\d*) \+(\d+),?(\d*) @@`)

	i := 0
	// Skip header lines (---, +++)
	for i < len(patchLines) && (strings.HasPrefix(patchLines[i], "---") || strings.HasPrefix(patchLines[i], "+++")) {
		i++
	}

	for i < len(patchLines) {
		line := patchLines[i]
		if line == "" {
			i++
			continue
		}

		match := hunkHeader.FindStringSubmatch(line)
		if match == nil {
			// If we are not at a hunk header, and not at the end, it might be junk or malformed.
			// Robustly skip or error. For now, we skip unexpected lines between hunks.
			i++
			continue
		}

		// Parse hunk header
		oldStart, _ := strconv.Atoi(match[1])
		// oldLen, _ := strconv.Atoi(match[2]) // not strictly needed if we just follow the context

		// Adjust oldStart to 0-based
		oldStartIdx := oldStart - 1
		if oldStartIdx < 0 {
			oldStartIdx = 0
		}

		// Copy lines from original up to the start of this hunk
		for currentLine < oldStartIdx && currentLine < len(lines) {
			resultLines = append(resultLines, lines[currentLine])
			currentLine++
		}

		i++ // Move to hunk content
		for i < len(patchLines) {
			pLine := patchLines[i]
			if strings.HasPrefix(pLine, "@@") || strings.HasPrefix(pLine, "---") || strings.HasPrefix(pLine, "+++") {
				break // Start of next hunk or header
			}

			if len(pLine) == 0 {
				// Empty line in unified diff usually means a space (context) was intended but trimmed
				pLine = " "
			}

			indicator := pLine[0]
			content := pLine[1:]

			switch indicator {
			case ' ':
				// Context line - verify it matches original
				if currentLine < len(lines) && lines[currentLine] == content {
					resultLines = append(resultLines, lines[currentLine])
					currentLine++
				} else if currentLine < len(lines) {
					// Fuzz logic could go here, but for "robustness" we require exact context matches
					// to avoid applying patches in the wrong place.
					return "", fmt.Errorf("context mismatch at line %d: expected %q, got %q", currentLine+1, content, lines[currentLine])
				}
			case '-':
				// Removal - verify it matches original
				if currentLine < len(lines) && lines[currentLine] == content {
					currentLine++
				} else if currentLine < len(lines) {
					return "", fmt.Errorf("removal mismatch at line %d: expected %q, got %q", currentLine+1, content, lines[currentLine])
				}
			case '+':
				// Addition
				resultLines = append(resultLines, content)
			}
			i++
		}
	}

	// Copy remaining lines from original
	for currentLine < len(lines) {
		resultLines = append(resultLines, lines[currentLine])
		currentLine++
	}

	// Handle trailing newline if original had one
	res := strings.Join(resultLines, "\n")
	if strings.HasSuffix(original, "\n") && !strings.HasSuffix(res, "\n") {
		res += "\n"
	}

	return res, nil
}
