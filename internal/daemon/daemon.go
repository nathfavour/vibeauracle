package daemon

import (
	"fmt"
	"net"

	"google.golang.org/grpc"
)

// Daemon represents the background service
type Daemon struct {
	socketPath string
	server     *grpc.Server
}

func New(socketPath string) *Daemon {
	return &Daemon{
		socketPath: socketPath,
		server:     grpc.NewServer(),
	}
}

// Start launches the background service
func (d *Daemon) Start() error {
	lis, err := net.Listen("unix", d.socketPath)
	if err != nil {
		return fmt.Errorf("listening on unix socket: %w", err)
	}

	fmt.Printf("Daemon starting on %s\n", d.socketPath)
	return d.server.Serve(lis)
}

// Stop shuts down the background service
func (d *Daemon) Stop() {
	d.server.GracefulStop()
}

