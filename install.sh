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

# --- Source & Config Detection ---
BUILD_FROM_SOURCE=false
CONFIG_FILE="$HOME/.vibeauracle/config.yaml"
if [ -f "$CONFIG_FILE" ]; then
    # Use a more robust grep that handles whitespace and both build_from_source and beta
    if grep -qE "build_from_source: *true" "$CONFIG_FILE" || grep -qE "beta: *true" "$CONFIG_FILE"; then
        BUILD_FROM_SOURCE=true
    fi
fi

# Detect if we are inside the source tree
if [ -f "go.work" ] && [ -d "cmd/vibeaura" ]; then
    BUILD_FROM_SOURCE=true
    LOCAL_SOURCE=true
fi

# Detect if existing installation is a source build
if command -v vibeaura >/dev/null 2>&1; then
    EXISTING_VIBE=$(command -v vibeaura)
    V_OUT=$("$EXISTING_VIBE" version 2>/dev/null || true)
    if echo "$V_OUT" | grep -qE "Version *: *(master|main|release|dev)"; then
        BUILD_FROM_SOURCE=true
    fi
fi

if [ "$BUILD_FROM_SOURCE" = true ]; then
    echo "Existing configuration or environment suggests building from source."
    if command -v go >/dev/null 2>&1; then
        if [ "$LOCAL_SOURCE" = true ]; then
            echo "Building from current directory..."
            COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
            DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
            BRANCH=$(git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "master")
            GOTOOLCHAIN=local go build -ldflags "-s -w -X main.Version=$BRANCH -X main.Commit=$COMMIT -X main.BuildDate=$DATE" -o vibeaura ./cmd/vibeaura
            # We continue with the rest of the script to install the binary we just built
        else
            if [ -n "$EXISTING_VIBE" ]; then
                echo "Handing over to 'vibeaura update' for source-based update..."
                "$EXISTING_VIBE" update
                exit 0
            else
                echo "Building from remote source..."
                TMP_SRC=$(mktemp -d)
                git clone --depth 1 "https://github.com/$REPO.git" "$TMP_SRC"
                (
                    cd "$TMP_SRC"
                    COMMIT=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
                    DATE=$(date -u +%Y-%m-%dT%H:%M:%SZ)
                    BRANCH="master"
                    GOTOOLCHAIN=local go build -ldflags "-s -w -X main.Version=$BRANCH -X main.Commit=$COMMIT -X main.BuildDate=$DATE" -o "$OLDPWD/vibeaura" ./cmd/vibeaura
                )
                rm -rf "$TMP_SRC"
                # We continue with the rest of the script to install the binary we just built
            fi
        fi
    else
        echo "Warning: Source build preferred but 'go' not found. Falling back to binary release."
    fi
fi

