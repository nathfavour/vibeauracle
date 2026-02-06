# Removal & Rollback

We believe that a tool's "Exit Strategy" is as important as its installation. VibeAuracle provides powerful tools to revert changes or completely remove itself from your system.

## Version Rollbacks

If a new version introduces a regression, you can use the `rollback` command:

```bash
# Rollback to the previous stable release
vibeaura rollback

# Rollback to a specific version or commit
vibeaura rollback --version v1.1.0
```

### Intelligent Rollback Behavior
*   **Auto-Update Suppression:** When you rollback, VibeAuracle automatically disables the `auto_update` flag in your config. This prevents the tool from immediately "fixing" itself by updating back to the version you just rolled away from.
*   **Manual Override:** You must run `vibeaura update` manually to re-enable the automatic update track.

## Uninstallation

To remove VibeAuracle, use the built-in `uninstall` command:

```bash
vibeaura uninstall --revert-shell
```

### What happens during uninstallation?

1.  **Binary Removal:** The tool identifies its own location on disk and deletes the binary.
2.  **Shell Cleanup:** If `--revert-shell` is passed, it searches your shell profiles (`.zshrc`, etc.) for the marker blocks and removes them.
3.  **Data Preservation:** By default, your configuration and secrets in `~/.vibeauracle` are **preserved**. This allows you to reinstall later without losing your settings.

### Complete Wipe
To remove everything, including your configurations, secrets, and audit logs:
```bash
vibeaura uninstall --clean
```

## Manual Removal
If for any reason the `uninstall` command is unavailable, you can manually remove the tool by:
1. Deleting the `~/.local/bin/vibeaura` binary.
2. Removing the `~/.vibeauracle` directory.
3. Deleting the marker-wrapped lines in your shell configuration files.
