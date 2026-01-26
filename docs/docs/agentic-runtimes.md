---
sidebar_position: 3
---

# Agentic Runtimes

VibeAuracle is a multi-engine orchestrator. You can switch the underlying agentic runtime depending on your needs using the `/agent` command.

## 🎨 Vibe Agent (`/agent /vibe`)

The **Vibe Agent** is our artisan internal loop. 

- **Transparency**: Every step of the thought process is visible in the TUI.
- **Loop Detection**: Features a custom heuristic loop detector that prevents runaway agent executions.
- **Customizability**: Uses our own internal prompt layering and tool selection heuristics.
- **Best for**: System-intimate tasks where you want full control and visibility.

## 🚀 Copilot SDK Agent (`/agent /sdk`)

The **Copilot SDK Agent** leverages the official GitHub Copilot SDK runtime.

- **Native Intelligence**: Delegates multi-step reasoning to GitHub's proprietary agentic engine.
- **Deep Tool-Intimacy**: Uses native SDK tool calling for higher reliability in complex tasks.
- **Zero-Config Auth**: Automatically uses your `gh` CLI credentials.
- **Best for**: High-stakes engineering, complex code refactoring, and secure tasks.

## 👤 Custom Agent (`/agent /custom`)

**Custom Agents** allow you to define specialized agent personas for specific workflows.

- **Personas**: Register experts with custom system prompts (e.g., "Go Performance Expert", "React Accessibility Specialist").
- **Focused Toolsets**: (Coming Soon) Restrict agents to specific subsets of tools for security and focus.
- **Usage**:
    - List: `/agent /custom /list`
    - Add: `/agent /custom /add <name> <prompt>`
    - Switch: `/agent /custom /use <name>`

## Switching Runtimes

You can toggle between engines on the fly in the TUI:
```bash
/agent /sdk
/agent /vibe
```
VibeAuracle intelligently remembers your manual choice and won't override it with auto-defaults.
