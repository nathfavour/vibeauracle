package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/nathfavour/vibeauracle/internal/audit"
	"github.com/nathfavour/vibeauracle/sys"
	"github.com/spf13/cobra"
)

var (
	cleanUninstall bool
	revertShell    bool
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove vibeaura from your system",
	Long: `Uninstall the vibeaura binary. 
By default, the application data directory (~/.vibeauracle) is preserved. 
Use the --clean flag to wipe everything.`,
	Run: func(cmd *cobra.Command, args []string) {
		printTitle("🗑️", "UNINSTALL VIBE AURACLE")

		// 1. Get binary path
		exePath, err := os.Executable()
		if err != nil {
			printError("Could not determine binary path: " + err.Error())
			return
		}

		// 2. Get data directory and config
		cm, err := sys.NewConfigManager()
		var dataDir string
		var modifiedFiles []string
		if err == nil {
			cfg, err := cm.Load()
			if err == nil {
				dataDir = cfg.DataDir
				modifiedFiles = cfg.Shell.ModifiedFiles
			}
		} else {
			// Fallback if config manager fails
			if home, err := os.UserHomeDir(); err == nil {
				dataDir = fmt.Sprintf("%s/.vibeauracle", home)
			}
		}

		// 3. Revert shell modifications if requested
		if revertShell && len(modifiedFiles) > 0 {
			printInfo("Reverting shell modifications...")
			revertShellModifications(modifiedFiles)
		}

		audit.LogSuccess(dataDir, audit.EventUninstall, "uninstall", Version, Commit, "user initiated uninstallation", map[string]interface{}{"clean": cleanUninstall, "revert_shell": revertShell})

		// 4. Remove binary
		printInfo("Removing binary: " + exePath)
		if err := os.Remove(exePath); err != nil {
			printError("Failed to remove binary: " + err.Error())
			// We continue to data wiping even if binary removal fails (e.g. permission issues)
		} else {
			printBullet("Binary removed successfully")
		}

		// 5. Clean data if requested
		if cleanUninstall && dataDir != "" {
			if _, err := os.Stat(dataDir); err == nil {
				printInfo("Wiping data directory: " + dataDir)
				if err := os.RemoveAll(dataDir); err != nil {
					printError("Failed to wipe data: " + err.Error())
				} else {
					printBullet("Application data wiped successfully")
				}
			} else {
				printInfo("Data directory not found, skipping wipe.")
			}
		} else if dataDir != "" {
			printInfo("Keeping application data at: " + dataDir)
		}

		printDone()
		printNewline()
		if !revertShell {
			fmt.Println(cliMuted.Render("Note: If you established any shells integrations manually, you may need to remove them from your shell profile."))
		}
	},
}

func revertShellModifications(files []string) {
	const (
		markerStart = "# >>> vibe auracle initialize >>>"
		markerEnd   = "# <<< vibe auracle initialize <<<"
	)

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			printWarning(fmt.Sprintf("Could not read %s: %v", file, err))
			continue
		}

		strContent := string(content)
		if !strings.Contains(strContent, markerStart) {
			continue
		}

		// Simple regex-less removal of marked block
		startIdx := strings.Index(strContent, markerStart)
		endIdx := strings.Index(strContent, markerEnd)

		if startIdx != -1 && endIdx != -1 && endIdx > startIdx {
			// Remove the markers and everything in between
			newContent := strContent[:startIdx] + strContent[endIdx+len(markerEnd):]
			// Trim potential leading/trailing newlines introduced by removal
			newContent = strings.TrimSpace(newContent) + "\n"

			if err := os.WriteFile(file, []byte(newContent), 0644); err != nil {
				printError(fmt.Sprintf("Failed to update %s: %v", file, err))
			} else {
				printBullet(fmt.Sprintf("Reverted modifications in %s", file))
			}
		}
	}
}

func init() {
	uninstallCmd.Flags().BoolVar(&cleanUninstall, "clean", false, "Wipe both binary and the entire data directory")
	uninstallCmd.Flags().BoolVar(&revertShell, "revert-shell", false, "Revert PATH modifications in shell configuration files")
	rootCmd.AddCommand(uninstallCmd)
}
