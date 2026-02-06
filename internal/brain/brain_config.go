package brain

import (
	"context"
	"fmt"
	"strings"

	"github.com/nathfavour/vibeauracle/auth"
	"github.com/nathfavour/vibeauracle/model"
	"github.com/nathfavour/vibeauracle/sys"
)

// DiscoverModels fetches available models from all configured providers
func (b *Brain) DiscoverModels(ctx context.Context) ([]ModelDiscovery, error) {
	var discoveries []ModelDiscovery

	// List of potential providers to check
	providersToCheck := []string{"ollama", "openai", "github-models", "github-copilot", "copilot-sdk"}

	for _, pName := range providersToCheck {
		configMap := map[string]string{
			"endpoint": b.config.Model.Endpoint,
			"base_url": b.config.Model.Endpoint,
		}

		// Hydrate with credentials
		if b.vault != nil {
			switch pName {
			case "github-models", "github-copilot":
				if token, err := b.vault.Get("github_models_pat"); err == nil {
					configMap["token"] = token
				} else {
					// Fallback to CLI token
					if ghToken, _ := auth.GetGithubCLIToken(); ghToken != "" {
						configMap["token"] = ghToken
					} else {
						continue // Still no token, skip
					}
				}
			case "openai":
				if key, err := b.vault.Get("openai_api_key"); err == nil {
					configMap["api_key"] = key
				} else {
					continue // No key, skip
				}
			case "ollama":
				// Usually no auth needed for local ollama
			}
		}

		p, err := model.GetProvider(pName, configMap)
		if err != nil {
			continue
		}

		models, err := p.ListModels(ctx)
		if err != nil {
			continue
		}

		for _, m := range models {
			discoveries = append(discoveries, ModelDiscovery{
				Name:     m,
				Provider: pName,
			})
		}
	}

	return discoveries, nil
}

// SetModel updates the active model and provider
func (b *Brain) SetModel(provider, name string) error {
	b.config.Model.Provider = provider
	b.config.Model.Name = name
	b.config.Model.UserConfigured = true

	// If provider is ollama, we might need to handle endpoint too,
	// but for now we keep the existing one or reset to default if changed.
	if provider == "ollama" && b.config.Model.Endpoint == "" {
		b.config.Model.Endpoint = "http://localhost:11434"
	}

	if err := b.cm.Save(b.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}

	b.initProvider()
	return nil
}

func (b *Brain) GetConfig() *sys.Config {
	return b.config
}

// Config is an alias for GetConfig
func (b *Brain) Config() interface{} {
	return b.config
}

// UpdateConfig updates the brain's configuration and persists it
func (b *Brain) UpdateConfig(cfg *sys.Config) error {
	b.config = cfg
	if err := b.cm.Save(b.config); err != nil {
		return fmt.Errorf("saving config: %w", err)
	}
	b.initProvider()
	return nil
}

func (b *Brain) autodetectBestModel() {
	// Only autodetect if we are using the default "llama3" which might not exist,
	// or if the model name is empty/none.
	// If we've already promoted to github-copilot, skip autodetection unless it fails.
	if b.config.Model.Provider == "github-copilot" {
		return
	}
	if b.config.Model.Name != "llama3" && b.config.Model.Name != "" && b.config.Model.Name != "none" {
		return
	}

	ctx := context.Background()
	discoveries, err := b.DiscoverModels(ctx)
	if err != nil || len(discoveries) == 0 {
		return
	}

	// 1. Try to find if LLAMA-3 or 3.2 is actually there (better matching than just 'llama3')
	for _, d := range discoveries {
		name := strings.ToLower(d.Name)
		if strings.Contains(name, "llama") || strings.Contains(name, "gpt-4o") || strings.Contains(name, "phi-3") {
			b.SetModel(d.Provider, d.Name)
			return
		}
	}

	// 2. Fallback to the first available model from any provider
	if len(discoveries) > 0 {
		b.SetModel(discoveries[0].Provider, discoveries[0].Name)
	}
}
