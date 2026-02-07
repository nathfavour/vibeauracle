package main

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"github.com/nathfavour/vibeauracle/internal/audit"
	"github.com/nathfavour/vibeauracle/sys"
)

func performBinaryUpdate(latest *releaseInfo) error {
	cm, err := sys.NewConfigManager()
	if err != nil {
		return err
	}
	cfg, err := cm.Load()
	if err != nil {
		return err
	}
	verbose := cfg.Update.Verbose
	dataDir := cfg.DataDir

	// Determine target asset name
	goos, goarch := getPlatform()
	targetAsset := fmt.Sprintf("vibeaura-%s-%s", goos, goarch)
	if goos == "windows" {
		targetAsset += ".exe"
	}

	var downloadURL string
	for _, asset := range latest.Assets {
		if asset.Name == targetAsset {
			downloadURL = asset.BrowserDownloadURL
			break
		}
	}

	// Fallback for synthesized releaseInfo (from git ls-remote) where assets are not populated
	if downloadURL == "" && latest.TagName != "" {
		downloadURL = fmt.Sprintf("https://github.com/%s/releases/download/%s/%s", repo, latest.TagName, targetAsset)
	}

	if downloadURL == "" {
		audit.LogFailure(dataDir, audit.EventUpdate, "download_binary", latest.TagName, latest.ActualSHA, "no binary for platform", map[string]interface{}{"os": goos, "arch": goarch})
		return fmt.Errorf("no binary for %s/%s", goos, goarch)
	}

	if verbose {
		fmt.Printf("Downloading %s...\n", targetAsset)
	}

	data, err := fetchWithFallback(downloadURL)
	if err != nil {
		audit.LogFailure(dataDir, audit.EventUpdate, "download_binary", latest.TagName, latest.ActualSHA, err.Error(), nil)
		return err
	}

	// Fetch and verify checksum - STRICT ENFORCEMENT
	checksumURL := fmt.Sprintf("https://github.com/%s/releases/download/%s/checksums.txt", repo, latest.TagName)
	if verbose {
		fmt.Println("Verifying integrity (strict)...")
	}
	checksumData, err := fetchWithFallback(checksumURL)
	if err != nil {
		audit.LogFailure(dataDir, audit.EventUpdate, "verify_checksum", latest.TagName, latest.ActualSHA, "checksums.txt missing: "+err.Error(), nil)
		return fmt.Errorf("STRICT POLICY: checksums.txt not found. Update aborted for security: %w", err)
	}

	if err := verifyChecksum(data, targetAsset, string(checksumData)); err != nil {
		audit.LogFailure(dataDir, audit.EventUpdate, "verify_checksum", latest.TagName, latest.ActualSHA, "checksum mismatch: "+err.Error(), nil)
		return fmt.Errorf("integrity check failed: %w", err)
	}

	tmpFile, err := os.CreateTemp("", "vibeaura-update-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	tmpFile.Close()

	exePath, _ := os.Executable()
	err = installBinary(tmpFile.Name(), exePath)
	if err != nil {
		audit.LogFailure(dataDir, audit.EventUpdate, "install_binary", latest.TagName, latest.ActualSHA, err.Error(), nil)
		return err
	}

	audit.LogSuccess(dataDir, audit.EventUpdate, "binary_update", latest.TagName, latest.ActualSHA, "successfully updated binary", nil)
	return nil
}

func verifyChecksum(data []byte, filename string, checksums string) error {
	sum := fmt.Sprintf("%x", sha256.Sum256(data))
	lines := strings.Split(checksums, "\n")
	for _, line := range lines {
		if strings.Contains(line, filename) {
			expected := strings.Fields(line)[0]
			if expected != sum {
				return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, sum)
			}
			return nil
		}
	}
	return fmt.Errorf("file %s not found in checksums.txt", filename)
}

