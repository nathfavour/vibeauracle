package main

import (
	"fmt"
	"strings"

	"github.com/nathfavour/vibeauracle/sys"
	"github.com/spf13/cobra"
)

const repo = "nathfavour/vibeauracle"

var (
	betaFlag       bool
	listAssetsFlag bool
	verboseFlag    bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update vibeaura to the latest version",
	RunE: func(cmd *cobra.Command, args []string) error {
		var updateErr error
		defer func() {
			if updateErr != nil {
				trackUpdateResult(false)
			}
		}()

		updateErr = func() error {
			cm, err := sys.NewConfigManager()
			if err != nil {
				return fmt.Errorf("initializing config: %w", err)
			}
			cfg, err := cm.Load()
			if err != nil {
				return fmt.Errorf("loading config: %w", err)
			}

			// If auto-update was disabled (likely due to a rollback), re-enable it
			// now that the user is explicitly running a manual update.
			if !cfg.Update.AutoUpdate {
				cfg.Update.AutoUpdate = true
				if err := cm.Save(cfg); err != nil {
					return fmt.Errorf("re-enabling auto-update: %w", err)
				}
				fmt.Println("🔄  Manual update detected. Automatic updates have been re-enabled.")
			}

			useBeta := betaFlag || cfg.Update.Beta
			buildFromSource := cfg.Update.BuildFromSource || useBeta
			verbose := verboseFlag || cfg.Update.Verbose

			if verboseFlag {
				cfg.Update.Verbose = true
			}

			if listAssetsFlag {
				if buildFromSource {
					return fmt.Errorf("--list-assets is only supported for the pre-built update pipeline (source updates do not use assets)")
				}

				fmt.Println("Fetching latest release assets...")
				reqChannel := ""
				if useBeta {
					reqChannel = "beta"
				}
				latest, err := getLatestRelease(reqChannel)
				if err != nil {
					return fmt.Errorf("checking for updates: %w", err)
				}

				fmt.Printf("\n📦 Assets for release %s:\n", latest.TagName)
				for _, asset := range latest.Assets {
					fmt.Printf("  - %s\n", asset.Name)
				}
				fmt.Println()
				return nil
			}

			curCommit := Commit
			if len(curCommit) > 7 {
				curCommit = curCommit[:7]
			}

			if verbose {
				fmt.Printf("Current version: %s (commit: %s)\n", Version, curCommit)
			}

			if buildFromSource {
				branch := "release"
				if useBeta {
					branch = "master"
				}

				if !verbose {
					fmt.Printf("🔄  Updating to %s... ", branch)
				} else {
					if useBeta {
						fmt.Println("🚀 Entering Beta Mode: Building bleeding-edge from master...")
					} else {
						fmt.Println("🛠️ Building from source (release branch)...")
					}
				}

				updated, err := updateFromSource(branch, cm)
				if err != nil {
					if !verbose {
						fmt.Println("FAILED")
					}
					return err
				}

				if !updated {
					if !verbose {
						fmt.Println("ALREADY UP TO DATE")
					} else {
						fmt.Println("vibeaura is already up to date on this branch.")
					}
					return nil
				}

				if !verbose {
					remoteSHA, _ := getBranchCommitSHA(branch)
					displaySHA := remoteSHA
					if len(displaySHA) > 7 {
						displaySHA = displaySHA[:7]
					}
					printSuccess("Upgraded to " + displaySHA + ": " + getCommitMessage(remoteSHA))
				} else {
					fmt.Printf("Successfully updated to bleeding-edge %s from source!\n", branch)
				}

				trackUpdateResult(true)
				restartSelf()
				return nil
			}

			fmt.Println("Checking for updates...")
			reqChannel := ""
			if useBeta {
				reqChannel = "beta"
			}
			latest, err := getLatestRelease(reqChannel)
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			isDev := strings.HasPrefix(Version, "dev")
			if !isUpdateAvailable(latest, false) && !isDev {
				fmt.Println("vibeaura is already up to date!")
				return nil
			}

			if isDev {
				fmt.Printf("Dev build detected. Force-updating to latest stable binary (%s)...\n", latest.TagName)
			}

			remoteVer := latest.ActualSHA
			if remoteVer == "" {
				remoteVer = latest.TargetCommitish
			}

			// Check if this commit has previously failed
			for _, failed := range cfg.Update.FailedCommits {
				if failed == remoteVer {
					fmt.Printf("\n⚠️ The latest version (%s) has previously failed to install/build and is likely unstable.\n", remoteVer[:7])
					fmt.Println("👉 Use '--beta' or '--source' flags to force a retry if you've fixed the issue.")
					return nil
				}
			}

			displaySHA := remoteVer
			if len(displaySHA) > 7 {
				displaySHA = displaySHA[:7]
			}

			fmt.Printf("New version available: %s (commit: %s)\n", latest.TagName, displaySHA)

			if err := performBinaryUpdate(latest); err != nil {
				return err
			}

			if verbose {
				fmt.Printf("Successfully updated to %s!\n", latest.TagName)
			} else {
				printSuccess("Upgraded to " + displaySHA + ": " + getCommitMessage(remoteVer))
			}

			trackUpdateResult(true)
			restartSelf()
			return nil
		}()

		return updateErr
	},
}

func init() {
	updateCmd.Flags().BoolVar(&betaFlag, "beta", false, "Install bleeding-edge version from source (master branch)")
	updateCmd.Flags().BoolVar(&listAssetsFlag, "list-assets", false, "List all assets available in the latest release")
	updateCmd.Flags().BoolVar(&verboseFlag, "verbose", false, "Show detailed output during update")
	rootCmd.AddCommand(updateCmd)
}