# vibeauracle: Strategic Roadmap

## Gemini CLI Integration (Current Focus)
- [x] **Auth & Credentials**
    - [x] Implement `internal/auth/gemini`: Read `~/.gemini/oauth_creds.json`.
    - [x] Implement OAuth2 Refresh Token flow using hardcoded CLI Client ID/Secret.
    - [x] Support `GOOGLE_CLOUD_PROJECT` auto-detection from environment.

- [x] **Core Provider implementation**
    - [x] Implement `internal/model/gemini`: `Generate`, `Stream`, `ListModels`, `Embed`.
    - [x] Implement SSE (Server-Sent Events) parser for Code Assist streaming API.
    - [x] Implement request/response converters to match Code Assist JSON protocol.

- [x] **Dynamic Model Discovery**
    - [x] Implement `LoadCodeAssist` call to verify tier and project access.
    - [x] Implement `RetrieveUserQuota` call to dynamically fetch available models.
    - [ ] Add `vibeaura models --gemini` to list discoverable models.

- [ ] **Advanced Features**
    - [ ] Implement Tool Bridge (Map VibeAuracle tools to Gemini Function Declarations).
    - [ ] Implement "Classifier" Routing: Auto-switch between Flash/Pro based on task complexity.
    - [ ] Support Gemini 3 specific features (multimodal, extended thinking).

- [ ] **Advanced Features**
    - [ ] Implement Tool Bridge (Map VibeAuracle tools to Gemini Function Declarations).
    - [ ] Implement "Classifier" Routing: Auto-switch between Flash/Pro based on task complexity.
    - [ ] Support Gemini 3 specific features (multimodal, extended thinking).

## Upcoming Phases
- [ ] **Fluid UI & Polish**
    - [ ] Remove hard borders/sections in the sidebar; implement weighted vertical flow.
    - [ ] Add "Heartbeat" animations for thinking/indexing states.
    - [ ] Benchmarking Widget: Display last request duration, token usage, and cost.

- [ ] **Expansion**
    - [ ] Shared Sessions: Support for `/share` and `/connect` via browser/TUI.
    - [ ] MCP Server Management: Deep integration with external MCP servers.
