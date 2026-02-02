---
sidebar_position: 11
---

# External Integrations (UDS/IPC)

VibeAuracle provides a high-performance Unix Domain Socket (UDS) for external tools to integrate with the system. This allows other applications, shell scripts, or IDE plugins to communicate with the VibeAuracle brain.

## 🔌 Connection Details

- **Socket Path**: `~/.vibeauracle/vibeaura.sock`
- **Protocol**: Line-delimited JSON
- **Availability**: The daemon starts automatically whenever the TUI launches.

## 📜 Message Format

Messages follow a JSON-RPC-like structure:

```json
{
  "type": "request",
  "method": "query",
  "id": "unique-id-123",
  "payload": {
    "content": "Your prompt here",
    "intent": "crud"
  }
}
```

### Response Format

```json
{
  "type": "response",
  "id": "unique-id-123",
  "payload": {
    "content": "The brain's response"
  }
}
```

## 🛡️ Security & Modes

By default, VibeAuracle prioritizes security. The brain behaves differently based on the **Intent**:

- **`chat` / `ask`**: Default for casual conversation. **Tool execution is disabled.** The brain can talk but cannot touch your files or system.
- **`crud` / `do`**: Required for actions. Allows the brain to use system tools (read/write files, run commands).
- **`plan`**: Allows architectural analysis and structured breakdown.

When using the IPC `query` method, you can optionally specify the `"intent"`. If omitted, the brain will attempt to auto-classify the request. For high-stakes automation, we recommend explicitly passing `"intent": "crud"`.

## 🛠️ Supported Methods

| Method | Description |
|--------|-------------|
| `ping` | Check if the daemon is alive. |
| `status` | Get a real-time system resource snapshot. |
| `config` | Retrieve the current system configuration. |
| `query` | Send a prompt to the brain for processing. |

## 💻 Example: Using `socat`

You can interact with the socket directly from your terminal using `socat`:

```bash
# Send a ping
echo '{"type":"request","method":"ping","id":"1","payload":{}}' | socat - UNIX-CONNECT:~/.vibeauracle/vibeaura.sock
```

## 🧠 Why use IPC?

- **IDE Integration**: Build custom plugins for VS Code, NeoVim, or JetBrains that leverage VibeAuracle's system-intimate knowledge.
- **Workflow Automation**: Trigger AI actions from shell scripts or cron jobs.
- **Custom TUIs**: Build specialized interfaces that talk to the same background brain.
