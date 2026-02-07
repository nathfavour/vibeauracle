package model

import (
	"testing"
)

func TestRegistry(t *testing.T) {
	name := "test-provider"
	Register(name, func(config map[string]string) (Provider, error) {
		return &MockProvider{}, nil
	})

	p, err := GetProvider(name, nil)
	if err != nil {
		t.Fatalf("GetProvider failed: %v", err)
	}

	if p.Name() != "mock" {
		t.Errorf("expected mock name, got %s", p.Name())
	}

	_, err = GetProvider("non-existent", nil)
	if err == nil {
		t.Error("expected error for non-existent provider")
	}
}

func TestExtractUsage(t *testing.T) {
	tests := []struct {
		name     string
		info     map[string]any
		expected Usage
	}{
		{
			name: "ints",
			info: map[string]any{
				"PromptTokens":     10,
				"CompletionTokens": 20,
				"TotalTokens":      30,
			},
			expected: Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		},
		{
			name: "float64s",
			info: map[string]any{
				"PromptTokens":     10.0,
				"CompletionTokens": 20.0,
				"TotalTokens":      30.0,
			},
			expected: Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		},
		{
			name: "missing total",
			info: map[string]any{
				"PromptTokens":     10,
				"CompletionTokens": 20,
			},
			expected: Usage{InputTokens: 10, OutputTokens: 20, TotalTokens: 30},
		},
		{
			name:     "nil info",
			info:     nil,
			expected: Usage{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExtractUsage(tt.info)
			if got.InputTokens != tt.expected.InputTokens ||
				got.OutputTokens != tt.expected.OutputTokens ||
				got.TotalTokens != tt.expected.TotalTokens {
				t.Errorf("ExtractUsage() = %v, want %v", got, tt.expected)
			}
		})
	}
}
