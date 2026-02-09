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
| `vibeaura daemon start` | Start the background IPC server |
| `vibeaura extension list` | Manage installed AI extensions |
| `vibeaura direct [prompt]` | Non-interactive CLI mode (Headless) |
| `vibeaura uninstall` | Remove the tool but keep your data |

## Headless Automation (`direct` command)

For developers who want to integrate VibeAuracle into CI/CD pipelines, shell scripts, or cron jobs, the `direct` command allows interaction without the TUI.

### One-Shot Prompts
```bash
vibeaura direct "Refactor the main.go file to use interfaces"
```

### Piped Input
You can pipe the output of other commands into VibeAuracle for analysis:
```bash
cat internal/brain/brain.go | vibeaura direct "Explain the cognitive loop in this file"
```

### Direct REPL
If you run `vibeaura direct` without a prompt, it enters a lightweight, raw REPL mode that is much faster to load than the full TUI—perfect for low-resource environments.

### Flags
- `-v, --verbose`: Enable detailed status reporting (logs the "Thinking" steps).
- `-n, --non-interactive`: Exit immediately after processing the prompt (default when piping).
- `-a, --agent`: Override the agent engine for this command (e.g., `-a sdk`).

## TUI Commands (Inside the Chat)

When you are inside the interactive TUI, you can use slash commands to control the agent:

### Agent Control
- `/agent /vibe`: Switch to the internal Vibe agent.
- `/agent /sdk`: Switch to the GitHub Copilot SDK agent.
- `/agent /custom`: Manage custom agent personas.

### Session Management
- `/session /list`: View all stored sessions.
- `/session /clear`: Wipe history for the current directory.

### Media & Sharing
- `/shot`: Take a beautiful, high-fidelity screenshot of the current TUI view.
- `/record`: Start/stop high-quality screen recording of your TUI interactions.
- `/copy`: Copy the last Q&A block (including formatting) to your clipboard.

## System Snapshot
VibeAuracle is "System-Intimate". At the start of every session and before major actions, it takes a snapshot of:
- **CWD**: Your current working directory.
- **Git State**: Current branch and last commit SHA.
- **Resources**: CPU usage and available VRAM (to decide if local models are feasible).
