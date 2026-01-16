package model

import (
	"context"
	"fmt"

	"github.com/nathfavour/vibeauracle/copilot"
)

// CopilotSDKProvider wraps the Copilot SDK provider to implement the model.Provider interface.
type CopilotSDKProvider struct {
	provider *copilot.Provider
}

func init() {
	Register("copilot-sdk", func(config map[string]string) (Provider, error) {
		return NewCopilotSDKProvider(config["model"])
	})
}

// NewCopilotSDKProvider creates a new provider using the official Copilot SDK.
// Returns an error if the copilot CLI is not available.
func NewCopilotSDKProvider(modelName string) (*CopilotSDKProvider, error) {
	if !copilot.IsAvailable() {
		return nil, fmt.Errorf("copilot CLI not available; install from https://docs.github.com/en/copilot/how-tos/set-up/install-copilot-cli")
	}

	provider, err := copilot.NewProvider(modelName)
	if err != nil {
		return nil, fmt.Errorf("creating copilot provider: %w", err)
	}

	return &CopilotSDKProvider{
		provider: provider,
	}, nil
}

// Name returns the provider name.
func (p *CopilotSDKProvider) Name() string {
	return "copilot-sdk"
}

// Generate sends a prompt and returns the response.
func (p *CopilotSDKProvider) Generate(ctx context.Context, prompt string) (string, error) {
	return p.provider.Generate(ctx, prompt)
}

// ListModels returns available models from the SDK.
func (p *CopilotSDKProvider) ListModels(ctx context.Context) ([]string, error) {
	return p.provider.ListModels(ctx)
}

// SetStreamCallbacks enables streaming mode with delta callbacks.
func (p *CopilotSDKProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.provider.SetStreamCallbacks(onDelta, onDone)
}

// Stop gracefully shuts down the SDK client.
func (p *CopilotSDKProvider) Stop() error {
	return p.provider.Stop()
}