if [ ! -f "vibeaura" ]; then
    # Get latest release tag
    # We prefer git ls-remote to avoid GitHub API rate limits (403 errors).
    # If git is not available, we fallback to the API.
    echo "Fetching release metadata..."

    LATEST_TAG=""
    if command -v git >/dev/null 2>&1; then
        # Try to get the latest tag (preferring 'latest' rolling tag or newest semver)
        ALL_TAGS=$(git ls-remote --tags "https://github.com/$REPO.git" | cut -d/ -f3)
        if echo "$ALL_TAGS" | grep -q "^latest$"; then
            LATEST_TAG="latest"
        else
            LATEST_TAG=$(echo "$ALL_TAGS" | grep -E "^v[0-9]" | sort -V | tail -n 1)
        fi
    fi

    if [ -z "$LATEST_TAG" ]; then
        # Fallback to API if git failed or wasn't found
        TMP_ERR=$(mktemp)
        TAG_DATA=$(curl -fsSL -H "User-Agent: vibe-auracle-installer" "https://api.github.com/repos/$REPO/releases" 2>"$TMP_ERR" || true)

        if [ -n "$TAG_DATA" ]; then
            LATEST_TAG=$(echo "$TAG_DATA" | grep -oE '"tag_name": *"[^"]+"' | head -n 1 | cut -d'"' -f4)

            # If we found tags but it wasn't the 'latest' tag specifically, 
            # try to see if 'latest' exists in the list for stability
            if [ "$LATEST_TAG" != "latest" ]; then
                STABLE_TAG=$(echo "$TAG_DATA" | grep -oE '"tag_name": *"latest"' | head -n 1 | cut -d'"' -f4)
                if [ -n "$STABLE_TAG" ]; then
                    LATEST_TAG="$STABLE_TAG"
                fi
            fi
        fi

        if [ -z "$LATEST_TAG" ]; then
            echo "Error: Failed to fetch releases from GitHub."
            if [ -f "$TMP_ERR" ] && [ -s "$TMP_ERR" ]; then
                cat "$TMP_ERR"
            fi
            rm -f "$TMP_ERR"
            exit 1
        fi
        rm -f "$TMP_ERR"
    fi

    echo "Resolved version: $LATEST_TAG"

    # Check if vibeaura is already installed and up-to-date
    if [ -z "$EXISTING_VIBE" ]; then
        # Strict priority: ~/.local/bin
        if [ -x "$HOME/.local/bin/vibeaura" ]; then
            EXISTING_VIBE="$HOME/.local/bin/vibeaura"
        elif command -v vibeaura >/dev/null 2>&1; then
            EXISTING_VIBE=$(command -v vibeaura)
        else
            # Check other common locations if not in PATH and not in ~/.local/bin
            [ -x "/usr/local/bin/vibeaura" ] && EXISTING_VIBE="/usr/local/bin/vibeaura"
            [ -x "$HOME/bin/vibeaura" ] && EXISTING_VIBE="$HOME/bin/vibeaura"
            [ -x "$HOME/go/bin/vibeaura" ] && EXISTING_VIBE="$HOME/go/bin/vibeaura"
        fi
    fi

    if [ -n "$EXISTING_VIBE" ]; then
        LOCAL_VERSION=$("$EXISTING_VIBE" version | grep "Version" | awk '{print $3}' || true)
        LOCAL_COMMIT=$("$EXISTING_VIBE" version | grep "Commit" | awk '{print $3}' || true)
        
        # Resolve the SHA of the latest tag to be sure
        LATEST_SHA=""
        if [ -n "$LATEST_TAG" ] && command -v git >/dev/null 2>&1; then
            LATEST_SHA=$(git ls-remote --tags "https://github.com/$REPO.git" | grep "refs/tags/$LATEST_TAG$" | awk '{print $1}' || true)
        fi

        # If the local version matches the latest tag, OR the local commit matches the latest SHA, we can skip
        if { [ -n "$LOCAL_VERSION" ] && [ "$LOCAL_VERSION" = "$LATEST_TAG" ]; } || \
           { [ -n "$LOCAL_COMMIT" ] && [ -n "$LATEST_SHA" ] && [ "$LOCAL_COMMIT" = "$LATEST_SHA" ]; }; then
            SHORT_COMMIT=$(echo "$LOCAL_COMMIT" | cut -c1-7)
            echo "Vibe Auracle is already up to date ($LATEST_TAG / $SHORT_COMMIT)."
            exit 0
        fi
    fi

    DOWNLOAD_URL="$GITHUB_URL/releases/download/$LATEST_TAG/$BINARY_NAME"

    echo "Downloading $BINARY_NAME ($LATEST_TAG)..."
    if command -v curl >/dev/null 2>&1; then
        curl -L "$DOWNLOAD_URL" -o vibeaura
    elif command -v wget >/dev/null 2>&1; then
        wget -qO vibeaura "$DOWNLOAD_URL"
    else
        echo "Error: curl or wget is required."
        exit 1
    fi

    chmod +x vibeaura
fi

# --- Install Directory Discovery ---
if [ -n "$EXISTING_VIBE" ]; then
    INSTALL_DIR=$(dirname "$EXISTING_VIBE")
fi

# Install binary
if [ -n "$INSTALL_DIR" ]; then
    # Use existing directory discovered above
    true
elif [ "$OS" = "android" ]; then
    INSTALL_DIR="$HOME/bin"
else
    # Absolute priority: ~/.local/bin
    INSTALL_DIR="$HOME/.local/bin"
fi

if [ ! -d "$INSTALL_DIR" ]; then
    mkdir -p "$INSTALL_DIR" 2>/dev/null || true
fi

if [ -w "$INSTALL_DIR" ] || [ ! -e "$INSTALL_DIR" ]; then
    mv vibeaura "$INSTALL_DIR/vibeaura" 2>/dev/null || sudo mv vibeaura "$INSTALL_DIR/vibeaura"
else
    echo "Requesting sudo to install to $INSTALL_DIR..."
    sudo mv vibeaura "$INSTALL_DIR/vibeaura"
fi

chmod +x "$INSTALL_DIR/vibeaura"
echo "Successfully installed vibe auracle to $INSTALL_DIR/vibeaura"

# Auto-add to PATH
SHELL_RC=""
if [ -n "$ZSH_VERSION" ]; then
    SHELL_RC="$HOME/.zshrc"
elif [ -n "$BASH_VERSION" ]; then
    SHELL_RC="$HOME/.bashrc"
else
    # Fallback to checking existence
    [ -f "$HOME/.zshrc" ] && SHELL_RC="$HOME/.zshrc"
    [ -f "$HOME/.bashrc" ] && [ -z "$SHELL_RC" ] && SHELL_RC="$HOME/.bashrc"
fi

if [ -n "$SHELL_RC" ]; then
    if ! grep -q "$INSTALL_DIR" "$SHELL_RC" 2>/dev/null; then
        echo "" >> "$SHELL_RC"
        echo "# vibe auracle path" >> "$SHELL_RC"
        echo "export PATH=\"\$PATH:$INSTALL_DIR\"" >> "$SHELL_RC"
        echo "Added $INSTALL_DIR to $SHELL_RC"
    fi
    echo "Please restart your shell or run: source $SHELL_RC"
fi

export PATH="$PATH:$INSTALL_DIR"
"$INSTALL_DIR/vibeaura" version || true
