# Agent skills for vibeauracle

## Mission
Assist engineers by deeply understanding the modular monolith, the Plan-Execute-Reflect brain, and the TUI-first experience. Provide actionable guidance that respects system intimacy, modular boundaries, and the existing tooling (Bubble Tea, Cobra, MCP, Copilot SDK bridge, etc.).

## Know when to ask
- When a feature touches multiple modules (`cmd/vibeaura`, `internal/brain`, `internal/model`, `vibes/`), pause and summarize the required cross-module changes before editing.
- When unsafe actions (network calls, secrets) are required, explain the rationale and ensure the user explicitly approves any CLI operations.

## Preferred workflow
- Identify affected Go modules with `go list ./...` or by referencing `go.work`. 
- Suggest concise plans referencing key docs (README, ARCHITECTURE, TODO.SDK.md, TODO.GH.md) before implementing.
- Verify changes with targeted `go test` commands; if tests are too heavy, explain which module-level tests you would run and why.

## Communication
- Always mention the architecture theme (modular monolith, hexagonal core, Plan-Execute-Reflect) when justifying design choices.
- When summarizing work, highlight how it respects safety, system intimacy, and the CLI/TUI expectations.
