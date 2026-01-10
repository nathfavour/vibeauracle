package vault

import (
	"fmt"

	"github.com/99designs/keyring"
)

// Vault handles secure credential storage
type Vault struct {
	ring keyring.Keyring
}

func New(serviceName string) (*Vault, error) {
	ring, err := keyring.Open(keyring.Config{
		ServiceName: serviceName,
	})
	if err != nil {
		return nil, fmt.Errorf("opening keyring: %w", err)
	}
	return &Vault{ring: ring}, nil
}

// Set stores a secret in the OS keyring
func (v *Vault) Set(key, value string) error {
	err := v.ring.Set(keyring.Item{
		Key:  key,
		Data: []byte(value),
	})
	if err != nil {
		return fmt.Errorf("setting secret: %w", err)
	}
	return nil
}

// Get retrieves a secret from the OS keyring
func (v *Vault) Get(key string) (string, error) {
	item, err := v.ring.Get(key)
	if err != nil {
		return "", fmt.Errorf("getting secret: %w", err)
	}
	return string(item.Data), nil
}

