# 🤖 VibeAuracle Agentic Context

Welcome, Agent. You are operating within **VibeAuracle**, a Distributed, System-Intimate AI Engineering Ecosystem. This project is built by and for AI-augmented engineers who value deep integration with the underlying operating system and hardware.

## 🌌 Project Essence

VibeAuracle is not just a CLI; it's a "God-tier" engineering companion that unifies the terminal, the IDE, and the AI assistant into a single, keyboard-centric experience.

### Core Philosophy
1.  **System Intimacy**: The agent must be aware of the environment (CWD, CPU, VRAM, OS) before suggesting or taking action.
2.  **Modular Runtimes**: We support multiple agentic engines (Internal Vibe loop, official Copilot SDK, and custom personas).
3.  **Keyboard-First**: The UI (TUI) is optimized for speed and fluidity, using the Bubble Tea framework.
4.  **Hexagonal Architecture**: The core logic (`internal/brain`) is decoupled from adapters (providers, tools, UI) to allow infinite extensibility.

## 🛠️ Development Guidelines

### For AI & Human Contributors
-   **Strict Go Patterns**: We use Go 1.21+ Workspaces. Maintain decoupling between modules.
-   **Security First**: Never expose secrets. Use the `vault` module. Tool execution requires explicit intent awareness.
-   **TUI Integrity**: UI changes must adhere to the Bubble Tea (TEA) pattern. Styling is done via Lipgloss.
-   **Agentic Responsibility**: When acting as an agent, prioritize `executeToolCalls` for multi-step tasks. Use the official Copilot SDK runtime (`/agent /sdk`) for high-stakes engineering.

## 🧠 Specialized Skills

We maintain localized expertise in `.agent/skills/`. Refer to these for specific domain knowledge:

1.  **arch-integrity**: Maintaining the modular monolith and hexagonal boundaries.
2.  **sys-intimacy**: Leveraging and extending the system monitoring and filesystem abstraction.
3.  **self-healing**: Implementing and refining the recursive doctor/healer loop.

## 🗺️ Architectural Mapping

- `cmd/vibeaura`: The TUI and CLI entry point.
- `internal/brain`: The cognitive orchestrator (The Brain).
- `internal/copilot`: Bridge to the official GitHub Copilot SDK.
- `internal/sys`: Hardware awareness and filesystem abstraction.
- `internal/model`: Unified interface for multiple AI providers (Ollama, OpenAI, GitHub).
- `pkg/vibe`: Public SDK for extensions.

---
*"We build tools that feel like an extension of the mind, not just the hands."*
