# ⚡ Skill: System Intimacy

This skill focuses on deepening the agent's awareness of the hardware and operating system environment.

## 🎯 Objective
Enable the agent to make intelligent decisions based on real-world system constraints (CPU, VRAM, CWD, OS).

## 🛠️ Instructions

### 1. Environment Awareness
- Use `sys.Monitor` to get real-time snapshots.
- When generating plans, always consider the user's OS (Arch Linux/KDE focus) and available resources.

### 2. Filesystem Abstraction
- Always use `sys.FS` for file operations to ensure potential sandboxing and consistency.
- Prefer `sys_patch` over full file rewrites for large files to save tokens and minimize system I/O.

### 3. Resource-Aware Inference
- If VRAM is low, the agent should proactively suggest switching to a smaller local model or a cloud provider.

## ✅ Verification
- Verify that system snapshots are correctly integrated into the prompt context in `internal/brain/brain.go`.
- Ensure tool executions like `shell_exec` are logged and monitored for resource spikes.
