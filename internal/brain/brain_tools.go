package brain

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/nathfavour/vibeauracle/copilot"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/prompt"
	"github.com/nathfavour/vibeauracle/tooling"
)

// registerToolsWithCopilot bridges VibeAuracle tools to the Copilot SDK.
func (b *Brain) registerToolsWithCopilot() {
	bridge := copilot.NewToolBridge()

	// Get core tools from the registry
	for _, t := range b.tools.List() {
		tool := t
		if false {
			continue
		}
		meta := t.Metadata()
		bridge.AddTool(copilot.VibeToolDefinition{
			Name:        meta.Name,
			Description: meta.Description,
			Parameters:  meta.Parameters,
			Execute: func(ctx context.Context, args json.RawMessage) (string, error) {
				result, err := tool.Execute(ctx, args)
				if err != nil {
					return "", err
				}
				return result.Content, nil
			},
		})
	}

	b.copilotProvider.RegisterTools(bridge)
}

func (b *Brain) executeToolCalls(ctx context.Context, input string, intent prompt.Intent) (bool, string, error, error) {
	var results []string
	var lastErr error
	var interventionErr error
	executed := false
	remaining := input

	// Find and execute ALL tool calls in the response
	for {
		start := strings.Index(remaining, "```json")
		if start == -1 {
			break
		}

		contentStart := start + 7
		blockContent := remaining[contentStart:]

		end := strings.Index(blockContent, "```")
		if end == -1 {
			break
		}

		jsonStr := strings.TrimSpace(blockContent[:end])
		remaining = blockContent[end+3:] // Move past this block

		// Attempt to parse tool call
		var call struct {
			Tool       string          `json:"tool"`
			Args       json.RawMessage `json:"parameters"`
			AltArgs    json.RawMessage `json:"args"`
			AltParams  json.RawMessage `json:"arguments"`
			Patch      json.RawMessage `json:"patch"`
			Content    json.RawMessage `json:"content"`
		}
		if err := json.Unmarshal([]byte(jsonStr), &call); err != nil {
			continue // Not a valid tool call, skip
		}

		if call.Tool == "" {
			continue
		}

		if len(call.Args) == 0 {
			switch {
			case len(call.AltArgs) > 0:
				call.Args = call.AltArgs
			case len(call.AltParams) > 0:
				call.Args = call.AltParams
			case len(call.Patch) > 0:
				call.Args = call.Patch
			case len(call.Content) > 0:
				call.Args = call.Content
			}
		}

		// Security: Block tool execution if intent is CHAT or ASK (unless specifically authorized).
		// This prevents the model from "hallucinating" tool calls during normal chat or Q&A.
		if intent == prompt.IntentChat || intent == prompt.IntentAsk {
			tooling.ReportStatus("🛡️", "security", fmt.Sprintf("Blocked tool call '%s' in %s mode", call.Tool, intent))
			results = append(results, fmt.Sprintf("Error: tool execution is disabled in %s mode. Please use '/do' or 'implement:' if you want me to take action.", intent))
			executed = true // Mark as executed so the loop can handle the "result"
			continue
		}

		// Found a tool call!
		executed = true

		// Detailed status reporting for specific tools
		extra := ""
		switch call.Tool {
		case "shell_exec":
			var args struct{ Command string `json:"command"` }
			if err := json.Unmarshal(call.Args, &args); err == nil {
				extra = "cmd:" + args.Command
			}
		case "sys_write_file":
			var args struct {
				Path    string `json:"path"`
				Content string `json:"content"`
			}
			if err := json.Unmarshal(call.Args, &args); err == nil {
				extra = "file:" + args.Path + "\n" + args.Content
			}
		case "sys_patch":
			var args struct {
				Path  string `json:"path"`
				Patch string `json:"patch"`
			}
			if err := json.Unmarshal(call.Args, &args); err == nil {
				extra = "patch:" + args.Path + "\n" + args.Patch
			}
		}

		if extra != "" {
			tooling.ReportStatus("🔧", "exec", fmt.Sprintf("Executing %s", call.Tool), extra)
		} else {
			tooling.ReportStatus("🔧", "tool", fmt.Sprintf("Executing: %s", call.Tool))
		}
		t, found := b.tools.Get(call.Tool)
		if !found {
			lastErr = fmt.Errorf("tool '%s' not found", call.Tool)
			doctor.Send("brain", "error", "Tool not found", map[string]any{"tool": call.Tool})
			results = append(results, fmt.Sprintf("Error: tool '%s' not found", call.Tool))
			continue
		}

		res, err := t.Execute(ctx, call.Args)
		if err != nil {
			// Check for intervention error
			if strings.Contains(err.Error(), "intervention required") {
				interventionErr = err
				doctor.Send("brain", "intervention", "Intervention required", map[string]any{"tool": call.Tool})
				break // Stop processing, need user input
			}
			lastErr = err
			doctor.Send("brain", "error", "Tool execution failed", map[string]any{"tool": call.Tool, "error": err.Error()})
			results = append(results, fmt.Sprintf("Error executing %s: %v", call.Tool, err))
			continue
		}

		results = append(results, fmt.Sprintf("[%s]: %s", call.Tool, res.Content))
	}

	if interventionErr != nil {
		return executed, strings.Join(results, "\n"), interventionErr, nil
	}

	return executed, strings.Join(results, "\n"), nil, lastErr
}

// ExecuteTool allows direct, non-agentic execution of a registered tool.
func (b *Brain) ExecuteTool(ctx context.Context, name string, args json.RawMessage) (*tooling.ToolResult, error) {
	t, found := b.tools.Get(name)
	if !found {
		return nil, fmt.Errorf("tool '%s' not found", name)
	}
	return t.Execute(ctx, args)
}

