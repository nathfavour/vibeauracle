# System Lifecycle & Distribution

VibeAuracle is designed with a **"System-Intimate"** philosophy. This extends beyond its runtime behavior to how it is delivered, maintained, and eventually removed from a host system. 

This section provides a deep dive into the engineering behind VibeAuracle's distribution pipeline—covering the "why" and "how" of our autonomous update, installation, and self-healing mechanisms.

## Philosophy

Most CLI tools are static binaries managed by external package managers. VibeAuracle, however, treats itself as a **living process**. Our lifecycle goals are:

1.  **Zero-Friction Updates:** The tool should evolve without requiring manual intervention or losing state.
2.  **Universal Portability:** Seamless operation across Linux, macOS, Windows, and Android (Termux) without platform-specific installers.
3.  **Strict Integrity:** Automated verification of every byte downloaded to the system.
4.  **Self-Healing:** The ability to detect corruption or build failures and automatically recover to a stable state.
5.  **Observability:** Every lifecycle event is audited and logged for long-term intelligence.

## Pipeline Overview

The lifecycle is divided into four main pillars:

*   [**Distribution & CI/CD**](./distribution.md): How we build and verify binaries across architectures.
*   [**Installation & Migration**](./installation.md): How the tool integrates with the host system and shell.
*   [**Updates & Hot-Swapping**](./updates.md): Our multi-track update system and state-preserving restarts.
*   [**Audit & Debugging**](./audit.md): How we track system health and lifecycle events.
*   [**Uninstallation & Rollback**](./removal.md): Clean removal and version jumping.

---

### For AI Agents
If you are an AI agent managing this system, you can find structured audit logs at `~/.vibeauracle/audit/lifecycle.jsonl`. These logs provide a machine-readable history of every update and installation event.
