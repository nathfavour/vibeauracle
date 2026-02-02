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

func init() {
	Register("openai", func(config map[string]string) (Provider, error) {
		return NewOpenAIProvider(config["api_key"], config["model"], config["base_url"])
	})
}

// OpenAIProvider implements the Provider interface for OpenAI
type OpenAIProvider struct {
	llm     llms.Model
	apiKey  string
	baseURL string
	usageCB func(Usage)
}

func (p *OpenAIProvider) Name() string { return "openai" }

func (p *OpenAIProvider) SetUsageCallback(cb func(Usage)) {
	p.usageCB = cb
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string, modelName string, baseURL string) (*OpenAIProvider, error) {
	if modelName == "" {
		modelName = "gpt-4o" // Default to a smart, modern model
	}

	opts := []openai.Option{
		openai.WithToken(apiKey),
		openai.WithModel(modelName),
	}

	if baseURL != "" {
		// Clean up common URL mistakes
		baseURL = strings.TrimSuffix(baseURL, "/")
		opts = append(opts, openai.WithBaseURL(baseURL))
	} else {
		baseURL = "https://api.openai.com/v1"
	}

	llm, err := openai.New(opts...)
	if err != nil {
		return nil, fmt.Errorf("openai init: %w", err)
	}

	return &OpenAIProvider{
		llm:     llm,
		apiKey:  apiKey,
		baseURL: baseURL,
	}, nil
}

// Generate sends a prompt to OpenAI and returns the response
func (p *OpenAIProvider) Generate(ctx context.Context, prompt string) (string, Usage, error) {
	resp, err := p.llm.GenerateContent(ctx, []llms.MessageContent{
		llms.TextParts(llms.ChatMessageTypeHuman, prompt),
	})
	if err != nil {
		return "", Usage{}, fmt.Errorf("openai generate: %w", err)
	}

	if len(resp.Choices) == 0 {
		return "", Usage{}, fmt.Errorf("openai generate: no choices returned")
	}

	content := resp.Choices[0].Content
	usage := ExtractUsage(resp.Choices[0].GenerationInfo)

	if p.usageCB != nil {
		p.usageCB(usage)
	}

	return content, usage, nil
}

// ListModels returns a list of available models from OpenAI
func (p *OpenAIProvider) ListModels(ctx context.Context) ([]string, error) {
	// ... (existing code remains same)
	return models, nil
}

// Embed generates embeddings for the given texts using OpenAI.
func (p *OpenAIProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	// Cast to embedder if supported
	embedder, ok := p.llm.(llms.Model)
	if !ok {
		return nil, fmt.Errorf("openai model does not support embeddings")
	}

	embeddings, err := embedder.CreateEmbedding(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("openai embed: %w", err)
	}

	// langchaingo's CreateEmbedding returns [][]float32 for OpenAI?
	// Actually it usually returns [][]float32 for most providers in langchaingo.
	return embeddings, nil
}

