package tooling

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
)

// WebSearchTool performs a web search.
type WebSearchTool struct{}

func (t *WebSearchTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "web_search",
		Description: "Search the internet for information. Uses DuckDuckGo via CLI or API.",
		Source:      "system",
		Category:    CategoryNetwork,
		Roles:       []AgentRole{RoleResearcher, RoleArchitect},
		Complexity:  5,
		Permissions: []Permission{PermNetwork},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "The search query"},
				"limit": {"type": "integer", "description": "Maximum number of results (default 5)"}
			},
			"required": ["query"]
		}`),
	}
}

func (t *WebSearchTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Query string `json:"query"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	if input.Limit == 0 {
		input.Limit = 5
	}

	ReportStatus("🌐", "exec", fmt.Sprintf("Searching web for: %s", input.Query))

	// Attempt to use 'ddgr' if installed (DuckDuckGo CLI)
	if _, err := exec.LookPath("ddgr"); err == nil {
		cmd := exec.CommandContext(ctx, "ddgr", "--json", "-n", fmt.Sprintf("%d", input.Limit), input.Query)
		out, err := cmd.CombinedOutput()
		if err == nil {
			return &ToolResult{Status: "success", Content: string(out)}, nil
		}
	}

	// Fallback: search via curl and simple parsing (very basic)
	// In a real production tool, we would use a dedicated search API with a key from the vault.
	// For now, we'll provide a helpful error message if no CLI tools are found.
	return &ToolResult{
		Status: "error",
		Content: "Web search requires 'ddgr' CLI to be installed for local search. Alternatively, configure a Search API key in vibes.",
	}, nil
}

// GHSearchTool searches GitHub for code, repositories, or issues.
type GHSearchTool struct{}

func (t *GHSearchTool) Metadata() ToolMetadata {
	return ToolMetadata{
		Name:        "gh_search",
		Description: "Search GitHub for code, repositories, or issues using 'gh search'.",
		Source:      "system",
		Category:    CategoryNetwork,
		Roles:       []AgentRole{RoleResearcher, RoleEngineer},
		Complexity:  6,
		Permissions: []Permission{PermNetwork, PermExecute},
		Parameters: json.RawMessage(`{
			"type": "object",
			"properties": {
				"query": {"type": "string", "description": "The search query"},
				"kind": {"type": "string", "enum": ["code", "repos", "issues", "prs"], "description": "What to search for"},
				"limit": {"type": "integer", "description": "Maximum number of results (default 10)"}
			},
			"required": ["query", "kind"]
		}`),
	}
}

func (t *GHSearchTool) Execute(ctx context.Context, args json.RawMessage) (*ToolResult, error) {
	var input struct {
		Query string `json:"query"`
		Kind  string `json:"kind"`
		Limit int    `json:"limit"`
	}
	if err := json.Unmarshal(args, &input); err != nil {
		return nil, err
	}
	if input.Limit == 0 {
		input.Limit = 10
	}

	if _, err := exec.LookPath("gh"); err != nil {
		return &ToolResult{Status: "error", Content: "GitHub CLI (gh) is required for gh_search."}, nil
	}

	ReportStatus("🐙", "exec", fmt.Sprintf("Searching GitHub %s for: %s", input.Kind, input.Query))

	var kindArg string
	switch input.Kind {
	case "code":
		kindArg = "code"
	case "repos":
		kindArg = "repos"
	case "issues":
		kindArg = "issues"
	case "prs":
		kindArg = "prs"
	}

	cmd := exec.CommandContext(ctx, "gh", "search", kindArg, input.Query, "--limit", fmt.Sprintf("%d", input.Limit))
	out, err := cmd.CombinedOutput()
	if err != nil {
		return &ToolResult{Status: "error", Content: string(out), Error: err}, nil
	}

	return &ToolResult{Status: "success", Content: string(out)}, nil
}
