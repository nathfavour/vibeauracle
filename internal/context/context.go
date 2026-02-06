package context

import (
	"bytes"
	"os/exec"
	"strings"
)

// GetGitSHA returns the current Git HEAD SHA for a directory
func GetGitSHA(path string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = path
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return "no-git"
	}
	return strings.TrimSpace(out.String())
}