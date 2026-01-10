package vibe

import "context"

// Plugin is the interface that community plugins must implement.
type Plugin interface {
	Name() string
	Description() string
	Execute(ctx context.Context, args map[string]interface{}) (string, error)
}

// Skill represents an agentic capability that can be registered with the Brain.
type Skill struct {
	ID          string
	Name        string
	Description string
	Action      func(ctx context.Context, input string) (string, error)
}

