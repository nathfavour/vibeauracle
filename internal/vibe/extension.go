package vibe

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
)

// ExtensionComms defines preferred communication methods
type ExtensionComms struct {
	TUI bool `json:"tui"`
	CLI bool `json:"cli"`
	UDS bool `json:"uds"`
}

// ExtensionCapabilities defines what the extension is allowed to do
type ExtensionCapabilities struct {
	Agentic  bool `json:"agentic"`   // Can be called by the AI agent
	ReadOnly bool `json:"read_only"` // Only allowed to read data
}

// Extension represents a Vibe extension/tool
type Extension struct {
	UUID         string                `json:"uuid"`
	Name         string                `json:"name"`
	Description  string                `json:"description"`
	Version      string                `json:"version"`
	RepoURI      string                `json:"repo_uri,omitempty"`
	DataDir      string                `json:"data_dir,omitempty"` // External tool's own data dir
	Enabled      bool                  `json:"enabled"`
	Comms        ExtensionComms        `json:"comms"`
	Capabilities ExtensionCapabilities `json:"capabilities"`
	Manifest     *Vibe                 `json:"manifest,omitempty"` // The underlying Vibe/MCP manifest
}

// Manager handles the lifecycle of extensions
type Manager struct {
	configDir  string
	mu         sync.RWMutex
	extensions map[string]*Extension
}

func NewManager(cfgDir string) *Manager {
	return &Manager{
		configDir:  cfgDir,
		extensions: make(map[string]*Extension),
	}
}

// Register initializes a new extension directory and vibe.json
func (m *Manager) Register(name string, description string) (*Extension, error) {
	id := uuid.New().String()
	ext := &Extension{
		UUID:        id,
		Name:        name,
		Description: description,
		Version:     "0.1.0",
		Enabled:     true,
		Comms: ExtensionComms{
			TUI: true,
			CLI: true,
			UDS: true,
		},
		Capabilities: ExtensionCapabilities{
			Agentic: true,
		},
	}

	if err := m.Save(ext); err != nil {
		return nil, err
	}

	m.mu.Lock()
	m.extensions[id] = ext
	m.mu.Unlock()

	return ext, nil
}

// Save persists the extension configuration
func (m *Manager) Save(ext *Extension) error {
	extDir := filepath.Join(m.configDir, "vibes", ext.UUID)
	if err := os.MkdirAll(extDir, 0755); err != nil {
		return fmt.Errorf("creating extension directory: %w", err)
	}

	data, err := json.MarshalIndent(ext, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling extension: %w", err)
	}

	return os.WriteFile(filepath.Join(extDir, "vibe.json"), data, 0644)
}

// LoadAll scans the vibes directory for extensions
func (m *Manager) LoadAll() error {
	vibesDir := filepath.Join(m.configDir, "vibes")
	if _, err := os.Stat(vibesDir); os.IsNotExist(err) {
		return nil
	}

	entries, err := os.ReadDir(vibesDir)
	if err != nil {
		return fmt.Errorf("reading vibes directory: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		vibeJson := filepath.Join(vibesDir, entry.Name(), "vibe.json")
		if _, err := os.Stat(vibeJson); os.IsNotExist(err) {
			continue
		}

		data, err := os.ReadFile(vibeJson)
		if err != nil {
			continue
		}

		var ext Extension
		if err := json.Unmarshal(data, &ext); err != nil {
			continue
		}

		m.extensions[ext.UUID] = &ext
	}

	return nil
}

// List returns all loaded extensions
func (m *Manager) List() []*Extension {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var list []*Extension
	for _, ext := range m.extensions {
		list = append(list, ext)
	}
	return list
}

// GetByUUID finds an extension by UUID
func (m *Manager) GetByUUID(id string) (*Extension, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	ext, ok := m.extensions[id]
	return ext, ok
}

// SetEnabled toggles an extension
func (m *Manager) SetEnabled(id string, enabled bool) error {
	m.mu.Lock()
	ext, ok := m.extensions[id]
	if !ok {
		m.mu.Unlock()
		return fmt.Errorf("extension not found: %s", id)
	}
	ext.Enabled = enabled
	m.mu.Unlock()

	return m.Save(ext)
}

// InitializeDefaults sets up auracrab and autocommiter if they don't exist
func (m *Manager) InitializeDefaults() error {
	defaults := []struct {
		Name string
		Desc string
	}{
		{"auracrab", "Intelligent Rust toolchain assistant"},
		{"autocommiter", "AI-powered git commit automator"},
	}

	for _, d := range defaults {
		exists := false
		for _, ext := range m.extensions {
			if ext.Name == d.Name {
				exists = true
				break
			}
		}

		if !exists {
			ext, err := m.Register(d.Name, d.Desc)
			if err != nil {
				return err
			}
			ext.Manifest = m.getDefaultManifest(d.Name)
			_ = m.Save(ext)
		}
	}

	return nil
}

func (m *Manager) getDefaultManifest(name string) *Vibe {
	switch name {
	case "autocommiter":
		return &Vibe{
			Name:        "autocommiter",
			Description: "AI-powered git commit automator",
			Protocol:    "stdio",
			Command:     "autocommiter",
			CLICommands: []CLICommand{
				{
					Name:        "commit",
					Description: "Generate and execute a smart commit for staged changes",
					Action:      "commit",
				},
			},
		}
	case "auracrab":
		return &Vibe{
			Name:        "auracrab",
			Description: "Intelligent Rust toolchain assistant",
			Protocol:    "stdio",
			Command:     "auracrab",
			CLICommands: []CLICommand{
				{
					Name:        "check",
					Description: "Run deep Rust health check and suggest fixes",
					Action:      "check",
				},
				{
					Name:        "fix",
					Description: "Automatically apply suggested Rust fixes",
					Action:      "fix",
				},
			},
		}
	}
	return nil
}
