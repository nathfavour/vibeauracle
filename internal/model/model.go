package model

import (
	"context"
	"fmt"
)

// Usage represents token usage for a model generation
type Usage struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
	Cost         float64 // Optional: estimated cost in USD
}

// Provider represents an AI model provider (e.g., Ollama, OpenAI)
type Provider interface {
	Generate(ctx context.Context, prompt string) (string, Usage, error)
	ListModels(ctx context.Context) ([]string, error)
	Name() string
	SetUsageCallback(cb func(Usage))
}

// Pullable represents a provider that supports downloading models (like Ollama)
type Pullable interface {
	// PullModel is specific to providers that manage their own local models
}

type ProviderFactory func(config map[string]string) (Provider, error)

var (
	registry = make(map[string]ProviderFactory)
)

// Register adds a new provider factory to the registry
func Register(name string, factory ProviderFactory) {
	registry[name] = factory
}

// GetProvider creates a provider instance using the registry
func GetProvider(name string, config map[string]string) (Provider, error) {
	factory, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return factory(config)
}

// Model handles AI interactions
type Model struct {
	provider Provider
}

// New creates a new Model with the given provider
func New(p Provider) *Model {
	return &Model{provider: p}
}

// Generate uses the configured provider to generate a response
func (m *Model) Generate(ctx context.Context, prompt string) (string, Usage, error) {
	if m.provider == nil {
		return "", Usage{}, fmt.Errorf("no provider configured")
	}
	return m.provider.Generate(ctx, prompt)
}

// SetUsageCallback sets a callback for usage updates
func (m *Model) SetUsageCallback(cb func(Usage)) {
	if m.provider != nil {
		m.provider.SetUsageCallback(cb)
	}
}
