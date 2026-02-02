package model

import (
	"context"
	"fmt"
	"net/http"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CopilotProvider implements the Provider interface for GitHub Copilot
type CopilotProvider struct {
	llm     llms.Model
	token   string
	usageCB func(Usage)
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

func (p *CopilotProvider) SetUsageCallback(cb func(Usage)) {
	p.usageCB = cb
}

// NewCopilotProvider creates a new GitHub Copilot provider
func NewCopilotProvider(token string, modelName string) (*CopilotProvider, error) {
	if modelName == "" {
		modelName = "gpt-4o" // Copilot default
	}

	llm, err := openai.New(
		openai.WithToken(token),
		openai.WithBaseURL(CopilotBaseURL), // Copilot LLM endpoint
		openai.WithModel(modelName),
		openai.WithHTTPClient(&http.Client{
			Transport: newGithubTransport(token, http.DefaultTransport),
		}),
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
func (p *CopilotProvider) Generate(ctx context.Context, prompt string) (string, Usage, error) {
	resp, err := p.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	})
	if err != nil {
		return "", Usage{}, fmt.Errorf("github copilot generate: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("github copilot generate: no choices returned")
	}

	content := resp.Choices[0].Content
	usage := ExtractUsage(resp.Choices[0].GenerationInfo)

	if p.usageCB != nil {
		p.usageCB(usage)
	}

	return content, usage, nil
}

// ListModels returns available models (stub for now, Copilot usually has fixed gpt-4o/gpt-3.5-turbo)
func (p *CopilotProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}, nil
}
