package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/nathfavour/vibeauracle/sys"
	"golang.org/x/mod/semver"
)

type releaseInfo struct {
	TagName         string `json:"tag_name"`
	TargetCommitish string `json:"target_commitish"`
	Prerelease      bool   `json:"prerelease"`
	ActualSHA       string `json:"-"`
	Assets          []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

type metadata struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Date    string `json:"date"`
}

func getResilientClient() *http.Client {
	dialer := &net.Dialer{
		Timeout:   10 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Try IPv4 first if dual-stack DNS is flaky
			conn, err := dialer.DialContext(ctx, "tcp4", addr)
			if err != nil {
				// Fallback to default behavior (IPv6/IPv4 as system prefers)
				return dialer.DialContext(ctx, "tcp", addr)
			}
			return conn, nil
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   30 * time.Second,
	}
}

func fetchWithFallback(url string) ([]byte, error) {
	client := getResilientClient()
	resp, err := client.Get(url)
	if err == nil {
		defer resp.Body.Close()
		if resp.StatusCode == http.StatusOK {
			return io.ReadAll(resp.Body)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("server returned status: %d", resp.StatusCode)
		}
	}

	if _, curlErr := exec.LookPath("curl"); curlErr == nil {
		cmd := exec.Command("curl", "-fsL", url)
		data, cmdErr := cmd.Output()
		if cmdErr == nil {
			return data, nil
		}
	}

	return nil, err
}

func gitLSDiscovery(refType string) (map[string]string, error) {
	if _, err := exec.LookPath("git"); err != nil {
		return nil, err
	}

	cmd := exec.Command("git", "ls-remote", "--"+refType, "https://github.com/"+repo+".git")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	results := make(map[string]string)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			sha := parts[0]
			ref := parts[1]
			refName := filepath.Base(ref)
			results[refName] = sha
		}
	}
	return results, nil
}

func getLatestRelease(channel string) (*releaseInfo, error) {
	discoveredTags, _ := gitLSDiscovery("tags")
	if discoveredTags != nil {
		var bestTag string
		if channel == "" || channel == "stable" {
			if _, ok := discoveredTags["latest"]; ok {
				bestTag = "latest"
			} else {
				for tag := range discoveredTags {
					vTag := tag
					if !strings.HasPrefix(vTag, "v") {
						vTag = "v" + vTag
					}
					if semver.IsValid(vTag) && semver.Prerelease(vTag) == "" {
						if bestTag == "" || semver.Compare(vTag, "v"+strings.TrimPrefix(bestTag, "v")) > 0 {
							bestTag = tag
						}
					}
				}
			}
		} else if channel != "" {
			if _, ok := discoveredTags[channel]; ok {
				bestTag = channel
			}
		}

		if bestTag != "" {
			// Try API first to get rich asset data, but don't fail if it hits rate limit
			data, err := fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/releases/tags/%s", repo, bestTag))
			if err == nil {
				var latest releaseInfo
				if err := json.Unmarshal(data, &latest); err == nil && latest.TagName != "" {
					populateActualSHA(&latest)
					return &latest, nil
				}
			}

			// Fallback: Synthesize from git discovery
			synthesized := &releaseInfo{
				TagName:   bestTag,
				ActualSHA: discoveredTags[bestTag],
			}
			return synthesized, nil
		}
	}

	var data []byte
	var err error

	if channel == "" {
		data, err = fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", repo))
		if err == nil {
			var latest releaseInfo
			if err := json.Unmarshal(data, &latest); err == nil && latest.TagName != "" {
				populateActualSHA(&latest)
				return &latest, nil
			}
		}
	}

	data, err = fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/releases", repo))
	if err != nil {
		return nil, err
	}

	var releases []releaseInfo
	if err := json.Unmarshal(data, &releases); err != nil {
		return nil, err
	}

	if len(releases) == 0 {
		return nil, fmt.Errorf("no releases found")
	}

	var latest *releaseInfo

	if channel != "" {
		for i := range releases {
			if strings.EqualFold(releases[i].TagName, channel) {
				latest = &releases[i]
				break
			}
		}
	}

	if latest == nil {
		for i := range releases {
			tag := releases[i].TagName
			if channel == "" && tag == "latest" {
				latest = &releases[i]
				break
			}

			vTag := tag
			if !strings.HasPrefix(vTag, "v") {
				vTag = "v" + vTag
			}

			if semver.IsValid(vTag) && (channel != "" || semver.Prerelease(vTag) == "") {
				if latest == nil {
					latest = &releases[i]
					continue
				}

				latestVTag := latest.TagName
				if latestVTag == "latest" {
					continue
				}
				if !strings.HasPrefix(latestVTag, "v") {
					latestVTag = "v" + latestVTag
				}

				if semver.IsValid(latestVTag) && semver.Compare(vTag, latestVTag) > 0 {
					latest = &releases[i]
				}
			}
		}
	}

	if latest == nil && len(releases) > 0 {
		latest = &releases[0]
	}

	if latest == nil || latest.TagName == "" {
		return nil, fmt.Errorf("could not resolve a valid release")
	}

	populateActualSHA(latest)
	return latest, nil
}

