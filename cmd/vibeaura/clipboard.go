package main

import (
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
)

func initClipboard() error {
	return nil
}

func writeToClipboard(text string) {
	// Try native Go clipboard first (for Desktop)
	// On Android, this might fail or be a no-op depending on how it's built
	// but we'll try it anyway.
	_ = clipboard.WriteAll(text)

	// Additionally support Termux clipboard if available
	if _, err := exec.LookPath("termux-clipboard-set"); err == nil {
		cmd := exec.Command("termux-clipboard-set")
		cmd.Stdin = strings.NewReader(text)
		_ = cmd.Run()
	}
}