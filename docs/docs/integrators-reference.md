---
sidebar_position: 13
---

# Integrator's Reference

This document provides the technical specifications required for external applications (IDEs, CLI tools, dashboards) to integrate with the VibeAuracle ecosystem.

## 🔌 Unix Domain Socket (IPC)

The primary integration path is the UDS located at `~/.vibeauracle/vibeaura.sock`.

### Protocol Specification
- **Format**: Line-Delimited JSON (LDJSON).
- **Encoding**: UTF-8.
- **Mode**: Request/Response (Synchronous over the stream).

### Message Schemas

#### Request
```json
{
  "type": "request",
  "method": "string",
  "id": "string",
  "payload": {}
}
```
| Field | Description |
|---|---|
| `method` | `ping`, `query`, `status`, `config` |
| `id` | Correlation ID (returned in response) |
| `payload` | Method-specific object |

#### Response
```json
{
  "type": "response",
  "id": "string",
  "payload": {}
}
```

#### Error
```json
{
  "type": "error",
  "id": "string",
  "payload": {
    "message": "Human readable error"
  }
}
```

### Method Payloads

#### `query`
Triggers the AI Brain.
- **Request Payload**: `{"content": "string", "intent": "ask|plan|crud"}`
- **Response Payload**: `{"content": "string"}`

#### `status`
- **Response Payload**: 
  ```json
  {
    "CPUUsage": 12.5,
    "MemoryUsage": 45.2,
    "WorkingDir": "/absolute/path"
  }
  ```

---

## 🗄️ Database Schema (SQLite)

VibeAuracle uses SQLite for persistence, located at `~/.vibeauracle/vibe.db`.

### Table: `app_state`
Used for session history and TUI state.
- **`id`**: `chat_session:{dir_hash}` (dir_hash is the first 8 chars of SHA256 of the CWD).
- **`data`**: JSON blob (see Session Schema below).

### Table: `project_knowledge`
Architectural insights discovered by the Brain.
- **`root_path`**: Primary Key.
- **`git_sha`**: Last indexed commit.
- **`logical_map`**: JSON mapping of entrypoints and architecture.

---

## 📑 Session JSON Schema

When reading from `app_state`, the `data` field follows this structure:

```json
{
  "messages": ["string"],
  "input": "string",
  "prompt_history": ["string"],
  "show_sidebar": true
}
```

### Message Formatting
Messages in the `messages` array use specific prefixes:
- **`User `**: Plain text question.
- **`VibeAuracle: `**: AI response (Markdown).
- **`BLOCK:`**: Pre-rendered agent process block (contains ANSI and Headers).

---

## 🛠️ Headless Integration (CLI)

External tools can use the binary directly for non-interactive tasks.

### Stream Analysis
```bash
cat file.txt | vibeaura direct "Summary" --non-interactive
```
- **Exit Codes**: `0` on success, `1` on error.
- **Output**: The AI response is sent to `stdout`, status updates (Thinking/Executing) are sent to `stderr` if `--verbose` is used.

---

## 🔒 Security
External tools must respect the `0600` permissions on the socket and database. VibeAuracle leverages the existing `gh` CLI token; integrators should not attempt to capture or store this token independently but instead rely on VibeAuracle's `vault` for plugin-specific secrets.
