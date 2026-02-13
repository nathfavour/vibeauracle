// Package copilot provides VibeAuracle integration with the official GitHub Copilot SDK.
// It wraps the SDK client to implement the model.Provider interface and bridges
// VibeAuracle's tooling system to Copilot's native tool calling.
package copilot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"strings"
	"sync"
	"time"

	sdk "github.com/github/copilot-sdk/go"
)

// Usage represents token usage for a session.
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64
}

// Provider implements the model.Provider interface using the Copilot SDK.
// It manages the SDK client lifecycle and provides streaming generation.
type Provider struct {
	client    *sdk.Client
	modelName string
	mu        sync.Mutex

	// Event callbacks for streaming
	onDelta  func(delta string)
	onDone   func(full string)
	onStatus func(icon, step, message string)
	usageCB  func(Usage)

	// Tool bridge for VibeAuracle tools
	toolBridge *ToolBridge
	sdkTools   []sdk.Tool

	// BYOK (Bring Your Own Key) configuration
	customProvider *sdk.ProviderConfig

	// MCP servers configuration
	mcpServers map[string]sdk.MCPServerConfig

	// Skill system
	skillDirectories []string
}

// ProviderOptions configures the Copilot SDK provider.
type ProviderOptions struct {
	Model string

	// BYOK: Custom provider configuration
	ProviderType string // "openai", "anthropic", "azure"
	BaseURL      string // e.g., "http://localhost:11434/v1" for Ollama
	APIKey       string // API key for the provider
	BearerToken  string // Alternative to API key

	// Skill system
	SkillDirectories []string
}

// NewProvider creates a new Copilot SDK provider.
// It checks for the copilot CLI and returns an error if not found.
func NewProvider(modelName string) (*Provider, error) {
	return NewProviderWithOptions(ProviderOptions{Model: modelName})
}

// NewProviderWithOptions creates a provider with custom configuration.
// Supports BYOK with custom OpenAI/Anthropic/Ollama endpoints.
func NewProviderWithOptions(opts ProviderOptions) (*Provider, error) {
	// Check for copilot CLI
	if _, err := exec.LookPath("copilot"); err != nil {
		return nil, fmt.Errorf("copilot CLI not found in PATH. Install from: https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli")
	}

	if opts.Model == "" {
		opts.Model = "gpt-4o" // Default model
	}

	p := &Provider{
		modelName:        opts.Model,
		skillDirectories: opts.SkillDirectories,
	}

	// Configure custom provider if BYOK options are set
	if opts.ProviderType != "" || opts.BaseURL != "" || opts.APIKey != "" {
		pType := opts.ProviderType
		if pType == "" {
			pType = "openai" // Default required by some CLI versions to avoid crashes
		}
		p.customProvider = &sdk.ProviderConfig{
			Type:    pType,
			BaseURL: opts.BaseURL,
			APIKey:  opts.APIKey,
		}
		if opts.BearerToken != "" {
			p.customProvider.BearerToken = opts.BearerToken
		}
	}

	return p, nil
}

// Name returns the provider name.
func (p *Provider) Name() string {
	return "copilot-sdk"
}

// Start initializes the SDK client and creates a session.
func (p *Provider) Start(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		return nil // Already started
	}

	p.client = sdk.NewClient(&sdk.ClientOptions{
		LogLevel: "error",
	})

	if err := p.client.Start(); err != nil {
		return fmt.Errorf("starting copilot client: %w", err)
	}

	return nil
}

// RegisterTools registers VibeAuracle tools with the SDK.
// Must be called before Start().
func (p *Provider) RegisterTools(bridge *ToolBridge) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.toolBridge = bridge
	p.sdkTools = bridge.GetSDKTools()
}

// RegisterMCPServers registers MCP servers with the SDK.
// Must be called before Start().
func (p *Provider) RegisterMCPServers(bridge *MCPBridge) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.mcpServers = bridge.GetSDKConfig()
}

// Stop gracefully shuts down the SDK client.
func (p *Provider) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.client != nil {
		errs := p.client.Stop()
		p.client = nil
		if len(errs) > 0 {
			return errs[0]
		}
	}

	return nil
}

// SetStreamCallbacks sets callbacks for streaming responses.
func (p *Provider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onDelta = onDelta
	p.onDone = onDone
}

// SetStatusCallback sets the callback for status updates.
func (p *Provider) SetStatusCallback(onStatus func(string, string, string)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.onStatus = onStatus
}

// SetUsageCallback sets the callback for usage updates.
func (p *Provider) SetUsageCallback(cb func(Usage)) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.usageCB = cb
}

