package brain

import (
	"context"
	"testing"

	"github.com/nathfavour/vibeauracle/model"
)

type MockProvider struct{}

func (m *MockProvider) Generate(ctx context.Context, prompt string) (string, model.Usage, error) {
	return "Mocked AI Response", model.Usage{TotalTokens: 10}, nil
}

func (m *MockProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) SetUsageCallback(cb func(model.Usage)) {}

func (m *MockProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {}

func (m *MockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{}, nil
}

func TestBrain_Process(t *testing.T) {
	b := New()
	// Force Vibe mode and mock provider to avoid system dependencies
	b.config.Agent.Mode = "vibe"
	b.usingCopilotSDK = false
	b.model = model.New(&MockProvider{})
	// Update prompts model as well
	b.prompts.SetModel(b.model)

	req := Request{
		ID:      "test-1",
		Content: "Hello Brain, please implement a test.",
	}

	resp, err := b.Process(context.Background(), req)
	if err != nil {
		t.Fatalf("Brain processing failed: %v", err)
	}

	if resp.Content != "Mocked AI Response" {
		t.Errorf("Unexpected brain response: %q", resp.Content)
	}
}
