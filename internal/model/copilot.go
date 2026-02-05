package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/tmc/langchaingo/embeddings"
	"github.com/tmc/langchaingo/llms"
	"github.com/tmc/langchaingo/llms/openai"
)

// CopilotProvider implements the Provider interface for GitHub Copilot
type CopilotProvider struct {
	llm     llms.Model
	token   string
	usageCB func(Usage)
	onDelta func(string)
	onDone  func(string)
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

func (p *CopilotProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.onDelta = onDelta
	p.onDone = onDone
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

	if p.onDone != nil {
		p.onDone(content)
	}

	return content, usage, nil
}

// ListModels returns available models
func (p *CopilotProvider) ListModels(ctx context.Context) ([]string, error) {
	// Fallback models
	fallback := []string{"gpt-4o", "gpt-4-turbo", "gpt-3.5-turbo"}

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

// Embed generates embeddings for the given texts using Copilot.
func (p *CopilotProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embedder, ok := p.llm.(embeddings.EmbedderClient)
	if !ok {
		return nil, fmt.Errorf("copilot model does not support embeddings")
	}

	embeddings, err := embedder.CreateEmbedding(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("copilot embed: %w", err)
	}

	return embeddings, nil
}
