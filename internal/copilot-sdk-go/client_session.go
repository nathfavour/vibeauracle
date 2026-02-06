package copilot

import (
	"fmt"
)

// CreateSession creates a new conversation session with the specified configuration.
//
// This initializes a fresh conversation context with its own history and tools.
// If the client is not started and autoStart is enabled, it will automatically call [Client.Start].
//
// Example:
//
//	session, err := client.CreateSession(&copilot.SessionConfig{
//	    Model: "gpt-4",
//	    Tools: []copilot.Tool{myTool},
//	})
func (c *Client) CreateSession(config *SessionConfig) (*Session, error) {
	if c.client == nil {
		if c.autoStart {
			if err := c.Start(); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("client not connected. Call Start() first")
		}
	}

	params := make(map[string]interface{})
	if config != nil {
		if config.Model != "" {
			params["model"] = config.Model
		}
		if config.SessionID != "" {
			params["sessionId"] = config.SessionID
		}
		if len(config.Tools) > 0 {
			toolDefs := make([]map[string]interface{}, 0, len(config.Tools))
			for _, tool := range config.Tools {
				if tool.Name == "" {
					continue
				}
				definition := map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
				}
				if tool.Parameters != nil {
					definition["parameters"] = tool.Parameters
				}
				toolDefs = append(toolDefs, definition)
			}
			if len(toolDefs) > 0 {
				params["tools"] = toolDefs
			}
		}
		// Add system message configuration if provided
		if config.SystemMessage != nil {
			systemMessage := make(map[string]interface{})

			mode := config.SystemMessage.Mode
			if mode == "" {
				mode = "append"
			}
			systemMessage["mode"] = mode

			if mode == "replace" {
				if config.SystemMessage.Content != "" {
					systemMessage["content"] = config.SystemMessage.Content
				}
			} else {
				if config.SystemMessage.Content != "" {
					systemMessage["content"] = config.SystemMessage.Content
				}
			}

			if len(systemMessage) > 0 {
				params["systemMessage"] = systemMessage
			}
		}
		// Add tool filtering options
		if len(config.AvailableTools) > 0 {
			params["availableTools"] = config.AvailableTools
		}
		if len(config.ExcludedTools) > 0 {
			params["excludedTools"] = config.ExcludedTools
		}
		// Add streaming option
		if config.Streaming {
			params["streaming"] = config.Streaming
		}
		// Add provider configuration
		if config.Provider != nil {
			params["provider"] = buildProviderParams(config.Provider)
		}
		// Add permission request flag
		if config.OnPermissionRequest != nil {
			params["requestPermission"] = true
		}
		// Add MCP servers configuration
		if len(config.MCPServers) > 0 {
			params["mcpServers"] = config.MCPServers
		}
		// Add custom agents configuration
		if len(config.CustomAgents) > 0 {
			customAgents := make([]map[string]interface{}, 0, len(config.CustomAgents))
			for _, agent := range config.CustomAgents {
				agentMap := map[string]interface{}{
					"name":   agent.Name,
					"prompt": agent.Prompt,
				}
				if agent.DisplayName != "" {
					agentMap["displayName"] = agent.DisplayName
				}
				if agent.Description != "" {
					agentMap["description"] = agent.Description
				}
				if len(agent.Tools) > 0 {
					agentMap["tools"] = agent.Tools
				}
				if len(agent.MCPServers) > 0 {
					agentMap["mcpServers"] = agent.MCPServers
				}
				if agent.Infer != nil {
					agentMap["infer"] = *agent.Infer
				}
				customAgents = append(customAgents, agentMap)
			}
			params["customAgents"] = customAgents
		}
		// Add config directory override
		if config.ConfigDir != "" {
			params["configDir"] = config.ConfigDir
		}
		// Add skill directories configuration
		if len(config.SkillDirectories) > 0 {
			params["skillDirectories"] = config.SkillDirectories
		}
		// Add disabled skills configuration
		if len(config.DisabledSkills) > 0 {
			params["disabledSkills"] = config.DisabledSkills
		}
	}

	result, err := c.client.Request("session.create", params)
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	sessionID, ok := result["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing sessionId")
	}

	session := NewSession(sessionID, c.client)

	if config != nil {
		session.registerTools(config.Tools)
		if config.OnPermissionRequest != nil {
			session.registerPermissionHandler(config.OnPermissionRequest)
		}
	} else {
		session.registerTools(nil)
	}

	c.sessionsMux.Lock()
	c.sessions[sessionID] = session
	c.sessionsMux.Unlock()

	return session, nil
}

