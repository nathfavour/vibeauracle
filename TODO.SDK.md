# 🤖 Copilot SDK Integration Roadmap (TODO.SDK.md)

Objective: Integrate the official GitHub Copilot SDK to provide native, streaming LLM capabilities with proper session management and tool bridging.

## Prerequisites
- [x] Analyze `copilot-sdk/go` architecture and types
- [x] Ensure `copilot` CLI is detectable/installable (graceful fallback if missing)

---

## Phase 1: Foundation (SDK Provider) ✅
- [x] **Create `internal/copilot/` package**
    - [x] `provider.go` - Implement `model.Provider` interface wrapping SDK
    - [x] `bridge.go` - Bridge VibeAuracle `tooling.Tool` → Copilot SDK `Tool`
    - [x] `events.go` - Handle streaming events (`assistant.message_delta`, etc.)
    - [x] `mcp.go` - Bridge for Model Context Protocol servers

- [x] **Add SDK dependency**
    - [x] Add `github.com/github/copilot-sdk/go` to workspace (`internal/copilot-sdk-go`)
    - [x] Configure local replace directive pointing to `./internal/copilot-sdk-go`

- [x] **Graceful Detection**
    - [x] Check if `copilot` CLI exists in PATH
    - [x] Fall back to existing `langchaingo` OpenAI provider if missing
    - [x] Log clear message about degraded mode

---

## Phase 2: Brain Integration ✅
- [x] **Wire into Cognitive Loop**
    - [x] Import copilot package in brain
    - [x] Auto-select `copilot-sdk` when CLI available and gh auth present
    - [x] Dual generation path (SDK or standard model)
    - [x] Graceful shutdown via `Brain.Shutdown()`

- [x] **System Message Customization**
    - [x] Use `SystemMessageConfig.Mode = "append"` to inject VibeAuracle personality
    - [x] Preserve SDK guardrails while adding our prompt layers

---

## Phase 3: Tool Bridge ✅
- [x] **Export VibeAuracle Tools to Copilot**
    - [x] Convert `tooling.ToolMetadata` → `copilot.Tool`
    - [x] Auto-generate JSON schema from our `Parameters` field
    - [x] Implement `ToolHandler` that routes to our `Tool.Execute()`
    - [x] Register core tools with SDK on startup

- [ ] **Bi-directional Tool Awareness**
    - [ ] Allow Copilot's native tools (file, bash, etc.) to coexist
    - [ ] Use `AvailableTools`/`ExcludedTools` for fine-grained control

---

## Phase 4: Streaming & Events ✅
- [x] **Replace Blocking Generation**
    - [x] Use `session.On()` event handler for callbacks
    - [x] Emit `assistant.message_delta` to provider callbacks
    - [x] Handle `session.idle` to know when response is complete

- [x] **TUI Integration**
    - [x] Add `streamDeltaMsg` and `streamDoneMsg` message types
    - [x] Add streaming content state to model struct
    - [x] Real-time viewport updates with cursor indicator

---

## Phase 5: BYOK (Bring Your Own Key) ✅
- [x] **Custom Provider Passthrough**
    - [x] Add `ProviderOptions` struct with BYOK fields
    - [x] Check vault for `openai_api_key` and `anthropic_api_key`
    - [x] Configure `sdk.ProviderConfig` with custom credentials
    - [x] Allow `BaseURL` override for local models (Ollama via OpenAI-compat)

---

## Phase 6: MCP Integration ✅
- [x] **Model Context Protocol**
    - [x] Create `MCPBridge` in `internal/copilot/mcp.go`
    - [x] Support local/stdio and remote HTTP/SSE servers
    - [x] `RegisterMCPServers()` for SDK session config
    - [x] Common MCP server presets (filesystem, github, memory)

---

## Implementation Status
✅ Phase 1 Complete - Foundation SDK provider with streaming and tool bridge
✅ Phase 2 Complete - Brain integration with dual-path generation
✅ Phase 3 Complete - Core tools registered with Copilot SDK
✅ Phase 4 Complete - Streaming events and TUI integration
✅ Phase 5 Complete - BYOK support for OpenAI/Anthropic/Ollama
✅ Phase 6 Complete - MCP server bridge

## Build Status
```
✅ internal/copilot - builds successfully
✅ internal/brain - builds successfully
✅ internal/model - builds successfully  
✅ cmd/vibeaura - builds successfully
```

## Files Created/Modified
- `internal/copilot/provider.go` - Main SDK provider with BYOK
- `internal/copilot/bridge.go` - VibeAuracle tool → SDK tool bridge
- `internal/copilot/events.go` - Event routing for streaming
- `internal/copilot/mcp.go` - MCP server integration
- `internal/brain/brain.go` - Dual-path generation, tool registration
- `cmd/vibeaura/chat.go` - TUI streaming support
