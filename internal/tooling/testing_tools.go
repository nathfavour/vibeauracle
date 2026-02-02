package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// TesterTool autonomously runs tests for the project.
type TesterTool struct{}

func (t *TesterTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "tester",
		Description: "Autonomously detect and run tests for the current project. Supports Go, Node.js, Python, and Rust.",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer, RoleCoder},
		Complexity:  5,
		Permissions: []Permission{PermExecute, PermRead},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"path": {"type": "string", "description": "Specific path or package to test. Defaults to current directory."},
				"args": {"type": "array", "items": {"type": "string"}, "description": "Extra arguments to pass to the test runner (e.g. ['-v', '-run', 'TestName'])."}
			}
		}`),
	}
}

func (t *TesterTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Path string   `json:"path"`
		Args []string `json:"args"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	// 1. Detect project type and runner
	runner, cmdArgs, err := t.detectRunner(ctx, input.Path)
	if err != nil {
		return &ToolResult{Status: "error", Content: fmt.Sprintf("Failed to detect test runner: %v", err)}, nil
	}

	// 2. Append extra args
	if len(input.Args) > 0 {
		cmdArgs = append(cmdArgs, input.Args...)
	}

	ReportStatus("🧪", "testing", fmt.Sprintf("Running: %s %s", runner, strings.Join(cmdArgs, " ")))

	// 3. Execute
	cmd := exec.CommandContext(ctx, runner, cmdArgs...)
	out, err := cmd.CombinedOutput()

	status := "success"
	if err != nil {
		status = "failure"
	}

	return &ToolResult{
		Status:  status,
		Content: string(out),
		Error:   err,
	}, nil
}

func (t *TesterTool) detectRunner(ctx context.Context, path string) (string, []string, error) {
	// Look for go.mod
	if _, err := exec.LookPath("go"); err == nil {
		return "go", []string{"test", "./..."}, nil
	}

	// Look for package.json
	if _, err := exec.LookPath("npm"); err == nil {
		return "npm", []string{"test"}, nil
	}

	// Look for requirements.txt or pyproject.toml
	if _, err := exec.LookPath("pytest"); err == nil {
		return "pytest", []string{}, nil
	} else if _, err := exec.LookPath("python"); err == nil {
		return "python", []string{"-m", "unittest", "discover"}, nil
	}

	// Look for Cargo.toml
	if _, err := exec.LookPath("cargo"); err == nil {
		return "cargo", []string{"test"}, nil
	}

	return "", nil, fmt.Errorf("no supported test runner found")
}
