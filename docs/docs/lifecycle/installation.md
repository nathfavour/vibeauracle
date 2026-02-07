# Installation & System Integration

VibeAuracle doesn't just "sit" on your disk; it integrates with your environment to provide a seamless engineering experience.

## The Bootstrap Flow

The standard installation starts with a one-liner:
```bash
curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | bash
```

### 1. Platform Detection
The script detects the OS and Architecture, then downloads the specific binary for the host. 

### 2. Binary Hand-off
The script executes the binary with the `system-install` flag. At this point, the binary takes over its own installation. This ensures that the installation logic is always in sync with the version of the tool being installed.

## System Migration (`system-install`)

When the binary runs its install routine, it performs the following:

*   **Self-Migration:** It copies itself to `~/.local/bin/vibeaura`.
*   **Shadow Cleanup:** It searches the `PATH` for other instances of `vibeaura` in the user's home directory and removes them to prevent version conflicts.
*   **PATH Automation:** It detects the user's shell (Bash, Zsh, or Fish) and adds `~/.local/bin` to the `PATH` if it's missing.

## Shell Integration

VibeAuracle is a "good citizen" of the shell. When modifying configuration files (like `.zshrc`), it uses idempotent markers:

```bash
# >>> vibe auracle initialize >>>
export PATH="$PATH:/home/user/.local/bin"
# <<< vibe auracle initialize <<<
```

### Why this approach?
*   **No Sudo:** By default, VibeAuracle installs to the user's home directory. This avoids the need for root permissions and keeps the tool isolated to the developer's environment.
*   **Clean Reverts:** The use of markers allows the `uninstall` command to remove these lines without disturbing the rest of the user's config.
