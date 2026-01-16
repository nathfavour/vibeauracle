package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// SCMStatusTool provides git status information.
type SCMStatusTool struct{}

func (t *SCMStatusTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "scm_status",
		Description: "Show the working tree status (git status).",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer, RoleCoder},
		Complexity:  2,
		Permissions: []Permission{PermRead},
		Parameters:  json.RawMessage(`{"type": "object"}`),
	}
}

func (t *SCMStatusTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "git", "status", "--short")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}
	return &ToolResult{Status: "success", Content: string(out)}, nil
}

// SCMCommitTool handles committing changes, optionally using 'autocommitter'.
type SCMCommitTool struct{}

func (t *SCMCommitTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "scm_commit",
		Description: "Commit staged changes. Intelligently uses 'autocommitter' if available.",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer, RoleCoder},
		Complexity:  4,
		Permissions: []Permission{PermExecute, PermWrite},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"message": {"type": "string", "description": "Commit message. Optional if autocommitter is used."},
				"all": {"type": "boolean", "description": "Whether to stage all changes before committing (git commit -a)."}
			}
		}`),
	}
}

func (t *SCMCommitTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Message string `json:"message"`
		All     bool   `json:"all"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	// Try autocommitter first if no message is provided or if we want smart commits
	if _, err := exec.LookPath("autocommitter"); err == nil {
		ReportStatus("🤖", "scm", "Using autocommitter for smart commit...")
		cmd := exec.CommandContext(ctx, "autocommitter")
		out, err := cmd.CombinedOutput()
		if err == nil {
			return &ToolResult{Status: "success", Content: string(out)}, nil
		}
		// If autocommitter fails, we might still want to fallback to normal commit if message is provided
		if input.Message == "" {
			return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
		}
	}

	if input.Message == "" {
		return &ToolResult{Status: "error", Content: "Commit message is required when autocommitter is not available."}, nil
	}

	argsList := []string{"commit"}
	if input.All {
		argsList = append(argsList, "-a")
	}
	argsList = append(argsList, "-m", input.Message)

	cmd := exec.CommandContext(ctx, "git", argsList...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}

	return &ToolResult{Status: "success", Content: string(out)}, nil
}

// GitHubPRTool manages Pull Requests via the 'gh' CLI.
type GitHubPRTool struct{}

func (t *GitHubPRTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "gh_pr_manage",
		Description: "Manage GitHub Pull Requests using the 'gh' CLI.",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer, RoleArchitect},
		Complexity:  6,
		Permissions: []Permission{PermNetwork, PermExecute},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"action": {"type": "string", "enum": ["create", "view", "list", "status", "merge", "diff", "comments"], "description": "The PR action to perform"},
				"title": {"type": "string", "description": "Title of the PR (for create)"},
				"body": {"type": "string", "description": "Body of the PR (for create)"},
				"number": {"type": "integer", "description": "PR number (for view/merge)"},
				"base": {"type": "string", "description": "Base branch (for create)"}
			},
			"required": ["action"]
		}`),
	}
}

func (t *GitHubPRTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Action string `json:"action"`
		Title  string `json:"title"`
		Body   string `json:"body"`
		Number int    `json:"number"`
		Base   string `json:"base"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	var cmdArgs []string
	switch input.Action {
	case "create":
		cmdArgs = []string{"pr", "create", "-t", input.Title, "-b", input.Body}
		if input.Base != "" {
			cmdArgs = append(cmdArgs, "-B", input.Base)
		}
	case "view":
		if input.Number > 0 {
			cmdArgs = []string{"pr", "view", fmt.Sprintf("%d", input.Number)}
		} else {
			cmdArgs = []string{"pr", "view"}
		}
	case "list":
		cmdArgs = []string{"pr", "list"}
	case "status":
		cmdArgs = []string{"pr", "status"}
	case "merge":
		if input.Number > 0 {
			cmdArgs = []string{"pr", "merge", fmt.Sprintf("%d", input.Number), "--merge"}
		} else {
			cmdArgs = []string{"pr", "merge", "--merge"}
		}
	case "diff":
		if input.Number > 0 {
			cmdArgs = []string{"pr", "diff", fmt.Sprintf("%d", input.Number)}
		} else {
			cmdArgs = []string{"pr", "diff"}
		}
	case "comments":
		if input.Number > 0 {
			cmdArgs = []string{"pr", "view", fmt.Sprintf("%d", input.Number), "--json", "comments", "--template", "{{range .comments}}{{.author.login}}: {{.body}}\n---\n{{end}}"}
		} else {
			cmdArgs = []string{"pr", "view", "--json", "comments", "--template", "{{range .comments}}{{.author.login}}: {{.body}}\n---\n{{end}}"}
		}
	default:
		return nil, fmt.Errorf("unsupported action: %s", input.Action)
	}

	ReportStatus("🐙", "gh", fmt.Sprintf("Running gh %s", strings.Join(cmdArgs, " ")))
	cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}

	return &ToolResult{Status: "success", Content: string(out)}, nil
}

