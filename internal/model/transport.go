package model

import (
	"net/http"
)

// githubTransport is a custom http.RoundTripper that mimics the GitHub CLI (gh)
// by injecting specific headers required for optimal interaction with Copilot and GitHub APIs.
type githubTransport struct {
	token string
	base  http.RoundTripper
}

func newGithubTransport(token string, base http.RoundTripper) *githubTransport {
	if base == nil {
		base = http.DefaultTransport
	}
	return &githubTransport{
		token: token,
		base:  base,
	}
}

func (t *githubTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Standard GitHub Authentication
	if req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+t.token)
	}

	// Mimic GitHub CLI for Copilot API
	if req.URL.Host == "api.githubcopilot.com" {
		req.Header.Set("Copilot-Integration-Id", "copilot-4-cli")
	}

	// Mimic GitHub CLI for GitHub Models (Azure Inference)
	if req.URL.Host == "models.inference.ai.azure.com" {
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	}

	// Universal User-Agent for consistency
	req.Header.Set("User-Agent", "VibeAuracle/1.0 (GitHub CLI Hybrid)")

	return t.base.RoundTrip(req)
}
