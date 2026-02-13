package model

import (
	"context"
	"testing"
)

type MockProvider struct {
	Response string
	Err      error
}

func (m *MockProvider) Generate(ctx context.Context, prompt string) (GeneratedResponse, error) {
	return GeneratedResponse{Content: m.Response, Usage: Usage{}}, m.Err
}

func (m *MockProvider) ListModels(ctx context.Context) ([]string, error) {
	return []string{"mock-model"}, nil
}

func (m *MockProvider) Name() string {
	return "mock"
}

func (m *MockProvider) SetUsageCallback(cb func(Usage)) {}

func (m *MockProvider) SetStreamCallbacks(onDelta func(string), onDone func(string)) {}

func (m *MockProvider) Embed(ctx context.Context, texts []string) ([][]float32, error) {
	return [][]float32{}, nil
}

func TestModel_Generate(t *testing.T) {
	mock := &MockProvider{Response: "Test Response"}
	m := New(mock)

	resp, usage, err := m.Generate(context.Background(), "Hello")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if resp != "Test Response" {
		t.Errorf("Expected 'Test Response', got '%s'", resp)
	}

	_ = usage // Placeholder for usage check if needed
}