func populateActualSHA(latest *releaseInfo) {
	if latest.ActualSHA != "" {
		return
	}

	for _, asset := range latest.Assets {
		if asset.Name == "metadata.json" {
			metaData, err := fetchWithFallback(asset.BrowserDownloadURL)
			if err == nil {
				var m metadata
				if err := json.Unmarshal(metaData, &m); err == nil && m.Commit != "" {
					latest.ActualSHA = m.Commit
					return
				}
			}
		}
	}

	discovered, err := gitLSDiscovery("tags")
	if err == nil {
		if sha, ok := discovered[latest.TagName]; ok {
			latest.ActualSHA = sha
			return
		}
	}

	tagData, err := fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/git/ref/tags/%s", repo, latest.TagName))
	if err == nil {
		var tagInfo struct {
			Object struct {
				SHA string `json:"sha"`
			} `json:"object"`
		}
		if err := json.Unmarshal(tagData, &tagInfo); err == nil && tagInfo.Object.SHA != "" {
			latest.ActualSHA = tagInfo.Object.SHA
			return
		}
	}

	sha, _ := getBranchCommitSHA(latest.TagName)
	if sha != "" {
		latest.ActualSHA = sha
	}
}

func isUpdateAvailable(latest *releaseInfo, silent bool) bool {
	if latest == nil || latest.TagName == "" {
		return false
	}

	localCommitUnknown := Commit == "" || Commit == "none"

	vLocal := Version
	if !strings.HasPrefix(vLocal, "v") && semver.IsValid("v"+vLocal) {
		vLocal = "v" + vLocal
	}
	vRemote := latest.TagName
	if !strings.HasPrefix(vRemote, "v") && semver.IsValid("v"+vRemote) {
		vRemote = "v" + vRemote
	}

	if semver.IsValid(vLocal) && semver.IsValid(vRemote) {
		return semver.Compare(vRemote, vLocal) > 0
	}

	if latest.TagName == Version && Version != "" && Version != "dev" {
		if localCommitUnknown {
			return false
		}
		return latest.ActualSHA != "" && latest.ActualSHA != Commit
	}

	if silent && (Version == "dev" || strings.HasPrefix(Version, "dev-")) {
		return false
	}

	if latest.TagName != Version {
		if Version == "dev" || strings.HasPrefix(Version, "dev-") {
			return !silent
		}
		if latest.ActualSHA != "" && latest.ActualSHA != Commit {
			return true
		}
		return true
	}

	if localCommitUnknown {
		return false
	}
	return latest.ActualSHA != "" && latest.ActualSHA != Commit
}

func trackUpdateResult(success bool) {
	cm, err := sys.NewConfigManager()
	if err != nil {
		return
	}
	cfg, err := cm.Load()
	if err != nil {
		return
	}

	if success {
		cfg.Update.FailureCount = 0
		cm.Save(cfg)
		return
	}

	now := time.Now()
	cfg.Update.FailureCount++

	shouldSelfSave := false
	if cfg.Update.FailureCount >= 3 {
		shouldSelfSave = true
	} else if cfg.Update.FailureCount >= 2 && !cfg.Update.LastAttempt.IsZero() && now.Sub(cfg.Update.LastAttempt) < 24*time.Hour {
		shouldSelfSave = true
	}

	cfg.Update.LastAttempt = now
	cm.Save(cfg)

	if shouldSelfSave {
		fmt.Printf("\n⚠️  Detected %d failed update attempts in quick succession.\n", cfg.Update.FailureCount)
		fmt.Println("🚀 Attempting self-healing recovery...")

		exe, _ := os.Executable()
		if exe != "" {
			os.Remove(exe)
		}

		useBeta := cfg.Update.Beta
		buildFromSource := cfg.Update.BuildFromSource || useBeta

		var cmd *exec.Cmd
		var recoveryMethod string

		if buildFromSource {
			branch := "release"
			if useBeta {
				branch = "master"
			}
			recoveryMethod = fmt.Sprintf("Source Build (%s)", branch)
			installCmd := fmt.Sprintf("GOTOOLCHAIN=local go install github.com/nathfavour/vibeauracle/cmd/vibeaura@%s", branch)
			cmd = exec.Command("sh", "-c", installCmd)
		} else {
			recoveryMethod = "Official Release Script"
			installCmd := "curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | sh"
			cmd = exec.Command("sh", "-c", installCmd)
		}

		fmt.Printf("⏳ Running recovery via %s...\n", recoveryMethod)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		if err := cmd.Run(); err == nil {
			fmt.Println("\n✅ Recovery successful! Please restart your terminal.")
			cfg.Update.FailureCount = 0
			cm.Save(cfg)
			os.Exit(0)
		} else {
			fmt.Printf("\n❌ Recovery failed: %v\n", err)
			if buildFromSource {
				fmt.Println("👉 Try running: go install github.com/nathfavour/vibeauracle/cmd/vibeaura@latest")
			} else {
				fmt.Println("👉 Please run the install command manually:")
				fmt.Println("   curl -fsSL https://raw.githubusercontent.com/nathfavour/vibeauracle/release/install.sh | sh")
			}
		}
	}
}

