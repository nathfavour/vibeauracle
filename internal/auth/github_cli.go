package auth

import (
	"os/exec"
	"strings"

	"github.com/cli/go-gh/v2/pkg/auth"
)

// GetGithubCLIToken attempts to retrieve the GitHub token from the gh CLI.
// It checks for a token on github.com by default.
func GetGithubCLIToken() (string, string) {
	token, host := auth.TokenForHost("github.com")
	return token, host
}

// GetGithubUser returns the current authenticated GitHub user login.
func GetGithubUser() string {
	cmd := exec.Command("gh", "api", "user", "--template", "{{.login}}")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
