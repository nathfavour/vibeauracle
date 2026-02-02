---
name: telegram-gateway
version: 1.1.0
author: nathfavour
hooks:
  - on_startup
permissions:
  - system.shell
  - agent.tools
---

# Telegram Agent Gateway Vibe

This vibe transforms your Telegram bots into remote agentic gateways for VibeAuracle. It allows you to interact with your system-intimate AI from anywhere via Telegram.

## Operation Modes
- **Chat Mode**: Standard conversational interface with full context.
- **CLI Mode**: Fast, one-shot mode for executing system tasks and queries.
- Use `/mode [chat|cli]` in Telegram to switch on the fly.

## Instructions

When this vibe is active:
1. On startup, remind the user to ensure the `tel` service is running: `cd /home/nathfavour/code/nathfavour/poc/tel && ./tel start`
2. Register the `telegram_broadcast` tool to allow the AI to push notifications or reports directly to the user's mobile device.

## Tools

```yaml
tools:
  - name: telegram_broadcast
    description: "Send a critical notification or report to all registered Telegram bot owners"
    parameters:
      message:
        type: string
        required: true
    action: |
      /home/nathfavour/code/nathfavour/poc/tel/tel broadcast "${message}"
```
