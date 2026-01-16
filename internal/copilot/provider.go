// Package copilot provides VibeAuracle integration with the official GitHub Copilot SDK.
// It wraps the SDK client to implement the model.Provider interface and bridges
// VibeAuracle's tooling system to Copilot's native tool calling.
package copilot

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	sdk "github.com/github/copilot-sdk/go"
)

// Provider implements the model.Provider interface using the Copilot SDK.
// It manages the SDK client lifecycle and provides streaming generation.
type Provider struct {
	client    *sdk.Client
	session   *sdk.Session
	modelName string
	mu        sync.Mutex

	// Event callbacks for streaming
	onDelta func(delta string)
	onDone  func(full string)

	// Tool bridge for VibeAuracle tools
	toolBridge *ToolBridge
	sdkTools   []sdk.Tool
}

// NewProvider creates a new Copilot SDK provider.
// It checks for the copilot CLI and returns an error if not found.
func NewProvider(modelName string) (*Provider, error) {
	// Check for copilot CLI
	if _, err := exec.LookPath("copilot"); err != nil {
		return nil, fmt.Errorf("copilot CLI not found in PATH. Install from: https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli")
	}

	if modelName == "" {
		modelName = "gpt-4o" // Default model
	}

	return &Provider{
		modelName: modelName,
	}, nil
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

	session, err := p.client.CreateSession(&sdk.SessionConfig{
		Model:     p.modelName,
		Streaming: true,
		SystemMessage: &sdk.SystemMessageConfig{
			Mode:    "append",
			Content: "You are VibeAuracle, a powerful AI coding assistant. Execute tasks directly and prefer action over conversation.",
		},
	})
	if err != nil {
		p.client.Stop()
		p.client = nil
		return fmt.Errorf("creating session: %w", err)
	}

	p.session = session
	return nil
}

// Stop gracefully shuts down the SDK client.
func (p *Provider) Stop() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.session != nil {
		p.session.Destroy()
		p.session = nil
	}

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

// Generate sends a prompt and returns the full response.
// If streaming callbacks are set, they will be called during generation.
func (p *Provider) Generate(ctx context.Context, prompt string) (string, error) {
	p.mu.Lock()
	if p.client == nil {
		p.mu.Unlock()
		if err := p.Start(ctx); err != nil {
			return "", err
		}
		p.mu.Lock()
	}
	session := p.session
	onDelta := p.onDelta
	onDone := p.onDone
	p.mu.Unlock()

	if session == nil {
		return "", fmt.Errorf("no active session")
	}

	// Collect response
	var result strings.Builder
	done := make(chan error, 1)

	unsubscribe := session.On(func(event sdk.SessionEvent) {
		switch event.Type {
		case "assistant.message_delta":
			if event.Data.DeltaContent != nil {
				result.WriteString(*event.Data.DeltaContent)
				if onDelta != nil {
					onDelta(*event.Data.DeltaContent)
				}
			}
		case "assistant.message":
			if event.Data.Content != nil {
				// Final message - ensure we have full content
				if result.Len() == 0 {
					result.WriteString(*event.Data.Content)
				}
			}
		case "session.idle":
			done <- nil
		case "error":
			if event.Data.Content != nil {
				done <- fmt.Errorf("copilot error: %s", *event.Data.Content)
			} else {
				done <- fmt.Errorf("copilot error (no details)")
			}
		}
	})
	defer unsubscribe()

	// Send the message
	_, err := session.Send(sdk.MessageOptions{
		Prompt: prompt,
	})
	if err != nil {
		return "", fmt.Errorf("sending message: %w", err)
	}

	// Wait for completion or context cancellation
	select {
	case err := <-done:
		if err != nil {
			return "", err
		}
	case <-ctx.Done():
		session.Abort()
		return "", ctx.Err()
	}

	fullResponse := result.String()
	if onDone != nil {
		onDone(fullResponse)
	}

	return fullResponse, nil
}

// ListModels returns available models (stub - Copilot SDK doesn't expose model listing).
func (p *Provider) ListModels(ctx context.Context) ([]string, error) {
	// Copilot SDK doesn't have a model listing API; return known models
	return []string{
		"gpt-4o",
		"gpt-4-turbo",
		"gpt-3.5-turbo",
		"claude-sonnet-4-20250514",
		"o3-mini",
	}, nil
}

// IsAvailable checks if the Copilot SDK can be used.
func IsAvailable() bool {
	_, err := exec.LookPath("copilot")
	return err == nil
}
