package main

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathfavour/vibeauracle/internal/audit"
	"github.com/nathfavour/vibeauracle/sys"
)

func updateFromSource(branch string, cm *sys.ConfigManager) (bool, error) {
	if err := checkDependencies(); err != nil {
		return false, err
	}

	cfg, err := cm.Load()
	if err != nil {
		return false, fmt.Errorf("loading config: %w", err)
	}
	verbose := cfg.Update.Verbose

	sourceRoot := cm.GetDataPath(filepath.Join("source", branch))
	if err := os.MkdirAll(filepath.Dir(sourceRoot), 0755); err != nil {
		return false, fmt.Errorf("creating source directory: %w", err)
	}

	if _, err := os.Stat(filepath.Join(sourceRoot, ".git")); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("Cloning %s branch to %s...\n", branch, sourceRoot)
		}
		cloneCmd := exec.Command("git", "clone", "-b", branch, "https://github.com/"+repo+".git", sourceRoot)
		if verbose {
			cloneCmd.Stdout = os.Stdout
			cloneCmd.Stderr = os.Stderr
		}
		if err := cloneCmd.Run(); err != nil {
			os.RemoveAll(sourceRoot)
			return false, fmt.Errorf("cloning repo: %w", err)
		}
	} else {
		// Robustness: Clear potential git locks that cause exit status 128
		os.Remove(filepath.Join(sourceRoot, ".git", "index.lock"))
		os.Remove(filepath.Join(sourceRoot, ".git", "refs", "heads", branch+".lock"))

		if verbose {
			fmt.Printf("Fetching updates for %s...\n", branch)
		}
		fetchCmd := exec.Command("git", "-C", sourceRoot, "fetch", "--prune", "origin", branch)
		if err := fetchCmd.Run(); err != nil {
			return false, fmt.Errorf("fetching updates: %w", err)
		}

		// Get remote SHA
		remoteSHA, err := getBranchCommitSHA(branch)
		if err != nil {
			return false, fmt.Errorf("getting remote SHA: %w", err)
		}

		// Check if we already have this commit
		if remoteSHA == Commit {
			return false, nil
		}

		// Check if this commit previously failed
		for _, failed := range cfg.Update.FailedCommits {
			if failed == remoteSHA {
				return false, nil
			}
		}

		if verbose {
			fmt.Printf("Updating local source in %s...\n", sourceRoot)
		}
		// Use reset --hard FETCH_HEAD instead of pull to handle diverged branches gracefully in managed source
		resetCmd := exec.Command("git", "-C", sourceRoot, "reset", "--hard", "FETCH_HEAD")
		if verbose {
			resetCmd.Stdout = os.Stdout
			resetCmd.Stderr = os.Stderr
		}
		if err := resetCmd.Run(); err != nil {
			return false, fmt.Errorf("resetting to remote: %w", err)
		}

		// Also clean untracked files to avoid build conflicts
		exec.Command("git", "-C", sourceRoot, "clean", "-fd").Run()
	}
	return buildAndInstallFromSource(sourceRoot, branch, cm)
}

