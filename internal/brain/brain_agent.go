package brain

import (
	"fmt"

	"github.com/nathfavour/vibeauracle/auth"
	"github.com/nathfavour/vibeauracle/internal/vibe"
	"github.com/nathfavour/vibeauracle/sys"
)

// SetAgentMode switches between 'vibe', 'sdk', 'gemini', and 'custom' agentic runtimes
func (b *Brain) SetAgentMode(mode string) error {
	if mode != "vibe" && mode != "sdk" && mode != "custom" && mode != "gemini" {
		return fmt.Errorf("invalid agent mode: %s (must be 'vibe', 'sdk', 'gemini', or 'custom')", mode)
	}
	b.config.Agent.Mode = mode
	b.config.Agent.UserConfigured = true
	return b.cm.Save(b.config)
}

// RegisterCustomAgent adds or updates a user-defined agent
func (b *Brain) RegisterCustomAgent(agent sys.CustomAgent) error {
	for i, a := range b.config.Agent.CustomAgents {
		if a.Name == agent.Name {
			b.config.Agent.CustomAgents[i] = agent
			return b.cm.Save(b.config)
		}
	}
	b.config.Agent.CustomAgents = append(b.config.Agent.CustomAgents, agent)
	return b.cm.Save(b.config)
}

// GetCustomAgents returns the list of registered custom agents
func (b *Brain) GetCustomAgents() []sys.CustomAgent {
	return b.config.Agent.CustomAgents
}

// SetActiveCustomAgent sets the active custom agent by name
func (b *Brain) SetActiveCustomAgent(name string) error {
	for _, a := range b.config.Agent.CustomAgents {
		if a.Name == name {
			b.config.Agent.ActiveCustom = name
			b.config.Agent.Mode = "custom"
			return b.cm.Save(b.config)
		}
	}
	return fmt.Errorf("custom agent '%s' not found", name)
}

// GetIdentity returns the current user identity if available
func (b *Brain) GetIdentity() string {
	if b.config.Model.Provider == "github-copilot" || b.config.Model.Provider == "github-models" {
		return auth.GetGithubUser()
	}
	return ""
}

// Extensions returns the list of loaded extensions
func (b *Brain) Extensions() []*vibe.Extension {
	return b.extMgr.List()
}

// RegisterExtension registers a new extension
func (b *Brain) RegisterExtension(name, desc string) (*vibe.Extension, error) {
	return b.extMgr.Register(name, desc)
}

// SetExtensionEnabled enables or disables an extension
func (b *Brain) SetExtensionEnabled(id string, enabled bool) error {
	return b.extMgr.SetEnabled(id, enabled)
}
