# Audit & Observability

VibeAuracle maintains a persistent record of its own lifecycle events. This is critical for debugging update failures and providing "System-Intimate" intelligence to the AI agent.

## The Lifecycle Database

All major events are stored in a JSONL (JSON Lines) file located at:
`~/.vibeauracle/audit/lifecycle.jsonl`

### Why JSONL?
*   **Append-Only:** Highly efficient and resistant to corruption during crashes.
*   **Streaming-Friendly:** Easy for AI agents or logs-processors to tail in real-time.
*   **Machine-Readable:** Perfect for structured queries without needing a heavy database engine like SQLite.

## Event Schema

Each entry in the log follows a strict schema:

| Field | Description |
| :--- | :--- |
| `timestamp` | RFC3339 time of the event. |
| `type` | The category: `install`, `update`, `rollback`, `uninstall`, `self_heal`. |
| `action` | Specific sub-action (e.g., `binary_update`, `verify_checksum`). |
| `status` | `success` or `error`. |
| `message` | A human-readable summary or the error message. |
| `version` | The semantic version or branch name. |
| `commit` | The specific 40-character Git SHA. |
| `metadata` | A flexible map for extra context (e.g., download URLs, recovery methods). |

## Usage for Debugging

If you encounter an issue with an update, the audit log is the first place to look. 

Example of a failed integrity check:
```json
{"timestamp":"2026-02-06T15:04:05Z","type":"update","action":"verify_checksum","version":"v1.2.3","commit":"abc123...","status":"error","message":"checksum mismatch: expected 123, got 456"}
```

## AI Agent Integration
The internal "Doctor" and "Self-Healing" systems use this log to determine when to trigger a recovery. If the log shows repeated failures for a specific commit SHA, the system will automatically blacklist that version and suggest a rollback.