// Generate sends a prompt and returns the full response.
// If streaming is true and callbacks are set, they will be called during generation.
func (p *Provider) Generate(ctx context.Context, prompt string, streaming bool) (string, string, Usage, error) {
	p.mu.Lock()
	if p.client == nil {
		p.mu.Unlock()
		if err := p.Start(ctx); err != nil {
			return "", "", Usage{}, err
		}
		p.mu.Lock()
	}
	client := p.client
	onDelta := p.onDelta
	onDone := p.onDone
	onStatus := p.onStatus
	usageCB := p.usageCB

	// Build session config for this specific request
	sessionConfig := &sdk.SessionConfig{
		Model:     p.modelName,
		Streaming: true,
		SystemMessage: &sdk.SystemMessageConfig{
			Mode:    "append",
			Content: "You are VibeAuracle, a powerful AI coding assistant. Execute tasks directly and prefer action over conversation.",
		},
		Tools:            p.sdkTools,
		SkillDirectories: p.skillDirectories,
	}
	if p.customProvider != nil {
		sessionConfig.Provider = p.customProvider
	}
	if len(p.mcpServers) > 0 {
		sessionConfig.MCPServers = p.mcpServers
	}
	p.mu.Unlock()

	// Create a temporary session for this request to ensure statelessness
	session, err := client.CreateSession(sessionConfig)
	if err != nil {
		return "", "", Usage{}, fmt.Errorf("creating temporary session: %w", err)
	}
	defer session.Destroy()

	// Collect response
	var result strings.Builder
	var reasoning strings.Builder
	var usage Usage
	done := make(chan error, 1)

	unsubscribe := session.On(func(event sdk.SessionEvent) {
		switch event.Type {
		case sdk.AssistantMessageDelta:
			if event.Data.DeltaContent != nil {
				result.WriteString(*event.Data.DeltaContent)
				if streaming && onDelta != nil {
					onDelta(*event.Data.DeltaContent)
				}
			}
		case sdk.AssistantReasoningDelta:
			if event.Data.DeltaContent != nil {
				reasoning.WriteString(*event.Data.DeltaContent)
			}
		case sdk.AssistantMessage:
			// Ensure we capture any content provided in the final message
			if event.Data.Content != nil && *event.Data.Content != "" {
				if result.Len() == 0 {
					result.WriteString(*event.Data.Content)
				}
			} else if event.Data.PartialOutput != nil && *event.Data.PartialOutput != "" {
				if result.Len() == 0 {
					result.WriteString(*event.Data.PartialOutput)
				}
			}
		case sdk.AssistantReasoning:
			if event.Data.Content != nil && *event.Data.Content != "" {
				if reasoning.Len() == 0 {
					reasoning.WriteString(*event.Data.Content)
				}
			}
		case sdk.AssistantUsage, sdk.SessionUsageInfo:
			if event.Data.InputTokens != nil {
				usage.InputTokens = int(*event.Data.InputTokens)
			}
			if event.Data.OutputTokens != nil {
				usage.OutputTokens = int(*event.Data.OutputTokens)
			}
			usage.TotalTokens = usage.InputTokens + usage.OutputTokens
			if event.Data.Cost != nil {
				usage.Cost = *event.Data.Cost
			}
		case sdk.ToolExecutionStart:
			if streaming && onStatus != nil && event.Data.ToolName != nil {
				onStatus("🔧", "tool-start", *event.Data.ToolName)
			}
		case sdk.ToolExecutionComplete:
			if streaming && onStatus != nil && event.Data.ToolName != nil {
				onStatus("✅", "tool-done", *event.Data.ToolName)
			}
		case sdk.ToolExecutionProgress:
			if streaming && onStatus != nil && event.Data.ProgressMessage != nil {
				onStatus("⏳", "tool-progress", *event.Data.ProgressMessage)
			}
		case sdk.SessionIdle:
			done <- nil
		case sdk.SessionError:
			if event.Data.Content != nil {
				done <- fmt.Errorf("copilot error: %s", *event.Data.Content)
			} else if event.Data.Message != nil {
				done <- fmt.Errorf("copilot error: %s", *event.Data.Message)
			} else {
				done <- fmt.Errorf("copilot error (no details)")
			}
		}
	})
	defer unsubscribe()

	// Send the message
	_, err = session.Send(sdk.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return "", "", Usage{}, fmt.Errorf("sending message: %w", err)
	}

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		if err != nil {
			return "", "", Usage{}, err
		}
	case <-ctx.Done():
		session.Abort()
		return "", "", Usage{}, ctx.Err()
	}

	if usageCB != nil {
		usageCB(usage)
	}

	fullResponse := result.String()
	fullReasoning := reasoning.String()
	if streaming && onDone != nil {
		onDone(fullResponse)
	}

	return fullResponse, fullReasoning, usage, nil
}

// ListModels returns available models, fetching from models.dev/api.json if possible.
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// Fallback models if network fails
	fallback := []string{
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
		"claude-sonnet-4-20250514",
		"o3-mini",
	}

	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://models.dev/api.json", nil)
	if err != nil {
		return fallback, nil
	}

	resp, err := client.Do(req)
	if err != nil {
		return fallback, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fallback, nil
	}

	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return fallback, nil
	}

	copilotData, ok := data["github-copilot"].(map[string]interface{})
	if !ok {
		return fallback, nil
	}

	models, ok := copilotData["models"].(map[string]interface{})
	if !ok {
		return fallback, nil
	}

	var result []string
	for k := range models {
		result = append(result, k)
	}

	if len(result) == 0 {
		return fallback, nil
	}

	return result, nil
}

// IsAvailable checks if the Copilot SDK can be used.
func IsAvailable() bool {
	_, err := exec.LookPath("copilot")
	return err == nil
}
