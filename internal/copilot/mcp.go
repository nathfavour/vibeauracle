package copilot

import (
	sdk "github.com/github/copilot-sdk/go"
)

// MCPServerConfig represents an MCP server configuration for Copilot SDK.
type MCPServerConfig struct {
	Name    string
	Type    string            // "local", "stdio", "http", "sse"
	Command string            // For local/stdio servers
	Args    []string          // Arguments for the command
	URL     string            // For http/sse servers
	Tools   []string          // List of tool names from this server
	Env     map[string]string // Environment variables
	Timeout int               // Timeout in seconds
}

// MCPBridge converts VibeAuracle MCP configs to Copilot SDK format.
type MCPBridge struct {
	servers map[string]sdk.MCPServerConfig
}

// NewMCPBridge creates a new MCP bridge.
func NewMCPBridge() *MCPBridge {
	return &MCPBridge{
		servers: make(map[string]sdk.MCPServerConfig),
	}
}

// AddLocalServer adds a local/stdio MCP server.
func (m *MCPBridge) AddLocalServer(config MCPServerConfig) {
	m.servers[config.Name] = sdk.MCPServerConfig{
		"type":    "local",
		"command": config.Command,
		"args":    config.Args,
		"tools":   config.Tools,
	}
	if len(config.Env) > 0 {
		m.servers[config.Name]["env"] = config.Env
	}
	if config.Timeout > 0 {
		m.servers[config.Name]["timeout"] = config.Timeout
	}
}

// AddRemoteServer adds an HTTP/SSE MCP server.
func (m *MCPBridge) AddRemoteServer(config MCPServerConfig) {
	serverType := config.Type
	if serverType == "" {
		serverType = "http"
	}
	m.servers[config.Name] = sdk.MCPServerConfig{
		"type":  serverType,
		"url":   config.URL,
		"tools": config.Tools,
	}
	if config.Timeout > 0 {
		m.servers[config.Name]["timeout"] = config.Timeout
	}
}

// GetSDKConfig returns the MCP servers configuration for the SDK.
func (m *MCPBridge) GetSDKConfig() map[string]sdk.MCPServerConfig {
	return m.servers
}

// CommonMCPServers returns commonly used MCP server configurations.
func CommonMCPServers() []MCPServerConfig {
	return []MCPServerConfig{
		{
			Name:    "filesystem",
			Type:    "local",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", "."},
			Tools:   []string{"read_file", "write_file", "list_directory"},
		},
		{
			Name:    "github",
			Type:    "local",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-github"},
			Tools:   []string{"search_repositories", "get_file_contents", "create_issue"},
		},
		{
			Name:    "memory",
			Type:    "local",
			Command: "npx",
			Args:    []string{"-y", "@modelcontextprotocol/server-memory"},
			Tools:   []string{"store", "retrieve", "list_keys"},
		},
	}
}
