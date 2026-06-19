package ipc

import (
	"os"
	"path/filepath"
)

// SocketPath returns the vibeauracle UDS path.
// Override with VIBEAURA_SOCKET, VIBEIPC_SOCK, or AGENTIC_RUN_DIR.
func SocketPath() string {
	if v := os.Getenv("VIBEAURA_SOCKET"); v != "" {
		return v
	}
	if v := os.Getenv("VIBEIPC_SOCK"); v != "" {
		return v
	}
	if run := os.Getenv("AGENTIC_RUN_DIR"); run != "" {
		return filepath.Join(run, "vibeaura.sock")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".vibeauracle", "vibeaura.sock")
}
