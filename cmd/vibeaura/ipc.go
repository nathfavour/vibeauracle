package main

import (
	"context"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/nathfavour/vibeauracle/daemon"
)

type Processor interface {
	Process(ctx context.Context, req interface{}) (interface{}, error)
	GetSnapshot() (interface{}, error)
	Config() interface{}
}

func ensureDaemonRunning(p Processor) {
	home, _ := os.UserHomeDir()
	socketPath := filepath.Join(home, ".vibeauracle", "vibeaura.sock")

	// 1. Check if socket exists and is alive
	if _, err := os.Stat(socketPath); err == nil {
		conn, err := net.DialTimeout("unix", socketPath, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return // Already running
		}
		// Stale socket, remove it
		os.Remove(socketPath)
	}

	// 2. Start daemon in background
	go func() {
		d := daemon.NewServer(socketPath, p)
		_ = d.Start(context.Background())
	}()
}
