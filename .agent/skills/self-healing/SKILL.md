# 🩺 Skill: Agentic Self-Healing

This skill manages the "Doctor" system that automatically detects and resolves project issues.

## 🎯 Objective
Create a resilient engineering environment where the AI proactively fixes its own errors or system inconsistencies.

## 🛠️ Instructions

### 1. Issue Detection
- Monitor the `doctor` channel for signals from various modules (Brain, TUI, Tooling).
- Implement specialized "Healers" in `internal/doctor` for common failure modes (e.g., build errors, git conflicts).

### 2. Recursive Correction
- When an execution fails (e.g., `shell_exec` returns non-zero), the Brain should invoke the `self-healing` logic to analyze the error and generate a fix.
- Ensure loop detection is active to prevent infinite retry-fail cycles.

### 3. Safety First
- Self-healing actions that involve significant filesystem changes or destructive commands MUST require user intervention/approval.

## ✅ Verification
- Test healers with simulated failures (see `internal/brain/self_healing.go`).
- Ensure the `doctor` successfully routes issues to the active agent runtime.
