# 🏗️ Skill: Architectural Integrity

This skill ensures that VibeAuracle maintains its "Modular Monolith" structure and Hexagonal Architecture principles.

## 🎯 Objective
Prevent tight coupling between the cognitive core (`brain`) and infrastructure adapters (UI, Providers, Tools).

## 🛠️ Instructions

### 1. Hexagonal Boundaries
- **Core Logic**: Keep `internal/brain` pure. It should define interfaces for what it needs, not import specific implementations from `internal/model` or `internal/sys` unless necessary for types.
- **Dependency Inversion**: Use factories or registry patterns (see `internal/model/model.go`) to inject dependencies.

### 2. Workspace Hygiene
- When adding new capabilities, consider if they belong in an existing module or deserve a new subdirectory in `internal/`.
- Maintain `go.work` consistency.

### 3. TUI Decoupling
- The UI (`cmd/vibeaura/chat_ui.go`) should strictly be a consumer of the Brain's output. 
- Avoid putting business logic or tool execution logic directly into the TUI's `Update` function.

## ✅ Verification
- Check for circular dependencies using `go list`.
- Ensure new features are accompanied by tests in the relevant module.
