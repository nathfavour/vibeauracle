package model

import (
	"context"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CopilotProvider implements the Provider interface for GitHub Copilot
type CopilotProvider struct {
	llm   llms.Model
	token string
}

const (
	CopilotBaseURL = "https://api.githubcopilot.com"
)

func init() {
	Register("github-copilot", func(config map[string]string) (Provider, error) {
		return NewCopilotProvider(config["token"], config["model"])
	})
}

func (p *CopilotProvider) Name() string { return "github-copilot" }

// NewCopilotProvider creates a new GitHub Copilot provider
func NewCopilotProvider(token string, modelName string) (*CopilotProvider, error) {
	if modelName == "" {
		modelName = "gpt-4o" // Copilot default
	}

	llm, err := openai.New(
		openai.WithToken(token),
		openai.WithBaseURL(CopilotBaseURL),
		openai.WithModel(modelName),
	)
	if err != nil {
		return nil, fmt.Errorf("github copilot init: %w", err)
	}

	return &CopilotProvider{
		llm:   llm,
		token: token,
	}, nil
}

// Generate sends a prompt to GitHub Copilot
func (p *CopilotProvider) Generate(ctx context.Context, prompt string) (string, error) {
	resp, err := llms.GenerateFromSinglePrompt(ctx, p.llm, prompt)
	if err != nil {
		return "", fmt.Errorf("github copilot generate: %w", err)
	}

	return resp, nil
}

// ListModels returns available models (stub for now, Copilot usually has fixed gpt-4o/gpt-3.5-turbo)
func (p *CopilotProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}, nil
}
