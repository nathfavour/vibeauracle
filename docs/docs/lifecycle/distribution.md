# Distribution & Integrity

VibeAuracle uses a highly portable distribution strategy centered around a single, statically-linked Go binary.

## CI/CD Pipeline

We use GitHub Actions (`release.yml`) to manage a complex matrix of builds. On every tag (`v*`) or push to the `release` branch, the following occurs:

1.  **Cross-Compilation:** We build for 7+ targets:
    *   `linux/amd64`, `linux/arm64`
    *   `darwin/amd64`, `darwin/arm64` (macOS)
    *   `windows/amd64`, `windows/arm64`
    *   `android/arm64` (Native Termux support)
2.  **Metadata Injection:** Version strings, Git commits, and build timestamps are baked into the binary using `-ldflags`.
3.  **Checksum Generation:** A `checksums.txt` file is generated containing SHA-256 hashes of every artifact.
4.  **Rolling Releases:** The `release` branch always maintains a `latest` rolling tag, allowing users to track the "Stable-Edge" without waiting for semantic version increments.

## Strict Integrity Policy

Security is a primary concern for a tool with system-level access. VibeAuracle enforces a **Strict Integrity Policy**:

*   **Mandatory Verification:** During any binary update, the tool *must* successfully download and parse the remote `checksums.txt`. 
*   **Fail-Hard:** If the checksum file is missing, or if the downloaded binary's hash does not match, the update process is aborted immediately. 
*   **Audit Trail:** Every integrity check (success or failure) is logged to the lifecycle audit database.

## Discovery Mechanism

The tool uses a multi-layered discovery process to check for updates:
1.  **Git Discovery:** Uses `git ls-remote` to bypass GitHub API rate limits and get the latest commit hashes directly from the repository.
2.  **API Fallback:** If Git is unavailable, it falls back to the GitHub Releases API.
3.  **Metadata Comparison:** It compares the embedded `Commit` hash of the running binary against the remote target to determine if an update is truly necessary, even if versions appear the same.
