package model

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/tmc/langchaingo/embeddings"
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
func (p *GithubProvider) Generate(ctx context.Context, prompt string) (GeneratedResponse, error) {
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
		return GeneratedResponse{}, fmt.Errorf("github models generate: %w", err)
	}

	if len(resp.Choices) == 0 {
		return GeneratedResponse{}, fmt.Errorf("github models generate: no choices returned")
	}

	content := resp.Choices[0].Content
	usage := ExtractUsage(resp.Choices[0].GenerationInfo)

	if p.usageCB != nil {
		p.usageCB(usage)
	}

	if p.onDone != nil {
		p.onDone(content)
	}

	return GeneratedResponse{
		Content: content,
		Usage:   usage,
	}, nil
}

// ListModels returns a list of available models from GitHub Models
func (p *GithubProvider) ListModels(ctx context.Context) ([]string, error) {
	// GitHub Models uses the standard OpenAI /models endpoint or its own models API
	req, err := http.NewRequestWithContext(ctx, "GET", GithubModelsBaseURL+"/models", nil)
	if err != nil {
		return nil, err
	}
	client := &http.Client{
		Transport: newGithubTransport(p.token, http.DefaultTransport),
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github models list failed: %s", resp.Status)
	}

	// GitHub Models API can return either a top-level array or an object with a "data" field (OpenAI style)
	var raw interface{}
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decoding github models: %w", err)
	}

	var models []string

	processEntry := func(m map[string]interface{}) {
		// Use "name" as the primary identifier if available, per AI.md example
		name, _ := m["name"].(string)
		id, _ := m["id"].(string)

		target := name
		if target == "" {
			target = id
		}

		if target != "" {
			// Per AI.md: Filter for chat-friendly models.
			// We check the "task" field, but we also check "type" and name patterns
			// to ensure we don't miss anything that could be used for chat.
			task, _ := m["task"].(string)
			lTask := strings.ToLower(task)
			isChat := strings.Contains(lTask, "chat") || strings.Contains(lTask, "completion")

			// Fallback: name-based filtering if task info is missing or generic
			if !isChat || lTask == "" {
				lTarget := strings.ToLower(target)
				isChat = isChat || strings.Contains(lTarget, "gpt") ||
					strings.Contains(lTarget, "llama") ||
					strings.Contains(lTarget, "phi") ||
					strings.Contains(lTarget, "mistral") ||
					strings.Contains(lTarget, "mixtral") ||
					strings.Contains(lTarget, "command") ||
					strings.Contains(lTarget, "claude")
			}

			if isChat {
				models = append(models, target)
			}
		}
	}

	switch v := raw.(type) {
	case []interface{}:
		// Top-level array format
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				processEntry(m)
			}
		}
	case map[string]interface{}:
		// Object format (check for "data" field)
		if data, ok := v["data"].([]interface{}); ok {
			for _, item := range data {
				if m, ok := item.(map[string]interface{}); ok {
					processEntry(m)
				}
			}
		} else {
			// Maybe it's just a single object? (unlikely but safe)
			processEntry(v)
		}
	}

	if len(models) == 0 {
		return nil, fmt.Errorf("no models found in github response")
	}

	return models, nil
}

// Embed generates embeddings for the given texts using GitHub Models.
func (p *GithubProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	embedder, ok := p.llm.(embeddings.EmbedderClient)
	if !ok {
		return nil, fmt.Errorf("github models model does not support embeddings")
	}

	embeddings, err := embedder.CreateEmbedding(ctx, texts)
	if err != nil {
		return nil, fmt.Errorf("github models embed: %w", err)
	}

	return embeddings, nil
}