// SCMAddTool stages changes.
type SCMAddTool struct{}

func (t *SCMAddTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "scm_add",
		Description: "Stage changes for commit (git add).",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer, RoleCoder},
		Complexity:  2,
		Permissions: []Permission{PermWrite},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"paths": {"type": "array", "items": {"type": "string"}, "description": "List of paths to stage. Use ['.'] for all."}
			},
			"required": ["paths"]
		}`),
	}
}

func (t *SCMAddTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Paths []string `json:"paths"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	argsList := append([]string{"add"}, input.Paths...)
	cmd := exec.CommandContext(ctx, "git", argsList...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}

	return &ToolResult{Status: "success", Content: "Changes staged successfully."}, nil
}

// GitHubRemoteTaskTool triggers a remote agent task via the 'gh' CLI.
type GitHubRemoteTaskTool struct{}

func (t *GitHubRemoteTaskTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "gh_remote_task",
		Description: "Trigger a remote agent task on GitHub using 'gh agent-task create'.",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleArchitect, RoleEngineer},
		Complexity:  7,
		Permissions: []Permission{PermNetwork, PermExecute},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"description": {"type": "string", "description": "High-level description of the task for the remote agent"},
				"base": {"type": "string", "description": "Base branch for the remote task"},
				"follow": {"type": "boolean", "description": "Whether to follow and stream remote logs"}
			},
			"required": ["description"]
		}`),
	}
}

func (t *GitHubRemoteTaskTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Description string `json:"description"`
		Base        string `json:"base"`
		Follow      bool   `json:"follow"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}

	cmdArgs := []string{"agent-task", "create", input.Description}
	if input.Base != "" {
		cmdArgs = append(cmdArgs, "--base", input.Base)
	}
	if input.Follow {
		cmdArgs = append(cmdArgs, "--follow")
	}

	ReportStatus("🚀", "gh-remote", fmt.Sprintf("Launching remote task: %s", input.Description))
	cmd := exec.CommandContext(ctx, "gh", cmdArgs...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}

	return &ToolResult{
		Status:  "success",
		Content: string(out),
		Meta:    map[string]interface{}{"description": input.Description},
	}, nil
}

// GitHubExtensionTool lists installed gh extensions.
type GitHubExtensionTool struct{}

func (t *GitHubExtensionTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "gh_extensions",
		Description: "List installed GitHub CLI (gh) extensions.",
		Source:      "system",
		Category:    CategoryDevOps,
		Roles:       []AgentRole{RoleEngineer},
		Complexity:  3,
		Permissions: []Permission{PermRead, PermExecute},
		Parameters:  json.RawMessage(`{"type": "object"}`),
	}
}

func (t *GitHubExtensionTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	cmd := exec.CommandContext(ctx, "gh", "extension", "list")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}
	return &ToolResult{Status: "success", Content: string(out)}, nil
}
