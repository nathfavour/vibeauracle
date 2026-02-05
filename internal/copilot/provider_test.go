package copilot

import (
	"context"
	"testing"
)

func TestListModels(t *testing.T) {
	p := &Provider{}
	models, err := p.ListModels(context.Background())
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(models) == 0 {
		t.Error("Expected models, got none")
	}

	t.Logf("Discovered models: %v", models)

	// Check for some common models that should be in the list or fallback
	foundGpt4o := false
	for _, m := range models {
		if m == "gpt-4o" {
			foundGpt4o = true
			break
		}
	}

	if !foundGpt4o {
		t.Error("Expected to find gpt-4o in model list")
	}
}
