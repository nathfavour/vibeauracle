# VibeAuracle Roadmap & Competitive Edge

Objective: To be the most system-intimate, high-performance, and secure AI engineering companion.

## 🚀 Competitive Analysis: VibeAuracle vs. OpenCode

| Feature | VibeAuracle (Edge) | OpenCode (Edge) | Learnings for VibeAuracle |
|---------|-------------------|-----------------|---------------------------|
| **Platform** | Go (Native, Fast, Single Binary) | Node.js (Web-Ready, Monorepo) | Maintain native performance while matching UI polish. |
| **Auth** | Zero-Config (Inherits from `gh` CLI) | Built-in OAuth Device Flow | Add built-in OAuth as fallback if `gh` is missing. |
| **SDK** | Official Copilot SDK (Stable) | Manual OpenAI-Compat Layer | Leverage SDK for stability but keep header control for betas. |
| **Context** | Hardware/System Monitor (`sys.Monitor`) | Web/Console Integration | Deepen system intimacy (process tracking, memory usage). |
| **Enterprise** | Secondary Focus | First-Class GHE Flow | Implement `/auth enterprise` for corporate users. |

---

## 🛠️ Immediate TODOs (Learned from OpenCode)

- [ ] **Native OAuth Fallback**: Implement standalone Device Flow for Copilot auth (using `CLIENT_ID`). *Essential for users without `gh` CLI.*
- [ ] **Enterprise Support**: Allow custom GitHub Enterprise domains in `/auth login`.
- [ ] **Vision Support**: Add attachment handling for images in the TUI & SDK bridge.
- [ ] **LSP Bridge**: Integrate language servers (e.g., `gopls`, `pyright`) to give the agent "Go to Definition" and "Find References" capabilities.
- [ ] **Safe Shell Execution**: Parse shell commands (potentially using a Go tree-sitter binding) to detect dangerous operations before execution.
- [ ] **Patch-based Editing**: Implement a `patch` tool for more efficient, token-saving file modifications.
- [ ] **Dynamic Model Discovery**: Fetch model capabilities from a remote JSON (like `models.dev`) instead of hardcoding.
- [ ] **Intent Header Control**: Ensure we set `Openai-Intent: conversation-edits` and `X-Initiator` in the Copilot bridge for parity with "Official" behavior.
- [ ] **Cost & Token Tracking**: Monitor per-session token usage and estimated costs.
- [ ] **Plugin/Skill Ecosystem**: Expand `/skill` to support external plugins similar to OpenCode's architecture.

---

## 🗺️ Long-Term Roadmap

### 1. 🧠 Deep Project Context (RAG 2.0)
- [ ] Local vector DB for project-wide semantic search.
- [ ] Native AST parsing (using tree-sitter) for language-aware code navigation.
- [ ] Contextual "Project Rules" injection (Auto-detecting `.cursorrules`, `.github/copilot-instructions.md`).

### 2. ⚡ Autonomous Self-Healing
- [ ] Loop that runs tests and fixes failures automatically.
- [ ] Real-time error capture from system logs/background processes.

### 3. 🛡️ Hardware-Agentic Security
- [ ] Secure Enclave integration for secret management (beyond simple vault).
- [ ] Sandboxed execution for risky shell commands.

### 4. 🌐 MCP Bridge
- [ ] Native support for connecting to external MCP servers (Model Context Protocol).
- [ ] Expose VibeAuracle tools *as* an MCP server for other agents.

### 5. 🎨 UI/UX Refinement
- [ ] Streaming viewport rendering (Incremental TUI updates).
- [ ] Keybinding customization engine.
- [ ] Integrated file diff viewer in the TUI.
