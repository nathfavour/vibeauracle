package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/nathfavour/vibeauracle/sys"
)

func updateFromSource(branch string, cm *sys.ConfigManager) (bool, error) {
	cfg, _ := cm.Load()
	verbose := cfg.Update.Verbose

	// Check if Go is installed
	if _, err := exec.LookPath("go"); err != nil {
		return false, fmt.Errorf("Go is not installed. Source build requires Go.")
	}
	// Check if git is installed
	if _, err := exec.LookPath("git"); err != nil {
		return false, fmt.Errorf("Git is not installed. Source build requires Git.")
	}

	sourceRoot := cm.GetDataPath(filepath.Join("source", branch))
	if err := os.MkdirAll(filepath.Dir(sourceRoot), 0755); err != nil {
		return false, fmt.Errorf("creating source directory: %w", err)
	}

	if _, err := os.Stat(filepath.Join(sourceRoot, ".git")); os.IsNotExist(err) {
		if verbose {
			fmt.Printf("Cloning %s branch to %s...
", branch, sourceRoot)
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
			fmt.Printf("Fetching updates for %s...
", branch)
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
			fmt.Printf("Updating local source in %s...
", sourceRoot)
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
			fmt.Println("
🛠️  Build failed. Attempting to upgrade Go toolchain automatically...")
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
						return false, err
					}
					return true, nil
				}
			}
		}

		if verbose {
			fmt.Println("
❌ Build failed! This usually happens if your installed Go version is older than the one required by the project.")
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
			}
		}
		return false, fmt.Errorf("building from source: %w", err)
	}

	exePath, _ := os.Executable()
	if err := installBinary(buildOut, exePath); err != nil {
		return false, err
	}

	return true, nil
}
