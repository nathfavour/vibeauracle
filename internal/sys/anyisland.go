package sys

import (
	"encoding/json"
	"net"
	"os"
	"path/filepath"
	"time"
)

// AnyislandHandshakeRequest is the request sent to the anyisland daemon
type AnyislandHandshakeRequest struct {
	Op string `json:"op"`
}

// AnyislandHandshakeResponse is the response from the anyisland daemon
type AnyislandHandshakeResponse struct {
	Status           string `json:"status"`
	ToolID           string `json:"tool_id,omitempty"`
	Version          string `json:"version,omitempty"`
	AnyislandVersion string `json:"anyisland_version,omitempty"`
}

// IsManagedByAnyisland checks if the current process is managed by Anyisland
// by performing a handshake with the local anyisland daemon socket.
func IsManagedByAnyisland() bool {
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}

	socketPath := filepath.Join(home, ".anyisland", "anyisland.sock")
	if _, err := os.Stat(socketPath); os.IsNotExist(err) {
		return false
	}

	conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
	if err != nil {
		return false
	}
	defer conn.Close()

	req := AnyislandHandshakeRequest{Op: "HANDSHAKE"}
	if err := json.NewEncoder(conn).Encode(req); err != nil {
		return false
	}

	var resp AnyislandHandshakeResponse
	conn.SetReadDeadline(time.Now().Add(100 * time.Millisecond))
	if err := json.NewDecoder(conn).Decode(&resp); err != nil {
		return false
	}

	return resp.Status == "MANAGED"
}
