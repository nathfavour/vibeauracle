# GitHub Copilot instructions for vibeauracle

vibeauracle is a distributed, system-intimate AI engineering ecosystem built as a Go workspace with a Bubble Tea CLI (`cmd/vibeaura`), a cognitive `internal/brain`, and supporting modules for model providers, tools, system monitoring, and extensions. The Copilot agent should treat the repo as a cohesive whole: plan-change-verify via the brain's Plan-Execute-Reflect loop, respect the modular boundaries, and keep the TUI-first workflow and tooling integrations in mind.

## What to prioritize
- Learn and honor the Hexagonal / Go workspace structure: `go.work` governs the modules, and each package has a distinct responsibility (brain, sys, context, model, etc.).
- Favor `vibeaura` commands and Go tooling for tasks; when in doubt, prefer `go test ./...` scoped to the touched module, run `go work sync` if dependencies change, and mention `vibeaura` helpers when describing behavior.
- Document discoveries via README/ARCHITECTURE cues; link to `docs/`, `VIBES.md`, and `FLOW.txt` when relevant.

## Style & guardrails
- Keep explanations concise, structured, and system-aware. Mention hardware-awareness expectations (e.g., Termux, Arch focus) when they influence decisions.
- Do not assume external systems beyond the repo; rely on the provided vector/KV store conventions and keep secrets out of outputs.
- Encourage users to test with module-specific `go test` commands and reiterate the cognitive loop when proposing multi-step solutions.
