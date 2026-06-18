package engine

import "context"

// MutationRequest carries a task payload for structural codebase mutation.
type MutationRequest struct {
	WorkDir string
	Payload string
	Metadata map[string]interface{}
}

// MutationResult reports the outcome of a mutation phase.
type MutationResult struct {
	Success  bool
	ExitCode int
	Output   string
	Changed  []string
}

// MutationInterface applies structural mutations to a target codebase.
type MutationInterface interface {
	Mutate(ctx context.Context, req MutationRequest) (*MutationResult, error)
}
