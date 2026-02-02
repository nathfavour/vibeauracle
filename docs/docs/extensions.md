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

## 🛠️ Management Commands

| Command | Description |
|---------|-------------|
| `vibeaura extension list` | List all installed extensions and their status. |
| `vibeaura extension register <name> <desc>` | Generate a unique `vibe.json` for a new tool. |
| `vibeaura extension enable <uuid>` | Enable an extension. |
| `vibeaura extension disable <uuid>` | Disable an extension. |
| `vibeaura extension install <repo>` | (Planned) Install from a remote repository. |
| `vibeaura extension uninstall <uuid>` | Remove an extension and its data. |

## 🚀 Pre-installed Extensions

VibeAuracle comes with two core extensions enabled by default:
1. **Auracrab**: Intelligent Rust toolchain assistant.
2. **Autocommiter**: AI-powered git commit automator.

## 🛡️ Security

Extensions that are disabled or do not have `agentic: true` capability cannot be invoked by the AI agent, even if their binaries are present on the system. This provides a robust "opt-in" model for all third-party integrations.
