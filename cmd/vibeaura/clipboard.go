package main

import (
	"github.com/atotto/clipboard"
)

func initClipboard() error {
	return nil
}

func writeToClipboard(text string) {
	_ = clipboard.WriteAll(text)
}