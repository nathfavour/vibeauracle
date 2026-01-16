package copilot

import (
	"context"
	"encoding/json"

	sdk "github.com/github/copilot-sdk/go"
)

// ToolBridge converts VibeAuracle tools to Copilot SDK tools.
type ToolBridge struct {
	tools []sdk.Tool
}

// VibeToolDefinition represents a simplified tool definition from VibeAuracle.
type VibeToolDefinition struct {
	Name        string
	Description string
	Parameters  json.RawMessage
	Execute     func(ctx context.Context, args json.RawMessage) (string, error)
}

// NewToolBridge creates a bridge that converts VibeAuracle tools to SDK format.
func NewToolBridge() *ToolBridge {
	return &ToolBridge{
		tools: make([]sdk.Tool, 0),
	}
}

// AddTool registers a VibeAuracle tool for use with Copilot.
func (b *ToolBridge) AddTool(def VibeToolDefinition) {
	// Convert Parameters from json.RawMessage to map[string]interface{}
	var params map[string]interface{}
	if len(def.Parameters) > 0 {
		json.Unmarshal(def.Parameters, &params)
	}
	if params == nil {
		params = map[string]interface{}{
			"type":       "object",
			"properties": map[string]interface{}{},
		}
	}

	// Create SDK tool with handler that routes to VibeAuracle's Execute
	tool := sdk.Tool{
		Name:        def.Name,
		Description: def.Description,
		Parameters:  params,
		Handler: func(inv sdk.ToolInvocation) (sdk.ToolResult, error) {
			// Convert arguments back to JSON for VibeAuracle tools
			argsJSON, err := json.Marshal(inv.Arguments)
			if err != nil {
				return sdk.ToolResult{
					TextResultForLLM: "Failed to marshal arguments",
					ResultType:       "error",
					Error:            err.Error(),
				}, nil
			}

			// Execute the VibeAuracle tool
			result, err := def.Execute(context.Background(), argsJSON)
			if err != nil {
				return sdk.ToolResult{
					TextResultForLLM: err.Error(),
					ResultType:       "error",
					Error:            err.Error(),
				}, nil
			}

			return sdk.ToolResult{
				TextResultForLLM: result,
				ResultType:       "success",
			}, nil
		},
	}

	b.tools = append(b.tools, tool)
}

// GetSDKTools returns all registered tools in SDK format.
func (b *ToolBridge) GetSDKTools() []sdk.Tool {
	return b.tools
}

// Clear removes all registered tools.
func (b *ToolBridge) Clear() {
	b.tools = make([]sdk.Tool, 0)
}
