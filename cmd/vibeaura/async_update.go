package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/nathfavour/vibeauracle/internal/audit"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathfavour/vibeauracle/sys"
)
// execGitCommand runs a git command and returns stdout.
func execGitCommand(args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// --- Async Hot-Swap Logic ---

type UpdateAvailableMsg struct {
	Latest *releaseInfo
}

type UpdateReadyMsg struct {
	Target string // SHA
}

type AsyncUpdateManager struct {
	cm *sys.ConfigManager
}

func NewAsyncUpdateManager() *AsyncUpdateManager {
	cm, err := sys.NewConfigManager()
	if err != nil {
		// We return a manager with a nil CM, but methods check for it
		return &AsyncUpdateManager{cm: nil}
	}
	return &AsyncUpdateManager{cm: cm}
}

// CheckUpdateCmd returns a command that checks for updates in the background.
func (chk *AsyncUpdateManager) CheckUpdateCmd(manual bool) tea.Cmd {
	return func() tea.Msg {
		cm, err := sys.NewConfigManager() // Reload config
		if err != nil {
			if manual {
				return UpdateNoUpdateMsg{}
			}
			return nil
		}
		chk.cm = cm
		cfg, err := chk.cm.Load()
		if err != nil {
			if manual {
				return UpdateNoUpdateMsg{}
			}
			return nil
		}

		// Manual updates always proceed; AutoUpdate setting is only for background.
		if manual || cfg.Update.AutoUpdate {
			updateAvailable, latest := CheckForUpdate(cfg, manual)
			if updateAvailable && latest != nil {
				return UpdateAvailableMsg{Latest: latest}
			}
		}

		if manual {
			return UpdateNoUpdateMsg{}
		}

		return nil
	}
}

type UpdateNoUpdateMsg struct{}

// DownloadUpdateCmd downloads the update in background
func (chk *AsyncUpdateManager) DownloadUpdateCmd(latest *releaseInfo) tea.Cmd {
	return func() tea.Msg {
		if chk.cm == nil {
			return nil
		}
		cfg, err := chk.cm.Load()
		if err != nil {
			return nil
		}

		err = PerformUpdate(cfg, latest)
		if err != nil {
			trackUpdateResult(false)
			audit.LogFailure(cfg.DataDir, audit.EventUpdate, "async_update", latest.TagName, latest.ActualSHA, err.Error(), nil)
			return nil
		}

		audit.LogSuccess(cfg.DataDir, audit.EventUpdate, "async_update", latest.TagName, latest.ActualSHA, "successfully updated in background", nil)
		trackUpdateResult(true)
		return UpdateReadyMsg{Target: latest.ActualSHA}
	}
}

// PerformHotSwap saves state and execs the new binary
func PerformHotSwap(headers []string, input string) {
	state := map[string]interface{}{
		"messages": headers,
		"input":    input,
	}

	bytes, _ := json.Marshal(state)
	tmpState, _ := os.CreateTemp("", "vibeaura-state-*.json")
	tmpState.Write(bytes)
	tmpState.Close()

	// 2. Restart
	exe, _ := os.Executable()

	// We need to construct args. We can't just use os.Args because we need to strip previous restart flags if any
	// and add the new one.
	var newArgs []string
	// Filter old flag
	// Note: os.Args[0] is the program name
	if len(os.Args) > 0 {
		newArgs = append(newArgs, os.Args[0])
	}

	for i := 1; i < len(os.Args); i++ {
		if os.Args[i] == "--resume-state" {
			i++ // skip value
			continue
		}
		if strings.HasPrefix(os.Args[i], "--resume-state=") {
			continue
		}
		newArgs = append(newArgs, os.Args[i])
	}
	newArgs = append(newArgs, "--resume-state", tmpState.Name())

	// Exec replaces the process
	syscall.Exec(exe, newArgs, os.Environ())
}
