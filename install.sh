#!/bin/bash

# vibe auracle Universal Installer
# Usage: curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | bash

set -e

REPO="nathfavour/vibeauracle"
GITHUB_URL="https://github.com/$REPO"

# Detect OS
OS="$(uname -s | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

if [ "$OS" = "darwin" ]; then
    OS="darwin"
elif [ "$OS" = "linux" ]; then
    # Check for Android (Termux)
    if [ -n "$TERMUX_VERSION" ] || [ -d "/data/data/com.termux" ]; then
        OS="android"
    else
        OS="linux"
    fi
else
    echo "Unsupported OS: $OS"
    exit 1
fi

BINARY_NAME="vibeaura-${OS}-${ARCH}"
if [ "$OS" = "windows" ]; then
    BINARY_NAME+=".exe"
fi

echo "Detected Platform: $OS/$ARCH"

# Get latest release tag
echo "Fetching release metadata..."
LATEST_TAG=""
if command -v git >/dev/null 2>&1; then
    ALL_TAGS=$(git ls-remote --tags "https://github.com/$REPO.git" | cut -d/ -f3)
    if echo "$ALL_TAGS" | grep -q "^latest$"; then
        LATEST_TAG="latest"
    else
        LATEST_TAG=$(echo "$ALL_TAGS" | grep -E "^v[0-9]" | sort -V | tail -n 1)
    fi
fi

if [ -z "$LATEST_TAG" ]; then
    TAG_DATA=$(curl -fsSL "https://api.github.com/repos/$REPO/releases/latest" || true)
    LATEST_TAG=$(echo "$TAG_DATA" | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4)
fi

if [ -z "$LATEST_TAG" ]; then
    echo "Error: Failed to fetch latest release."
    exit 1
fi

DOWNLOAD_URL="$GITHUB_URL/releases/download/$LATEST_TAG/$BINARY_NAME"
CHECKSUM_URL="$GITHUB_URL/releases/download/$LATEST_TAG/checksums.txt"

echo "Downloading $BINARY_NAME ($LATEST_TAG)..."
TMP_BIN=$(mktemp)
TMP_CHECKSUM=$(mktemp)

if command -v curl >/dev/null 2>&1; then
    curl -L "$DOWNLOAD_URL" -o "$TMP_BIN"
    curl -L "$CHECKSUM_URL" -o "$TMP_CHECKSUM"
elif command -v wget >/dev/null 2>&1; then
    wget -qO "$TMP_BIN" "$DOWNLOAD_URL"
    wget -qO "$TMP_CHECKSUM" "$CHECKSUM_URL"
else
    echo "Error: curl or wget is required."
    exit 1
fi

# --- Verify Integrity ---
echo "Verifying integrity..."
if command -v sha256sum >/dev/null 2>&1; then
    # We need to extract the line for our binary from checksums.txt and check it
    EXPECTED_SHA=$(grep "$BINARY_NAME" "$TMP_CHECKSUM" | cut -d' ' -f1)
    ACTUAL_SHA=$(sha256sum "$TMP_BIN" | cut -d' ' -f1)
    if [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
        echo "Error: Checksum mismatch! The downloaded binary may be corrupted or tampered with."
        rm -f "$TMP_BIN" "$TMP_CHECKSUM"
        exit 1
    fi
elif command -v shasum >/dev/null 2>&1; then
    EXPECTED_SHA=$(grep "$BINARY_NAME" "$TMP_CHECKSUM" | cut -d' ' -f1)
    ACTUAL_SHA=$(shasum -a 256 "$TMP_BIN" | cut -d' ' -f1)
    if [ "$EXPECTED_SHA" != "$ACTUAL_SHA" ]; then
        echo "Error: Checksum mismatch!"
        rm -f "$TMP_BIN" "$TMP_CHECKSUM"
        exit 1
    fi
else
    echo "Warning: sha256sum or shasum not found. Skipping integrity check."
fi

chmod +x "$TMP_BIN"

# --- Hand over to the Smart Binary ---
echo "Finalizing installation..."
"$TMP_BIN" system-install
rm -f "$TMP_BIN" "$TMP_CHECKSUM"

echo "✅ vibe auracle has been successfully installed!"
echo "👉 Run 'vibeaura version' to verify."