---
sidebar_position: 12
---

# Extensions System

VibeAuracle features a professional-grade extension system where tools are treated as user-space applications. This allows for granular control over tool permissions, communication methods, and lifecycle management.

## 📁 Extension Structure

Each extension resides in its own directory within the VibeAuracle data directory:
`~/.vibeauracle/vibes/{extension-uuid}/vibe.json`

### `vibe.json` Schema

```json
{
  "uuid": "unique-identifier",
  "name": "extension-name",
  "description": "helpful description",
  "version": "0.1.0",
  "enabled": true,
  "comms": {
    "tui": true,
    "cli": true,
    "uds": true
  },
  "capabilities": {
    "agentic": true,
    "read_only": false
  }
}
```

## 🛠️ Developer Guide: Building an Extension

Integrating your tool into the VibeAuracle ecosystem allows the AI agent to understand and use your binary as a first-class skill.

### 1. Registration
Start by registering your tool. This creates a directory and a base `vibe.json` file.
```bash
vibeaura extension register "my-db-tool" "A database migration helper"
```

### 2. Implementation & Configuration
Modify the generated `~/.vibeauracle/vibes/{uuid}/vibe.json` to point to your binary.

```json
{
  "name": "my-db-tool",
  "command": "/usr/local/bin/my-db-tool",
  "capabilities": {
    "agentic": true
  },
  "cli_commands": [
    {
      "name": "migrate",
      "description": "Run pending database migrations",
      "action": "up"
    }
  ]
}
```

### 3. Agentic Discovery
Once registered and enabled, the Brain automatically adds your tool to its internal "Tool Map." During the **Plan** phase of any request, the Brain will consider your extension if the request relates to its description.

### 4. Communication Methods
- **CLI**: Your tool can be invoked directly: `vibeaura my-db-tool migrate`.
- **TUI**: Use `/my-db-tool migrate` inside the chat.
- **UDS**: External tools can check if your extension is active via the `config` method.

## 🚀 Native Tooling (MCP)

For deeper integration (e.g., providing complex JSON parameters), we recommend building a **Model Context Protocol (MCP)** server. VibeAuracle can bridge standard MCP tools directly into its reasoning loop.

## 🛡️ Security

Extensions that are disabled or do not have `agentic: true` capability cannot be invoked by the AI agent, even if their binaries are present on the system. This provides a robust "opt-in" model for all third-party integrations.
