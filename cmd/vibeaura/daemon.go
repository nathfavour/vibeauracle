package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/daemon"
	"github.com/spf13/cobra"
)

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
		home, _ := os.UserHomeDir()
		socketPath := filepath.Join(home, ".vibeauracle", "vibeaura.sock")

		d := daemon.New(socketPath, b)

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
