package connect

import (
	"context"
	"fmt"
	"os"

	"github.com/libp2p/go-libp2p"
	"github.com/libp2p/go-libp2p/core/host"
)

func init() {
	if os.Getenv("QUIC_GO_LOG_LEVEL") == "" {
		os.Setenv("QUIC_GO_LOG_LEVEL", "error")
	}
}

// Connector manages P2P connectivity
type Connector struct {
	host host.Host
	sm   *ShareManager
}

func NewConnector(ctx context.Context) (*Connector, error) {
	h, err := libp2p.New()
	if err != nil {
		return nil, fmt.Errorf("creating libp2p host: %w", err)
	}
	return &Connector{
		host: h,
		sm:   NewShareManager(),
	}, nil
}

// CreateSharedSession creates a new shared session with specific security and permissions
func (c *Connector) CreateSharedSession(sessionType, permissions, targetUser string, allowedClients []string) *SharedSession {
	return c.sm.CreateSession(sessionType, permissions, targetUser, allowedClients)
}

// GetAddress returns the P2P multiaddress of this node
func (c *Connector) GetAddress() string {
	addrs := c.host.Addrs()
	if len(addrs) == 0 {
		return ""
	}
	return fmt.Sprintf("%s/p2p/%s", addrs[0], c.host.ID())
}

// Close shuts down the connector
func (c *Connector) Close() error {
	return c.host.Close()
}
