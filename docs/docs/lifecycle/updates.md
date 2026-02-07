# Updates & Hot-Swapping

VibeAuracle features one of the most advanced update systems for a CLI tool, allowing it to evolve without breaking the developer's flow.

## Multi-Track Updates

Users can choose their stability level:
1.  **Stable (Binary):** Downloads pre-built binaries from the `release` branch. (Default)
2.  **Beta (Source):** Tracks the `master` branch and builds from source locally.
3.  **Source Track:** Users can opt to always build from source to ensure they are running code optimized for their specific machine environment.

## The Hot-Swap Mechanism

This is VibeAuracle's "killer feature" for productivity. In the TUI mode:

1.  **Background Check:** The tool periodically checks for updates in the background.
2.  **Shadow Download:** When an update is found, it is downloaded to a temporary location and verified.
3.  **State Serialization:** If the user approves the update (or if auto-update is on), the tool serializes its current state:
    *   Command history
    *   Active input buffer
    *   Session context
4.  **Process Exec:** It uses `syscall.Exec` to replace the current process image with the new binary.
5.  **Restoration:** The new version starts, detects the `--resume-state` flag, loads the JSON state, and restores the UI exactly where the user left off.

## Self-Healing Build Pipeline

Building from source can be brittle, especially on mobile environments like Termux. VibeAuracle includes an autonomous recovery system:

*   **Dependency Injection:** If a build fails on Termux, the tool checks if the Go toolchain is outdated and attempts to run `pkg upgrade golang` automatically.
*   **Recursive Recovery:** If three consecutive updates fail, the tool enters a "Self-Healing" mode where it deletes its own binary and triggers a fresh installation from the official release script to restore stability.

## Audit Logs
Every update attempt, success, or failure is recorded in the Lifecycle Audit Log (`~/.vibeauracle/audit/lifecycle.jsonl`).
