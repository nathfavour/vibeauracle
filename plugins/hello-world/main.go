package main

import (
	"context"
	"fmt"

	"github.com/nathfavour/vibeauracle/pkg/vibe"
)

// HelloWorldPlugin is a simple community-contributed plugin.
type HelloWorldPlugin struct{}

func (p *HelloWorldPlugin) Name() string {
	return "hello-world"
}

func (p *HelloWorldPlugin) Description() string {
	return "A simple plugin that says hello to the community."
}

func (p *HelloWorldPlugin) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	name, ok := args["name"].(string)
	if !ok {
		name = "World"
	}
	return fmt.Sprintf("Hello, %s! Welcome to the vibeauracle ecosystem.", name), nil
}

// Ensure the plugin implements the interface
var _ vibe.Plugin = (*HelloWorldPlugin)(nil)

func main() {
	fmt.Println("This is a plugin and not meant to be run directly.")
}

