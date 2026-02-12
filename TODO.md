# vibeauracle: Smart Sidebar & Intelligence Roadmap

- [x] **Phase 1: The Foundation (Real-Time & Shortcuts)**
    - [x] Fix `autocommiter` vs `autocommitter` spelling bug in `scm_tools.go`.
    - [x] Implement `Ctrl+A` shortcut for smart committing (CLI first, then internal fallback).
    - [x] Integrate `internal/watcher` into the main TUI loop to trigger instant UI refreshes.
    - [x] Create `SidebarManager` to handle fluid component rendering and real estate allocation.

- [x] **Phase 2: Live Intelligence (The "#" Aura)**
    - [x] Implement live `#` parser in the input field (detects files + line ranges).
    - [x] Build the "Pre-Opening" system: sidebar previews files/lines as the user types.
    - [x] Implement Auto-Scrolling and Syntax Highlighting for targeted ranges.
    - [x] Develop the "Focus Decay" algorithm to automatically manage sidebar real estate.

- [x] **Phase 3: Deep Context (Project Intimacy)**
    - [x] Implement "TODO Deep Scanner": parse `TODO.md` and `VIBES.md` to show pending tasks in idle states.
    - [x] Hook into `sys_read_file` and `sys_patch` to auto-focus the sidebar during agent turns.
    - [x] Implement "Command Snooping": reflect `ls`, `grep`, or `find` outputs in the sidebar explorer.
    - [x] Dynamic "Empty State" system: show pro-tips, project vitals, or "Agent Mood" when idle.

- [ ] **Phase 4: Fluid UI & Polish**
    - [ ] Remove hard borders/sections in the sidebar; implement weighted vertical flow.
    - [ ] Add "Heartbeat" animations for thinking/indexing states.
    - [ ] Benchmarking Widget: Display last request duration, token usage, and cost in a beautiful way.
