# 🤖 Copilot SDK Integration Roadmap (TODO.SDK.md)

Objective: Integrate the official GitHub Copilot SDK to provide native, streaming LLM capabilities with proper session management and tool bridging.

## Prerequisites
- [x] Analyze `copilot-sdk/go` architecture and types
- [x] Ensure `copilot` CLI is detectable/installable (graceful fallback if missing)

---

## Phase 1: Foundation (SDK Provider)
- [x] **Create `internal/copilot/` package**
    - [x] `provider.go` - Implement `model.Provider` interface wrapping SDK
    - [x] `bridge.go` - Bridge VibeAuracle `tooling.Tool` → Copilot SDK `Tool`
    - [x] `events.go` - Handle streaming events (`assistant.message_delta`, etc.)

- [x] **Add SDK dependency**
    - [x] Add `github.com/github/copilot-sdk/go` to workspace
    - [x] Configure local replace directive pointing to `../copilot-sdk/go`

- [x] **Graceful Detection**
    - [x] Check if `copilot` CLI exists in PATH
    - [x] Fall back to existing `langchaingo` OpenAI provider if missing
    - [x] Log clear message about degraded mode

---

## Phase 2: Session Management
- [ ] **Session Lifecycle**
    - [ ] Map VibeAuracle "chat sessions" to Copilot SDK sessions
    - [ ] Implement `session.create` on new chat
    - [ ] Implement `session.resume` for conversation continuity
    - [ ] Implement `session.destroy` on clean exit

- [ ] **System Message Customization**
    - [x] Use `SystemMessageConfig.Mode = "append"` to inject VibeAuracle personality
    - [ ] Preserve SDK guardrails while adding our prompt layers

---

## Phase 3: Tool Bridge
- [ ] **Export VibeAuracle Tools to Copilot**
    - [x] Convert `tooling.ToolMetadata` → `copilot.Tool`
    - [x] Auto-generate JSON schema from our `Parameters` field
    - [x] Implement `ToolHandler` that routes to our `Tool.Execute()`

- [ ] **Bi-directional Tool Awareness**
    - [ ] Allow Copilot's native tools (file, bash, etc.) to coexist
    - [ ] Use `AvailableTools`/`ExcludedTools` for fine-grained control

---

## Phase 4: Streaming & Events
- [x] **Replace Blocking Generation**
    - [x] Use `session.On()` event handler for callbacks
    - [x] Emit `assistant.message_delta` to TUI's viewport incrementally
    - [x] Handle `session.idle` to know when response is complete

- [ ] **Reasoning Support (Optional)**
    - [x] Capture `assistant.reasoning_delta` for models that support it
    - [ ] Display in a collapsible "thinking" section in TUI

---

## Phase 5: BYOK (Bring Your Own Key)
- [ ] **Custom Provider Passthrough**
    - [ ] If user has OpenAI/Anthropic key in vault, configure `ProviderConfig`
    - [ ] Support `BearerToken` for custom backends
    - [ ] Allow `BaseURL` override for local models (Ollama via OpenAI-compat)

---

## Phase 6: MCP Integration (Future)
- [ ] **Model Context Protocol**
    - [ ] Bridge our planned MCP tools to Copilot's `MCPServers` config
    - [ ] Enable database, browser, and docs servers

---

## Implementation Status
✅ Phase 1 Complete - Foundation SDK provider with streaming and tool bridge
⏳ Phase 2 In Progress - Session lifecycle management
⏳ Phase 3 In Progress - Full tool bridge activation