func buildAndInstallFromSource(sourceRoot, branch string, cm *sys.ConfigManager) (bool, error) {
	if err := checkDependencies(); err != nil {
		return false, err
	}

	cfg, err := cm.Load()
	if err != nil {
		return false, err
	}
	verbose := cfg.Update.Verbose

	if verbose {
		fmt.Println("Building from source...")
	}

	// Get current commit SHA for the local build
	commitCmd := exec.Command("git", "-C", sourceRoot, "rev-parse", "HEAD")
	commitSHABytes, _ := commitCmd.Output()
	localCommit := strings.TrimSpace(string(commitSHABytes))

	// Final check: if the localCommit we just pulled/checked out matches current Commit, no update needed.
	if localCommit == Commit {
		return false, nil
	}

	buildDate := time.Now().UTC().Format(time.RFC3339)
	ldflags := fmt.Sprintf("-s -w -X main.Version=%s -X main.Commit=%s -X main.BuildDate=%s", branch, localCommit, buildDate)

	buildOut := filepath.Join(sourceRoot, "vibeaura_new")
	buildCmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", buildOut, "./cmd/vibeaura")
	buildCmd.Dir = sourceRoot

	// Force Go to use the locally installed toolchain and avoid automatic downloads
	// which often fail on mobile/Termux.
	buildCmd.Env = append(os.Environ(), "GOTOOLCHAIN=local")

	if verbose {
		buildCmd.Stdout = os.Stdout
		buildCmd.Stderr = os.Stderr
	}

	if err := buildCmd.Run(); err != nil {
		goos, _ := getPlatform()
		if goos == "android" {
			fmt.Println("\n🛠️  Build failed. Attempting to upgrade Go toolchain automatically...")
			upgradeCmd := exec.Command("pkg", "upgrade", "golang", "-y")
			upgradeCmd.Stdout = os.Stdout
			upgradeCmd.Stderr = os.Stderr
			if err := upgradeCmd.Run(); err == nil {
				fmt.Println("✅ Go upgraded. Retrying build...")
				if err := buildCmd.Run(); err == nil {
					if !verbose {
						fmt.Println("DONE")
					}
					exePath, _ := os.Executable()
					if err := installBinary(buildOut, exePath); err != nil {
						audit.LogFailure(cfg.DataDir, audit.EventUpdate, "source_install", branch, localCommit, err.Error(), nil)
						return false, err
					}
					audit.LogSuccess(cfg.DataDir, audit.EventUpdate, "source_update", branch, localCommit, "successfully built and installed from source (after auto-upgrade)", nil)
					return true, nil
				}
			}
		}

		if verbose {
			fmt.Println("\n❌ Build failed! This usually happens if your installed Go version is older than the one required by the project.")
			if goos == "android" {
				fmt.Println("👉 Try running: pkg upgrade golang (on Termux)")
			} else {
				fmt.Println("👉 Try updating Go on your desktop.")
			}
		}
		commitCmd := exec.Command("git", "-C", sourceRoot, "rev-parse", "HEAD")
		if out, err := commitCmd.Output(); err == nil {
			failedSHA := strings.TrimSpace(string(out))
			cfg, err := cm.Load()
			if err == nil {
				cfg.Update.FailedCommits = append(cfg.Update.FailedCommits, failedSHA)
				cm.Save(cfg)
				audit.LogFailure(cfg.DataDir, audit.EventUpdate, "source_build", branch, failedSHA, "build failed", nil)
			}
		}
		return false, fmt.Errorf("building from source: %w", err)
	}

	exePath, _ := os.Executable()
	if err := installBinary(buildOut, exePath); err != nil {
		audit.LogFailure(cfg.DataDir, audit.EventUpdate, "source_install", branch, localCommit, err.Error(), nil)
		return false, err
	}

	audit.LogSuccess(cfg.DataDir, audit.EventUpdate, "source_update", branch, localCommit, "successfully built and installed from source", nil)
	return true, nil
}

func checkDependencies() error {
	var missing []string
	if _, err := exec.LookPath("go"); err != nil {
		missing = append(missing, "Go")
	}
	if _, err := exec.LookPath("git"); err != nil {
		missing = append(missing, "Git")
	}

	if len(missing) == 0 {
		return nil
	}

	goos, _ := getPlatform()
	var msg strings.Builder
	msg.WriteString("Missing dependencies for source build: " + strings.Join(missing, ", ") + "\n")
	msg.WriteString("👉 Please install them to continue:\n\n")

	for _, dep := range missing {
		switch dep {
		case "Go":
			switch goos {
			case "android":
				msg.WriteString("   - Termux: pkg install golang\n")
			case "darwin":
				msg.WriteString("   - macOS: brew install go\n")
			case "linux":
				msg.WriteString("   - Ubuntu/Debian: sudo apt install golang\n")
				msg.WriteString("   - Fedora: sudo dnf install golang\n")
				msg.WriteString("   - Arch: sudo pacman -S go\n")
			case "windows":
				msg.WriteString("   - Windows: winget install GoLang.Go\n")
			}
		case "Git":
			switch goos {
			case "android":
				msg.WriteString("   - Termux: pkg install git\n")
			case "darwin":
				msg.WriteString("   - macOS: brew install git\n")
			case "linux":
				msg.WriteString("   - Ubuntu/Debian: sudo apt install git\n")
				msg.WriteString("   - Fedora: sudo dnf install git\n")
				msg.WriteString("   - Arch: sudo pacman -S git\n")
			case "windows":
				msg.WriteString("   - Windows: winget install Git.Git\n")
			}
		}
	}

	return errors.New(msg.String())
}
