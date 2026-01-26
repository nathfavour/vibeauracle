---
sidebar_position: 5
---

# Configuration

VibeAuracle is designed to be flexible and secure. Configuration is managed via the CLI and stored in `~/.vibeauracle/config.yaml`.

## Managing Config

Use the `config` command to view or update settings:

```bash
# List all settings
vibeaura config

# Change a provider
vibeaura config model.provider openai

# Enable beta updates
vibeaura config update.beta true
```

## Authentication

We provide a specialized `/auth` command in the TUI to manage credentials securely in our vault.

| Provider | Command | Requirement |
|----------|---------|-------------|
| **Ollama** | `/auth /ollama <endpoint>` | Local Ollama running |
| **OpenAI** | `/auth /openai <api-key>` | API Key |
| **GitHub Models** | `/auth /github-models <pat>` | GitHub Personal Access Token |
| **Copilot SDK** | N/A | Authenticated `gh` CLI |

## Auto-Switching Logic

VibeAuracle features an intelligent **Zero-Config Onboarding** system:
1. It checks if the `copilot` CLI extension is installed.
2. If detected **and** you haven't manually locked in another provider, it automatically promotes you to `copilot-sdk` and `sdk` agent mode.
3. If you manually switch (e.g., `/agent /vibe`), the system honors your choice and stops auto-switching.
