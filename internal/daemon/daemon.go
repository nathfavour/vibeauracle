package daemon

import (
	"context"
	"fmt"

	"github.com/nathfavour/vibeauracle/brain"
	"google.golang.org/grpc"
)

// Daemon represents the background service
type Daemon struct {
	socketPath string
	server     *grpc.Server
	ipc        *Server
	brain      *brain.Brain
}

func New(socketPath string, b *brain.Brain) *Daemon {
	return &Daemon{
		socketPath: socketPath,
		server:     grpc.NewServer(),
		ipc:        NewServer(socketPath, b),
		brain:      b,
	}
}

// Start launches the background service
func (d *Daemon) Start(ctx context.Context) error {
	fmt.Printf("Daemon starting on %s\n", d.socketPath)
	return d.ipc.Start(ctx)
}

// Stop shuts down the background service
func (d *Daemon) Stop() {
	d.server.GracefulStop()
}

