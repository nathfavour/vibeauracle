---
sidebar_position: 4
---

# Usage

VibeAuracle provides both an interactive TUI and a set of CLI commands for system management.

## CLI Commands

| Command | Description |
|---------|-------------|
| `vibeaura` | Launch the interactive session (TUI) |
| `vibeaura update` | Pull the latest SHA from your current branch |
| `vibeaura sys stats` | View real-time system power snapshot |
| `vibeaura models` | Discover and switch AI providers |
| `vibeaura config` | View or update configuration |
| `vibeaura auth` | Manage credentials securely |
| `vibeaura uninstall` | Remove the tool but keep your data |

## TUI Commands (Inside the Chat)

When you are inside the interactive TUI, you can use slash commands to control the agent:

### Agent Control
- `/agent /vibe`: Switch to the internal Vibe agent.
- `/agent /sdk`: Switch to the GitHub Copilot SDK agent.
- `/agent /custom`: Manage custom agent personas.

### Session Management
- `/session /list`: View all stored sessions.
- `/session /clear`: Wipe history for the current directory.

### Authentication
- `/auth /openai <key>`: Set OpenAI API key.
- `/auth /github-models <pat>`: Set GitHub Models PAT.
- `/auth /ollama <url>`: Set Ollama endpoint.

## System Snapshot
VibeAuracle is "System-Intimate". At the start of every session and before major actions, it takes a snapshot of:
- **CWD**: Your current working directory.
- **Git State**: Current branch and last commit SHA.
- **Resources**: CPU usage and available VRAM (to decide if local models are feasible).
