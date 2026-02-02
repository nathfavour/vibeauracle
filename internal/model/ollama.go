package model

import (
	"context"
	"fmt"
	"net/http"
	"net/url"

	"github.com/ollama/ollama/api"
)

func init() {
	Register("ollama", func(config map[string]string) (Provider, error) {
		return NewOllamaProvider(config["endpoint"], config["model"])
	})
}

// OllamaProvider implements the Provider interface for Ollama
type OllamaProvider struct {
	client  *api.Client
	model   string
	usageCB func(Usage)
	onDelta func(string)
	onDone  func(string)
}

func (p *OllamaProvider) Name() string { return "ollama" }

func (p *OllamaProvider) SetUsageCallback(cb func(Usage)) {
	p.usageCB = cb
}

func (p *OllamaProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {
	p.onDelta = onDelta
	p.onDone = onDone
}

// PullModel attempts to pull a model from the Ollama registry
func (p *OllamaProvider) PullModel(ctx context.Context, name string, progress func(any)) error {
	req := &api.PullRequest{
		Model: name,
	}

	fn := func(resp api.ProgressResponse) error {
		if progress != nil {
			progress(resp)
		}
		return nil
	}

	err := p.client.Pull(ctx, req, fn)
	if err != nil {
		return fmt.Errorf("ollama pull: %w", err)
	}
	return nil
}

// NewOllamaProvider creates a new Ollama provider
// host is the Ollama server URL (e.g., "http://localhost:11434")
// modelName is the model to use (e.g., "llama3")
func NewOllamaProvider(host string, modelName string) (*OllamaProvider, error) {
	client, err := api.ClientFromEnvironment()
	if err != nil {
		// Fallback to manual client creation if env vars are not set
		client = api.NewClient(&url.URL{Scheme: "http", Host: host}, http.DefaultClient)
	}

	return &OllamaProvider{
		client: client,
		model:  modelName,
	}, nil
}

// Generate sends a prompt to Ollama and returns the response
func (p *OllamaProvider) Generate(ctx context.Context, prompt string) (string, Usage, error) {
	var response string
	var usage Usage
	
	stream := p.onDelta != nil
	req := &api.GenerateRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: &stream,
	}

	fn := func(resp api.GenerateResponse) error {
		response += resp.Response
		if p.onDelta != nil {
			p.onDelta(resp.Response)
		}
		if resp.Done {
			usage = Usage{
				InputTokens:  resp.PromptEvalCount,
				OutputTokens: resp.EvalCount,
				TotalTokens:  resp.PromptEvalCount + resp.EvalCount,
			}
		}
		return nil
	}

	err := p.client.Generate(ctx, req, fn)
	if err != nil {
		return "", Usage{}, fmt.Errorf("ollama generate: %w", err)
	}

	if p.usageCB != nil {
		p.usageCB(usage)
	}

	if p.onDone != nil {
		p.onDone(response)
	}

	return response, usage, nil
}

// ListModels returns a list of available models from Ollama
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	resp, err := p.client.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("ollama list models: %w", err)
	}

	var models []string
	for _, m := range resp.Models {
		models = append(models, m.Name)
	}
	return models, nil
}

// Embed generates embeddings for the given texts using Ollama.
func (p *OllamaProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	req := &api.EmbedRequest{
		Model:  p.model,
		Input:  texts,
	}
	resp, err := p.client.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("ollama embed: %w", err)
	}

	embeddings := make([][]float32, len(resp.Embeddings))
	for i, emb := range resp.Embeddings {
		f32 := make([]float32, len(emb))
		for j, v := range emb {
			f32[j] = v
		}
		embeddings[i] = f32
	}
	return embeddings, nil
}

