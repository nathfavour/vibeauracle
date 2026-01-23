# Copilot guidance for vibeauracle

This file records the GitHub Copilot CLI behavioral guardrails for the project.

## Core themes
- Treat the repo as a modular monolith governed by `go.work`, with a Bubble Tea CLI entry point (`cmd/vibeaura`) and dedicated modules for brain, model, sys, context, MCP, daemon, vault, and community vibes.
- Follow the Plan-Execute-Reflect loop, respect the Hexagonal layering, and keep explanations concise, system-aware, and tied back to README/ARCHITECTURE/FLOW when recommending changes.
- Prefer `vibeaura` and Go tooling (`go test ./...`, `go work sync`) for operations, and mention testing plans when touching multiple modules.

## Agent skill reminders
- Pause before editing when work spans multiple modules; summarize cross-module impact first.
- Request explicit approval for risky network, secret, or environment-impacting actions.
- Highlight the architecture (system intimacy, TUI-first, brain loop) when describing how solutions align with the project.
- Suggest module-scoped tests, citing specific commands, and link to docs (TODO.SDK.md, TODO.GH.md, docs/) when relevant.
