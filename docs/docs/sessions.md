---
sidebar_position: 4
---

# Session Management

VibeAuracle features a robust, directory-aware session system that ensures your projects stay isolated.

## Directory Isolation

Sessions are automatically keyed by the **SHA-256 hash** of your current working directory (CWD). 

- **Project Security**: A chat started in `~/project-a` will never leak into a session in `~/project-b`.
- **Hashed Persistence**: Internal storage uses short hex-hashes for robustness, while the UI displays the clear-text path for clarity.

## Session Lifecycle

### Starting a Session
Simply launch `vibeaura` in any directory. The agent will check for an existing session tied to that path. If none is found, a fresh session is initialized.

### Viewing Sessions
Use the `/session /list` command to see all stored sessions on your system.

### Clearing History
You can wipe the history for your current project without affecting others:
```bash
/session /clear
```

## Persistent Knowledge

While chat history is isolated per project, VibeAuracle's **Project Knowledge** system goes deeper. It background-indexes architectural insights (entry points, languages, patterns) and ties them to specific Git commit SHAs. This ensures the agent's "perception" of your codebase remains accurate even as your code evolves.