// ResumeSession resumes an existing conversation session by its ID using default options.
//
// This is a convenience method that calls [Client.ResumeSessionWithOptions] with nil config.
//
// Example:
//
//	session, err := client.ResumeSession("session-123")
func (c *Client) ResumeSession(sessionID string) (*Session, error) {
	return c.ResumeSessionWithOptions(sessionID, nil)
}

// ResumeSessionWithOptions resumes an existing conversation session with additional configuration.
//
// This allows you to continue a previous conversation, maintaining all conversation history.
// The session must have been previously created and not deleted.
//
// Example:
//
//	session, err := client.ResumeSessionWithOptions("session-123", &copilot.ResumeSessionConfig{
//	    Tools: []copilot.Tool{myNewTool},
//	})
func (c *Client) ResumeSessionWithOptions(sessionID string, config *ResumeSessionConfig) (*Session, error) {
	if c.client == nil {
		if c.autoStart {
			if err := c.Start(); err != nil {
				return nil, err
			}
		} else {
			return nil, fmt.Errorf("client not connected. Call Start() first")
		}
	}

	params := map[string]interface{}{
		"sessionId": sessionID,
	}

	if config != nil {
		if len(config.Tools) > 0 {
			toolDefs := make([]map[string]interface{}, 0, len(config.Tools))
			for _, tool := range config.Tools {
				if tool.Name == "" {
					continue
				}
				definition := map[string]interface{}{
					"name":        tool.Name,
					"description": tool.Description,
				}
				if tool.Parameters != nil {
					definition["parameters"] = tool.Parameters
				}
				toolDefs = append(toolDefs, definition)
			}
			if len(toolDefs) > 0 {
				params["tools"] = toolDefs
			}
		}
		if config.Provider != nil {
			params["provider"] = buildProviderParams(config.Provider)
		}
		// Add streaming option
		if config.Streaming {
			params["streaming"] = config.Streaming
		}
		// Add permission request flag
		if config.OnPermissionRequest != nil {
			params["requestPermission"] = true
		}
		// Add MCP servers configuration
		if len(config.MCPServers) > 0 {
			params["mcpServers"] = config.MCPServers
		}
		// Add custom agents configuration
		if len(config.CustomAgents) > 0 {
			customAgents := make([]map[string]interface{}, 0, len(config.CustomAgents))
			for _, agent := range config.CustomAgents {
				agentMap := map[string]interface{}{
					"name":   agent.Name,
					"prompt": agent.Prompt,
				}
				if agent.DisplayName != "" {
					agentMap["displayName"] = agent.DisplayName
				}
				if agent.Description != "" {
					agentMap["description"] = agent.Description
				}
				if len(agent.Tools) > 0 {
					agentMap["tools"] = agent.Tools
				}
				if len(agent.MCPServers) > 0 {
					agentMap["mcpServers"] = agent.MCPServers
				}
				if agent.Infer != nil {
					agentMap["infer"] = *agent.Infer
				}
				customAgents = append(customAgents, agentMap)
			}
			params["customAgents"] = customAgents
		}
		// Add skill directories configuration
		if len(config.SkillDirectories) > 0 {
			params["skillDirectories"] = config.SkillDirectories
		}
		// Add disabled skills configuration
		if len(config.DisabledSkills) > 0 {
			params["disabledSkills"] = config.DisabledSkills
		}
	}

	result, err := c.client.Request("session.resume", params)
	if err != nil {
		return nil, fmt.Errorf("failed to resume session: %w", err)
	}

	resumedSessionID, ok := result["sessionId"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid response: missing sessionId")
	}

	session := NewSession(resumedSessionID, c.client)
	if config != nil {
		session.registerTools(config.Tools)
		if config.OnPermissionRequest != nil {
			session.registerPermissionHandler(config.OnPermissionRequest)
		}
	} else {
		session.registerTools(nil)
	}

	c.sessionsMux.Lock()
	c.sessions[resumedSessionID] = session
	c.sessionsMux.Unlock()

	return session, nil
}
