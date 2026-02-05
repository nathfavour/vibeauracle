---
sidebar_position: 11
---

# External Integrations (UDS/IPC)

VibeAuracle is designed to be the "cognitive kernel" of your development environment. To enable this, it exposes a high-performance **Unix Domain Socket (UDS)** that allows external tools—such as IDE plugins (VS Code, NeoVim), custom dashboard, or orchestration scripts—to leverage the Brain's system-intimate capabilities.

## 🔌 Connection Specifications

- **Socket Path**: `~/.vibeauracle/vibeaura.sock`
- **Protocol**: Line-Delimited JSON (LDJSON)
- **Lifecycle**: The UDS server is managed by `vibe-daemon`. It starts automatically when you run `vibeaura` (TUI) and remains active as long as the session is open.

## 📜 Communication Protocol

VibeAuracle uses a simplified JSON-RPC 2.0-like protocol. Every message sent to the socket must be a single line of JSON.

### Request Schema

```json
{
  "type": "request",
  "method": "METHOD_NAME",
  "id": "CORRELATION_ID",
  "payload": { ... }
}
```

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Must be `"request"`. |
| `method`| string | One of `ping`, `query`, `status`, or `config`. |
| `id` | string | A client-generated unique identifier for tracking the response. |
| `payload`| object | Method-specific arguments. |

### Response Schema

```json
{
  "type": "response",
  "id": "CORRELATION_ID",
  "payload": { ... }
}
```

## 🛠️ Supported Methods

### 1. `query` (The Main Brain Entrypoint)
Send a prompt to the brain for processing. This triggers the full "Perceive-Plan-Execute-Reflect" loop.

**Payload:**
```json
{
  "content": "Analyze the current project and find memory leaks",
  "intent": "crud" 
}
```

- **`intent` (Optional)**: Can be `ask`, `plan`, or `crud`.
    - `ask`: Casual Q&A. **Tool execution disabled.**
    - `plan`: Architectural analysis.
    - `crud`: Full system intimacy. **Required for the brain to modify files or run commands.**

### 2. `status`
Returns a real-time snapshot of the system as perceived by the Brain.

**Response Payload:**
```json
{
  "CPUUsage": 12.5,
  "MemoryUsage": 45.2,
  "WorkingDir": "/home/user/project"
}
```

### 3. `config`
Returns the current active configuration (AI providers, active models, agent modes).

### 4. `ping`
Returns `{"status": "pong"}`. Used for health checks.

## 🛡️ Security & Access Control

The UDS socket is created with `0600` permissions (read/write only by the owner). 

**Important:** Any tool with access to the socket can effectively execute commands on your system if they use the `crud` intent. We recommend that external tools:
1. Use a unique `id` for every request.
2. Default to `ask` intent unless the user explicitly requests an action.
3. Handle the "Intervention" responses (not yet fully implemented in UDS, coming soon) where the Brain asks for permission.

## 💻 Integration Examples

### Python (Simple Client)
```python
import socket
import json
import os

socket_path = os.path.expanduser("~/.vibeauracle/vibeaura.sock")
client = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
client.connect(socket_path)

req = {
    "type": "request",
    "method": "status",
    "id": "health-check",
    "payload": {}
}

client.sendall(json.dumps(req).encode() + b"\n")
response = client.recv(4096)
print(json.loads(response))
```

### Shell (using `nc` or `socat`)
```bash
echo '{"type":"request","method":"query","id":"1","payload":{"content":"what is my CWD?"}}' | nc -U ~/.vibeauracle/vibeaura.sock
```

## 🚀 Building on top of VibeAuracle

If you are building an IDE extension, you can use the UDS to:
- **Project Awareness**: Use `status` to keep your plugin synced with the Brain's current working directory.
- **Background Tasks**: Trigger `query` with `intent: crud` to perform background refactoring or test generation.
- **Unified Config**: Use `config` to ensure your tool uses the same AI model settings as the TUI.
