package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/daemon"
	vipc "github.com/nathfavour/vibeauracle/pkg/ipc"
	"github.com/spf13/cobra"
)

type Processor interface {
	Process(ctx context.Context, req interface{}) (interface{}, error)
	GetSnapshot() (interface{}, error)
	Config() interface{}
}

func ensureDaemonRunning(p Processor) {
	socketPath := vipc.SocketPath()

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

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Manage the vibeaura background daemon",
}

var daemonStartCmd = &cobra.Command{
	Use:   "start",
	Short: "Start the background daemon with UDS support",
	Run: func(cmd *cobra.Command, args []string) {
		b := brain.New()

		// Determine socket path
		socketPath := vipc.SocketPath()

		d := daemon.NewServer(socketPath, b)

		fmt.Printf("Starting vibeaura daemon...\n")
		fmt.Printf("Socket: %s\n", socketPath)

		if err := d.Start(context.Background()); err != nil {
			fmt.Printf("Error starting daemon: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	daemonCmd.AddCommand(daemonStartCmd)
}