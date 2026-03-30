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
		Description: "Apply a single unified diff to an existing file. Use exact context, standard diff headers, and no prose.",
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

	hunkHeader := regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)
	seenHunk := false

	i := 0
	// Skip header lines (---, +++)
	for i < len(patchLines) && (strings.HasPrefix(patchLines[i], "---") || strings.HasPrefix(patchLines[i], "+++")) {
		i++
	}

	for i < len(patchLines) {
		line := strings.TrimRight(patchLines[i], "\r")
		if line == "" {
			i++
			continue
		}

		if strings.HasPrefix(line, "diff --git ") ||
			strings.HasPrefix(line, "index ") ||
			strings.HasPrefix(line, "new file mode ") ||
			strings.HasPrefix(line, "deleted file mode ") ||
			strings.HasPrefix(line, "similarity index ") ||
			strings.HasPrefix(line, "rename from ") ||
			strings.HasPrefix(line, "rename to ") {
			i++
			continue
		}

		match := hunkHeader.FindStringSubmatch(line)
		if match == nil {
			return "", fmt.Errorf("malformed patch: expected hunk header, got %q", line)
		}

		seenHunk = true

		// Parse hunk header
		oldStart, _ := strconv.Atoi(match[1])
		oldLen := 0
		if match[2] != "" {
			oldLen, _ = strconv.Atoi(match[2])
		}
		newLen := 0
		if match[4] != "" {
			newLen, _ = strconv.Atoi(match[4])
		}

		// Adjust oldStart to 0-based
		oldStartIdx := oldStart - 1
		if oldStartIdx < 0 {
			oldStartIdx = 0
		}

		if currentLine > oldStartIdx {
			return "", fmt.Errorf("patch hunks out of order: current line %d already past hunk start %d", currentLine+1, oldStartIdx+1)
		}

		// Copy lines from original up to the start of this hunk
		for currentLine < oldStartIdx && currentLine < len(lines) {
			resultLines = append(resultLines, lines[currentLine])
			currentLine++
		}

		i++ // Move to hunk content
		consumedOld := 0
		producedNew := 0
		for i < len(patchLines) {
			pLine := strings.TrimRight(patchLines[i], "\r")
			if strings.HasPrefix(pLine, "@@") || strings.HasPrefix(pLine, "---") || strings.HasPrefix(pLine, "+++") {
				break // Start of next hunk or header
			}

			if pLine == "" {
				return "", fmt.Errorf("malformed patch: empty line inside hunk at patch line %d", i+1)
			}

			indicator := pLine[0]
			content := pLine[1:]

			switch indicator {
			case ' ':
				// Context line - verify it matches original
				if currentLine < len(lines) && lines[currentLine] == content {
					resultLines = append(resultLines, lines[currentLine])
					currentLine++
					consumedOld++
					producedNew++
				} else if currentLine < len(lines) {
					return "", fmt.Errorf("context mismatch at line %d: expected %q, got %q", currentLine+1, content, lines[currentLine])
				} else {
					return "", fmt.Errorf("context mismatch at line %d: expected %q, got EOF", currentLine+1, content)
				}
			case '-':
				// Removal - verify it matches original
				if currentLine < len(lines) && lines[currentLine] == content {
					currentLine++
					consumedOld++
				} else if currentLine < len(lines) {
					return "", fmt.Errorf("removal mismatch at line %d: expected %q, got %q", currentLine+1, content, lines[currentLine])
				} else {
					return "", fmt.Errorf("removal mismatch at line %d: expected %q, got EOF", currentLine+1, content)
				}
			case '+':
				// Addition
				resultLines = append(resultLines, content)
				producedNew++
			default:
				return "", fmt.Errorf("malformed patch line at %d: %q", i+1, pLine)
			}
			i++
		}

		if oldLen > 0 && consumedOld != oldLen {
			return "", fmt.Errorf("old line count mismatch in hunk starting at %d: expected %d, got %d", oldStart, oldLen, consumedOld)
		}
		if newLen > 0 && producedNew != newLen {
			return "", fmt.Errorf("new line count mismatch in hunk starting at %d: expected %d, got %d", oldStart, newLen, producedNew)
		}
	}

	if !seenHunk {
		return "", fmt.Errorf("malformed patch: no hunks found")
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
