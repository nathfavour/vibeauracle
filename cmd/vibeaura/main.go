package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/daemon"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/spf13/cobra"
)

var (
	Version         = "dev"
	Commit          = "none"
	BuildDate       = "unknown"
	resumeStateFile string // For hot-swap restoration
)

func init() {
	// Try to populate Version and Commit from build info if they are defaults
	if info, ok := debug.ReadBuildInfo(); ok {
		if Version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}

		for _, setting := range info.Settings {
			switch setting.Key {
			case "vcs.revision":
				if Commit == "none" {
					Commit = setting.Value
				}
			case "vcs.time":
				if BuildDate == "unknown" {
					BuildDate = setting.Value
				}
			}
		}
	}

	// If we're still in "dev" mode, try to find the current git branch
	if Version == "dev" {
		if _, err := os.Stat(".git"); err == nil {
			branchCmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
			if branchBytes, err := branchCmd.Output(); err == nil {
				Version = "dev-" + strings.TrimSpace(string(branchBytes))
			}
		}
	}
}

var rootCmd = &cobra.Command{
	Use:     "vibeaura",
	Version: Version,
	Short:   "vibe auracle - Distributed, System-Intimate AI Engineering Ecosystem",
		Long: `vibe auracle is a keyboard-centric interface that unifies the terminal, 
	the IDE, and the AI assistant into a single system-aware experience.`,
	}
	
	var (
		authCmd = &cobra.Command{Use: "auth", Short: "Manage AI credentials"}
		authCopilotCmd = &cobra.Command{Use: "github-copilot"}
		authGithubCmd = &cobra.Command{Use: "github-models <token>"}
		authOllamaCmd = &cobra.Command{Use: "ollama <endpoint>"}
		authOpenAICmd = &cobra.Command{Use: "openai <key>"}
		
		modelsCmd = &cobra.Command{Use: "models", Short: "Manage AI models"}
		modelsListCmd = &cobra.Command{Use: "list"}
		modelsUseCmd = &cobra.Command{Use: "use <provider> <model>"}
		
		agentCmd = &cobra.Command{Use: "agent", Short: "Select agent engine"}
		agentVibeCmd = &cobra.Command{Use: "vibe"}
		agentSDKCmd = &cobra.Command{Use: "sdk"}
		
			sysCmd = &cobra.Command{Use: "sys", Short: "System controls"}
			sysStatsCmd = &cobra.Command{Use: "stats"}
			
			restartCmd = &cobra.Command{Use: "restart", Short: "Restart VibeAuracle"}
		)
		
		func main() {
			ensureInstalled()

	// 1. Initialize Core Brain (loads extensions, configs, etc.)
	b := brain.New()

	// 2. Setup CLI Context
	rootCmd.SetOut(NewColorWriter(os.Stdout))
	rootCmd.SetErr(NewColorWriter(os.Stderr))
	rootCmd.PersistentFlags().StringVar(&resumeStateFile, "resume-state", "", "Internal use: resume state from file")
	rootCmd.PersistentFlags().MarkHidden("resume-state")

	// 3. Define Main Interactive Loop
	rootCmd.Run = func(cmd *cobra.Command, args []string) {
		doctor.Start()

		// Start Background Daemon for IPC
	home, _ := os.UserHomeDir()
	socketPath := filepath.Join(home, ".vibeauracle", "vibeaura.sock")
	d := daemon.New(socketPath, b)
	go d.Start(context.Background())

		// Inject Status Reporting into Tooling
		tooling.StatusReporter = func(icon, step, msg string) {
			doctor.Send("tooling", doctor.SignalInit, fmt.Sprintf("%s %s", step, msg), nil)
			select {
			case StatusStream <- StatusEvent{Icon: icon, Step: step, Message: msg}:
			default:
				// Drop if buffer full
			}
		}

		// Run TUI
		m := initialModel(b)
		p := tea.NewProgram(m, tea.WithAltScreen())

		                // Connect brain callbacks to the TUI program
		                b.OnStreamDelta = func(delta string) {
		                        p.Send(streamDeltaMsg{Delta: delta})
		                }
		                b.OnStreamDone = func(full string) {
		                        p.Send(streamDoneMsg{FullContent: full})
		                }
		                b.OnUsage = func(usage model.Usage) {
		                        p.Send(usageMsg(usage))
		                }
				if _, err := p.Run(); err != nil {
			doctor.Send("tui", doctor.SignalError, err.Error(), nil)
			fmt.Printf("Alas, there's been an error: %v", err)
			os.Exit(1)
		}
	}

	// 4. Register Sub-Commands (sharing the pre-initialized brain 'b')
	setupCommands(b)

	// 5. Register Dynamic Commands from Extensions
	registerDynamicCommands(rootCmd, b)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func setupCommands(b *brain.Brain) {
	// Auth
	authCmd.Run = nil // group command
	rootCmd.AddCommand(authCmd)
	
	authCopilotCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := b.SetModel("github-copilot", "gpt-4o"); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printSuccess("GitHub Copilot configured automatically.")
	}
	authCmd.AddCommand(authCopilotCmd)

	authGithubCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := b.StoreSecret("github_models_pat", args[0]); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printSuccess("GitHub Models PAT stored.")
	}
	authCmd.AddCommand(authGithubCmd)

	authOllamaCmd.Run = func(cmd *cobra.Command, args []string) {
		cfg := b.Config()
		cfg.Model.Endpoint = args[0]
		if err := b.UpdateConfig(cfg); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printSuccess("Ollama endpoint updated.")
	}
	authCmd.AddCommand(authOllamaCmd)

	authOpenAICmd.Run = func(cmd *cobra.Command, args []string) {
		if err := b.StoreSecret("openai_api_key", args[0]); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printSuccess("OpenAI API key stored.")
	}
	authCmd.AddCommand(authOpenAICmd)

	// Models
	rootCmd.AddCommand(modelsCmd)
	modelsListCmd.Run = func(cmd *cobra.Command, args []string) {
		printInfo("Discovering models...")
		discoveries, err := b.DiscoverModels(cmd.Context())
		if err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		if len(discoveries) == 0 {
			printWarning("No models found.")
			return
		}
		printTitle("✨", "AVAILABLE MODELS")
		for _, d := range discoveries {
			printBulletWithMeta(fmt.Sprintf("% -30s", brain.ShortenModelName(d.Name)), d.Provider)
		}
	}
	modelsCmd.AddCommand(modelsListCmd)

	modelsUseCmd.Run = func(cmd *cobra.Command, args []string) {
		if err := b.SetModel(args[0], args[1]); err != nil {
			printError(err.Error())
			os.Exit(1)
		}
		printStatus("SWITCHED", args[1])
	}
	modelsCmd.AddCommand(modelsUseCmd)

	// Agent
	rootCmd.AddCommand(agentCmd)
	agentVibeCmd.Run = func(cmd *cobra.Command, args []string) {
		b.SetAgentMode("vibe")
		printStatus("AGENT", "Now using Vibe loop")
	}
	agentCmd.AddCommand(agentVibeCmd)

	agentSDKCmd.Run = func(cmd *cobra.Command, args []string) {
		b.SetAgentMode("sdk")
		printStatus("AGENT", "Now using SDK loop")
	}
	agentCmd.AddCommand(agentSDKCmd)

	// Sys
	rootCmd.AddCommand(sysCmd)
	sysStatsCmd.Run = func(cmd *cobra.Command, args []string) {
		snapshot, _ := b.GetSnapshot()
		printTitle("⚡", "POWER SNAPSHOT")
		printKeyValueHighlight("CPU", fmt.Sprintf("%.1f%%", snapshot.CPUUsage))
		printKeyValueHighlight("MEM", fmt.Sprintf("%.1f%%", snapshot.MemoryUsage))
		printKeyValue("CWD", snapshot.WorkingDir)
	}
	sysCmd.AddCommand(sysStatsCmd)

	// Other
	rootCmd.AddCommand(daemonCmd)
	rootCmd.AddCommand(extensionCmd)
	rootCmd.AddCommand(directCmd)
	rootCmd.AddCommand(restartCmd)
}

func registerDynamicCommands(root *cobra.Command, b *brain.Brain) {
	for _, ext := range b.Extensions() {
		if !ext.Enabled || ext.Manifest == nil || len(ext.Manifest.CLICommands) == 0 {
			continue
		}

		for _, cliCmd := range ext.Manifest.CLICommands {
			cmd := cliCmd // capture
			extRef := ext // capture
			dynamicCmd := &cobra.Command{
				Use:   cmd.Name,
				Short: fmt.Sprintf("[%s] %s", extRef.Name, cmd.Description),
				Run: func(cobraCmd *cobra.Command, args []string) {
					execCmd := exec.Command(extRef.Manifest.Command, cmd.Action)
					execCmd.Stdout = os.Stdout
					execCmd.Stderr = os.Stderr
					execCmd.Stdin = os.Stdin
					if err := execCmd.Run(); err != nil {
						fmt.Printf("Error: %v\n", err)
						os.Exit(1)
					}
				},
			}
			root.AddCommand(dynamicCmd)
		}
	}
}