func getBranchCommitSHA(branch string) (string, error) {
	// 1. Prioritize git ls-remote to avoid API rate limits
	discovered, err := gitLSDiscovery("heads")
	if err == nil {
		if sha, ok := discovered[branch]; ok {
			return sha, nil
		}
	}

	// 2. Fallback to API
	data, err := fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, branch))
	if err != nil {
		return "", err
	}

	var commit struct {
		SHA string `json:"sha"`
	}
	if err := json.Unmarshal(data, &commit); err != nil {
		return "", err
	}
	return commit.SHA, nil
}

func getCommitMessage(sha string) string {
	if sha == "" {
		return ""
	}

	// 1. If we are in a git repo, try local git log
	if _, err := os.Stat(".git"); err == nil {
		cmd := exec.Command("git", "log", "-1", "--pretty=%s", sha)
		if out, err := cmd.Output(); err == nil {
			msg := strings.TrimSpace(string(out))
			if msg != "" {
				return truncateMessage(msg)
			}
		}
	}

	// 2. Try API, but silently fail if rate limited
	data, err := fetchWithFallback(fmt.Sprintf("https://api.github.com/repos/%s/commits/%s", repo, sha))
	if err == nil {
		var commitData struct {
			Commit struct {
				Message string `json:"message"`
			} `json:"commit"`
		}
		if err := json.Unmarshal(data, &commitData); err == nil {
			msg := strings.Split(commitData.Commit.Message, "\n")[0]
			return truncateMessage(msg)
		}
	}

	return "Update to " + sha[:7]
}

func truncateMessage(msg string) string {
	const maxLen = 60
	if len(msg) > maxLen {
		return msg[:maxLen-3] + "..."
	}
	return msg
}

func checkUpdateSilent() {
	cm, err := sys.NewConfigManager()
	if err != nil {
		return
	}
	cfg, err := cm.Load()
	if err != nil {
		return
	}

	useBeta := cfg.Update.Beta
	buildFromSource := cfg.Update.BuildFromSource || useBeta
	autoUpdate := cfg.Update.AutoUpdate

	isDev := Version == "dev" || strings.HasPrefix(Version, "dev-")
	if isDev && !buildFromSource && !useBeta {
		return
	}

	var latestSHA string
	var latest *releaseInfo

	if buildFromSource || isDev {
		branch := "release"
		if useBeta || strings.HasSuffix(Version, "-master") {
			branch = "master"
		}
		latestSHA, _ = getBranchCommitSHA(branch)
	} else if useBeta {
		latest, err = getLatestRelease("beta")
		if err == nil && isUpdateAvailable(latest, true) {
			latestSHA = latest.ActualSHA
		}
	} else {
		latest, err = getLatestRelease("")
		if err == nil && isUpdateAvailable(latest, true) {
			latestSHA = latest.ActualSHA
		}
	}

	if latestSHA != "" && latestSHA != Commit {
		for _, failed := range cfg.Update.FailedCommits {
			if failed == latestSHA {
				return
			}
		}

		if autoUpdate {
			// Logic handled elsewhere or keep it minimal here
		}
	}
}