func installBinary(srcPath, dstPath string) error {
	cm, err := sys.NewConfigManager()
	if err != nil {
		return err
	}
	cfg, err := cm.Load()
	if err != nil {
		return err
	}
	verbose := cfg.Update.Verbose

	if verbose {
		fmt.Printf("Installing binary to %s...\n", dstPath)
	}

	// Ensure the destination directory exists
	dstDir := filepath.Dir(dstPath)
	if _, err := os.Stat(dstDir); os.IsNotExist(err) {
		if err := os.MkdirAll(dstDir, 0755); err != nil {
			// If we can't create the directory, we might need sudo later
		}
	}

	// Ensure the new binary is executable
	os.Chmod(srcPath, 0755)

	// Determine if we need sudo based on path and permissions
	needsSudo := false
	home, _ := os.UserHomeDir()

	if (runtime.GOOS == "linux" || runtime.GOOS == "darwin") && os.Geteuid() != 0 {
		if !strings.HasPrefix(dstPath, home) {
			testPath := filepath.Join(dstDir, ".vibe-perm-test")
			if f, err := os.OpenFile(testPath, os.O_CREATE|os.O_WRONLY, 0644); err != nil {
				needsSudo = true
			} else {
				f.Close()
				os.Remove(testPath)
				if _, err := os.Stat(dstPath); err == nil {
					if f, err := os.OpenFile(dstPath, os.O_WRONLY, 0); err != nil {
						needsSudo = true
					} else {
						f.Close()
					}
				}
			}
		}
	}

	if needsSudo {
		if verbose {
			fmt.Printf("Permission denied or busy. Trying with sudo to install to %s...\n", dstPath)
		}

		exec.Command("sudo", "rm", "-f", dstPath).Run()

		sudoCp := exec.Command("sudo", "cp", srcPath, dstPath)
		if verbose {
			sudoCp.Stdout = os.Stdout
			sudoCp.Stderr = os.Stderr
		}
		sudoCp.Stdin = os.Stdin
		if err := sudoCp.Run(); err != nil {
			if verbose {
				fmt.Println("FAILED")
			}
			return fmt.Errorf("replacing binary with sudo: %w", err)
		}

		exec.Command("sudo", "chmod", "+x", dstPath).Run()
		return nil
	}

	if runtime.GOOS == "windows" {
		if _, err := os.Stat(dstPath); err == nil {
			oldPath := dstPath + ".old"
			os.Remove(oldPath)
			if err := os.Rename(dstPath, oldPath); err != nil {
				return fmt.Errorf("could not move existing binary on Windows: %w", err)
			}
		}
	} else {
		os.Remove(dstPath)
		if _, err := os.Stat(dstPath); err == nil {
			oldPath := dstPath + ".old"
			os.Remove(oldPath)
			if err := os.Rename(dstPath, oldPath); err != nil {
			} else {
				defer os.Remove(oldPath)
			}
		}
	}

	src, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.OpenFile(dstPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0755)
	if err != nil {
		return fmt.Errorf("opening destination binary: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copying binary: %w", err)
	}

	return nil
}

func restartSelf() {
	restartWithArgs(os.Args)
}

func restartWithArgs(args []string) {
	if runtime.GOOS == "windows" {
		fmt.Println("\n✅ Operation complete. Please restart vibeaura.")
		os.Exit(0)
	}

	exe, err := os.Executable()
	if err == nil {
		targetPath := filepath.Join(getUniversalBin(), "vibeaura")
		if runtime.GOOS == "windows" {
			targetPath += ".exe"
		}
		if _, statErr := os.Stat(targetPath); statErr == nil {
			exe = targetPath
		}
	} else {
		fmt.Printf("Error getting executable path for restart: %v\n", err)
		os.Exit(1)
	}

	err = syscall.Exec(exe, args, os.Environ())
	if err != nil {
		fmt.Printf("Error handing off to new binary: %v\n", err)
		os.Exit(1)
	}
}

func getUniversalBin() string {
	home, _ := os.UserHomeDir()
	localBin := filepath.Join(home, ".local", "bin")
	return localBin
}

func ensureInstalled(silent bool) {
	exe, err := os.Executable()
	if err != nil {
		return
	}

	realExe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		realExe = exe
	}

	targetDir := getUniversalBin()
	targetPath := filepath.Join(targetDir, "vibeaura")
	home, _ := os.UserHomeDir()

	if runtime.GOOS == "windows" {
		targetPath += ".exe"
		os.Remove(targetPath + ".old")
	}

	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0755)
	}

	migrated := false
	if realExe != targetPath {
		if err := installBinary(realExe, targetPath); err == nil {
			migrated = true
		}
	}

	locations := getAllBinaryLocations()
	removedAny := false
	for _, loc := range locations {
		if loc != targetPath && loc != realExe && !sameFile(loc, targetPath) && !sameFile(loc, realExe) {
			if strings.HasPrefix(loc, home) {
				if err := os.Remove(loc); err == nil {
					removedAny = true
				}
			}
		}
	}

	updatedPath := ensureGoBinInPath(targetDir)
	if migrated || removedAny || updatedPath {
		audit.LogSuccess(home+"/.vibeauracle", audit.EventInstall, "system_install", Version, Commit, "successfully installed/migrated vibeaura", map[string]interface{}{"migrated": migrated, "removed_others": removedAny, "updated_path": updatedPath})
	}

	if !silent && (migrated || removedAny || updatedPath) {
		if runtime.GOOS == "windows" {
			fmt.Println("\n👉 Since you are on Windows, please close this window and run 'vibeaura' from a new terminal.")
			fmt.Println("Press Enter to exit...")
			var dummy string
			fmt.Scanln(&dummy)
			os.Exit(0)
		}
		restartWithArgs(os.Args)
	}
}

func getAllBinaryLocations() []string {
	var locations []string
	cmd := exec.Command("which", "-a", "vibeaura")
	out, _ := cmd.Output()
	for _, line := range strings.Split(string(out), "\n") {
		path := strings.TrimSpace(line)
		if path != "" {
			if abs, err := filepath.Abs(path); err == nil {
				locations = append(locations, abs)
			}
		}
	}

	unique := make(map[string]bool)
	var final []string
	for _, loc := range locations {
		if !unique[loc] {
			unique[loc] = true
			if _, err := os.Stat(loc); err == nil {
				final = append(final, loc)
			}
		}
	}
	return final
}
