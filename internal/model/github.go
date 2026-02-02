package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// GithubProvider implements the Provider interface for GitHub Models
type GithubProvider struct {
	llm     llms.Model
	token   string
	usageCB func(Usage)
	onDelta func(string)
	onDone  func(string)
}

const (
	GithubModelsBaseURL = "https://models.inference.ai.azure.com"
)

func init() {
	Register("github-models", func(config map[string]string) (Provider, error) {
		return NewGithubProvider(config["token"], config["model"])
	})
}

func (p *GithubProvider) Name() string { return "github-models" }

func (p *GithubProvider) SetUsageCallback(cb func(Usage)) {
	p.usageCB = cb
}

func (p *GithubProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.onDelta = onDelta
	p.onDone = onDone
}

// NewGithubProvider creates a new GitHub Models provider
func NewGithubProvider(token string, modelName string) (*GithubProvider, error) {
	if modelName == "" {
		modelName = "gpt-4o" // Sensible default for GitHub Models
	}

	llm, err := openai.New(
		openai.WithToken(token),
		openai.WithBaseURL(GithubModelsBaseURL),
		openai.WithModel(modelName),
		openai.WithHTTPClient(&http.Client{
			Transport: newGithubTransport(token, http.DefaultTransport),
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("github models init: %w", err)
	}

	return &GithubProvider{
		llm:   llm,
		token: token,
	}, nil
}

// Generate sends a prompt to GitHub Models and returns the response
func (p *GithubProvider) Generate(ctx context.Context, prompt string) (string, Usage, error) {
	opts := []llms.CallOption{}
	if p.onDelta != nil {
		opts = append(opts, llms.WithStreamingFunc(func(ctx context.Context, chunk []byte) error {
			p.onDelta(string(chunk))
			return nil
		}))
	}

	resp, err := p.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	}, opts...)
	if err != nil {
		return "", Usage{}, fmt.Errorf("github models generate: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("github models generate: no choices returned")
	}

	content := resp.Choices[0].Content
	usage := ExtractUsage(resp.Choices[0].GenerationInfo)

	if p.usageCB != nil {
		p.usageCB(usage)
	}

	if p.onDone != nil {
		p.onDone(content)
	}

	return content, usage, nil
}

// ListModels returns a list of available models from GitHub Models
func (p *GithubProvider) ListModels(ctx context.Context) ([]string, error) {
	// ... (existing code remains same)
	return models, nil
}

// Embed generates embeddings for the given texts using GitHub Models.
func (p *GithubProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embeddings, err := p.llm.CreateEmbedding(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("github models embed: %w", err)
	}

	return embeddings, nil
}
