---
sidebar_position: 7
---

# Installation

VibeAuracle is distributed via a universal installer that tracks Git SHAs directly.

## Quick Install

### 🐧 Linux / 🍎 macOS
```bash
curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | sh
```

### 🪟 Windows
```powershell
iex (irm https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.ps1)
```

## Keeping it Fresh

VibeAuracle supports seamless, background updates. You can trigger an update manually to pull the latest version from your current branch (Stable or Beta):

```bash
vibeaura update
```

## Uninstallation

We respect your system hygiene. To remove the tool but keep your data:
```bash
vibeaura uninstall
```

To wipe **everything** (binary + secrets + config):
```bash
vibeaura uninstall --clean
```

---

For a deep dive into how we manage distribution, integrity, and self-healing updates, see the [**Lifecycle Documentation**](./lifecycle/index.md).
