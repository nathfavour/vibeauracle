---
sidebar_position: 8
---

# Vibes: The Extension API

**Vibes** are the soul of VibeAuracle's extensibility. They are natural-language-powered, markdown-defined extensions that can modify, extend, or completely redefine any aspect of the system—from UI colors to update schedules to agent behavior.

> **Philosophy:** Treat VibeAuracle like a microkernel. The core provides primitives; Vibes provide policy.

## 📜 Vibe Anatomy

A Vibe is a `.vibe.md` file placed in `~/.vibeauracle/vibes/` or any registered vibe directory. It uses a strict YAML front matter + Markdown body format.

```markdown
---
name: my-custom-vibe
version: 1.0.0
author: nathfavour
hooks:
  - on_startup
  - on_file_change
  - on_schedule
permissions:
  - config.write
  - ui.theme
  - scheduler.create
schedule: "*/5 * * * *"  # Cron syntax (optional)
---

# My Custom Vibe

This vibe does XYZ...

## Instructions

When activated, perform the following:
1. Change the TUI accent color to `#FF5733`
2. Every 5 minutes, run a git status check
3. If there are uncommitted changes, notify me
```

## ⚡ Core Concepts

### 1. Hooks
Vibes attach to lifecycle events like `on_startup`, `on_file_change`, `on_command`, `on_tool_call`, and more.

### 2. Permissions
Vibes must declare what they need to access, such as `config.read`, `ui.theme`, `agent.prompt`, or `system.shell`.

### 3. Natural Language Instructions
The Markdown body is parsed by the AI to understand intent, enabling zero-code extension development.

## 🗓️ Scheduler

Vibes can schedule recurring or one-shot tasks using cron syntax or ISO 8601 timestamps.

## 🎨 UI Customization

Vibes can define themes and layouts:
```yaml
ui:
  theme:
    primary: "#7C3AED"
    secondary: "#06B6D4"
    accent: "#F59E0B"
    background: "#0D0D0D"
```

## 🛠️ Defining Custom Tools

Vibes can register new tools for the AI:
```yaml
tools:
  - name: check_weather
    description: "Get current weather for a city"
    parameters:
      city:
        type: string
        required: true
    action: |
      curl -s "wttr.in/${city}?format=3"
```

## 🌍 Sharing Vibes

Vibes can be shared via Git repositories or direct file transfer.
```bash
vibeaura vibe install https://github.com/user/my-vibe
vibeaura vibe enable my-vibe
```