func ensureGoBinInPath(goBin string) bool {
	// 1. First, check if vibeaura is ALREADY accessible in a login shell.
	// This is the most accurate test because it respects the user's actual environment.
	if isCommandInPath("vibeaura") {
		return false
	}

	pathEnv := os.Getenv("PATH")
	if strings.Contains(pathEnv, goBin) {
		return false
	}

	home, _ := os.UserHomeDir()

	if runtime.GOOS == "windows" {
		fmt.Printf("📝 Adding %s to your Windows User PATH...\n", goBin)
		cmdStr := fmt.Sprintf(`$oldPath = [System.Environment]::GetEnvironmentVariable("Path", "User"); if ($oldPath -notlike "*%s*") { [System.Environment]::SetEnvironmentVariable("Path", "$oldPath;%s", "User") }`, goBin, goBin)
		err := exec.Command("powershell", "-Command", cmdStr).Run()
		if err != nil {
			fmt.Printf("⚠️  Failed to update Windows PATH automatically: %v\n", err)
			fmt.Printf("👉 Please manually add %s to your PATH.\n", goBin)
			return false
		}
		return true
	}

	cm, _ := sys.NewConfigManager()
	cfg, _ := cm.Load()
	modifiedMap := make(map[string]bool)
	for _, f := range cfg.Shell.ModifiedFiles {
		modifiedMap[f] = true
	}

	// For Unix systems, we prefer absolute paths to avoid tilde expansion issues
	absBinPath := goBin
	if strings.HasPrefix(absBinPath, "~") {
		absBinPath = filepath.Join(home, absBinPath[1:])
	}

	updated := false

	const (
		markerStart = "# >>> vibe auracle initialize >>>"
		markerEnd   = "# <<< vibe auracle initialize <<<"
	)

	// Handle Fish shell specifically
	fishConfig := filepath.Join(home, ".config", "fish", "config.fish")
	if _, err := os.Stat(fishConfig); err == nil {
		content, _ := os.ReadFile(fishConfig)
		if !strings.Contains(string(content), markerStart) && !strings.Contains(string(content), absBinPath) {
			f, err := os.OpenFile(fishConfig, os.O_APPEND|os.O_WRONLY, 0644)
			if err == nil {
				f.WriteString(fmt.Sprintf("\n%s\nfish_add_path %s\n%s\n", markerStart, absBinPath, markerEnd))
				f.Close()
				fmt.Printf("📝 Added %s to Fish configuration (%s)\n", absBinPath, fishConfig)
				updated = true
				modifiedMap[fishConfig] = true
			}
		}
	}

	// Handle Bash/Zsh/POSIX shells
	configs := []string{".zshrc", ".bashrc", ".profile", ".bash_profile"}
	for _, conf := range configs {
		confPath := filepath.Join(home, conf)
		if _, err := os.Stat(confPath); err == nil {
			content, _ := os.ReadFile(confPath)
			if !strings.Contains(string(content), markerStart) && !strings.Contains(string(content), absBinPath) {
				f, err := os.OpenFile(confPath, os.O_APPEND|os.O_WRONLY, 0644)
				if err == nil {
					f.WriteString(fmt.Sprintf("\n%s\nexport PATH=\"$PATH:%s\"\n%s\n", markerStart, absBinPath, markerEnd))
					f.Close()
					fmt.Printf("📝 Added %s to %s\n", absBinPath, conf)
					updated = true
					modifiedMap[confPath] = true
				}
			}
		}
	}

	if updated {
		var newList []string
		for f := range modifiedMap {
			newList = append(newList, f)
		}
		cfg.Shell.ModifiedFiles = newList
		cm.Save(cfg)
		fmt.Println("✅ PATH updated. Please restart your terminal or source your config to apply changes.")
	}
	return updated
}

func isCommandInPath(cmdName string) bool {
	// Check if it's in the current process PATH first
	if _, err := exec.LookPath(cmdName); err == nil {
		return true
	}

	// Try running it in a login shell to see if it's in the user's configured PATH
	shell := os.Getenv("SHELL")
	if shell == "" {
		if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
			shell = "/bin/sh"
		} else {
			return false
		}
	}

	// Use -l to simulate a login shell
	cmd := exec.Command(shell, "-l", "-c", "command -v "+cmdName)
	if err := cmd.Run(); err == nil {
		return true
	}

	return false
}

func sameFile(path1, path2 string) bool {
	fi1, err1 := os.Stat(path1)
	fi2, err2 := os.Stat(path2)
	if err1 != nil || err2 != nil {
		return false
	}
	return os.SameFile(fi1, fi2)
}

func getPlatform() (string, string) {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	if goos == "linux" {
		if _, err := os.Stat("/data/data/com.termux/files/usr/bin/bash"); err == nil || os.Getenv("TERMUX_VERSION") != "" {
			goos = "android"
		}
	}

	return goos, goarch
}
