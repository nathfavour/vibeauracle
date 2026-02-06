package brain

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
)

// StoreState persists application state
func (b *Brain) StoreState(id string, state interface{}) error {
	return b.memory.SaveState(id, state)
}

// RecallState retrieves application state
func (b *Brain) RecallState(id string, target interface{}) error {
	return b.memory.LoadState(id, target)
}

// ClearState removes application state
func (b *Brain) ClearState(id string) error {
	return b.memory.ClearState(id)
}

// ListSessions returns all stored directory-aware sessions
func (b *Brain) ListSessions() ([]string, error) {
	return b.memory.ListStates("chat_session:")
}

// StoreSecret saves a secret in the vault
func (b *Brain) StoreSecret(key, value string) error {
	if b.vault == nil {
		return fmt.Errorf("vault not initialized")
	}
	return b.vault.Set(key, value)
}

// GetSecret retrieves a secret from the vault
func (b *Brain) GetSecret(key string) (string, error) {
	if b.vault == nil {
		return "", fmt.Errorf("vault not initialized")
	}
	return b.vault.Get(key)
}

// GetSessionID returns a robust session ID based on the current directory.
// This ensures chats are directory-specific.
func (b *Brain) GetSessionID() string {
	cwd, _ := os.Getwd()
	hash := sha256.Sum256([]byte(cwd))
	return "chat_session:" + hex.EncodeToString(hash[:8])
}

// GetSessionPath returns the CWD for display purposes
func (b *Brain) GetSessionPath() string {
	cwd, _ := os.Getwd()
	return cwd
}

// GetDataPath returns a path inside the .vibeauracle directory
func (b *Brain) GetDataPath(subpath string) string {
	return b.cm.GetDataPath(subpath)
}
