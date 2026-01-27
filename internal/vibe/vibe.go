package vibe

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/nathfavour/vibeauracle/internal/tooling"
)

type Vibe struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Repo        string            `json:"repo"`
	Version     string            `json:"version"`
	Description string            `json:"description"`
	Protocol    string            `json:"protocol"`
	Endpoint    string            `json:"endpoint"`
	Command     string            `json:"command"`
	UpdateCmd   string            `json:"update_cmd"`
	Inbuilt     bool              `json:"inbuilt"`
	ToolSet     []tooling.MCPTool `json:"tool_set"`
}

type VibeProvider struct {
	vibe *Vibe
}

func NewVibeProvider(v *Vibe) *VibeProvider {
	return &VibeProvider{vibe: v}
}

func (vp *VibeProvider) Name() string {
	return vp.vibe.Name
}

func (vp *VibeProvider) Provide(ctx context.Context) ([]tooling.Tool, error) {
	var tools []tooling.Tool
	for _, t := range vp.vibe.ToolSet {
		tools = append(tools, &vibeTool{
			vibe:     vp.vibe,
			metadata: t,
		})
	}
	return tools, nil
}

type vibeTool struct {
	vibe     *Vibe
	metadata tooling.MCPTool
}

func (vt *vibeTool) Metadata() tooling.ToolMetadata {
	return tooling.ToolMetadata{
		Name:        vt.metadata.Name,
		Description: vt.metadata.Description,
		Parameters:  vt.metadata.InputSchema,
		Source:      vt.vibe.Name,
		Category:    tooling.CategoryDevOps,
	}
}

func (vt *vibeTool) Execute(ctx context.Context, args json.RawMessage) (*tooling.ToolResult, error) {
	if vt.vibe.Protocol == "stdio" {
		cmd := exec.CommandContext(ctx, vt.vibe.Command, "execute", vt.metadata.Name, string(args))
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nil, fmt.Errorf("executing vibe tool %s: %w (output: %s)", vt.metadata.Name, err, string(out))
		}
		
		var result tooling.ToolResult
		if err := json.Unmarshal(out, &result); err != nil {
			return &tooling.ToolResult{
				Content: string(out),
				Status:  "success",
			}, nil
		}
		return &result, nil
	}
	
	return nil, fmt.Errorf("protocol %s not yet implemented", vt.vibe.Protocol)
}
