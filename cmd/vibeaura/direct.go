package main

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/nathfavour/vibeauracle/brain"
	"github.com/nathfavour/vibeauracle/internal/doctor"
	"github.com/nathfavour/vibeauracle/tooling"
	"github.com/spf13/cobra"
)

var (
	directVerbose bool
	directNonInteractive bool
)

var directCmd = &cobra.Command{
	Use:   "direct [prompt]",
	Short: "Direct CLI interaction without TUI (Verbose Debug Mode)",
	Long: `Direct mode bypasses the Bubble Tea TUI to provide a raw, 
stream-to-terminal experience. Highly recommended for debugging 
complex agentic loops and provider issues.`,
	Run: func(cmd *cobra.Command, args []string) {
	doctor.Start()
	b := brain.New()

		// Setup Verbose Status Reporting
		tooling.StatusReporter = func(icon, step, msg string) {
			if directVerbose {
				fmt.Printf("\033[34m[%s] %-12s |\033[0m %s\n", icon, strings.ToUpper(step), msg)
			}
			// Always send to doctor for persistent logs
		doctor.Send("tooling", doctor.SignalInit, fmt.Sprintf("%s %s", step, msg), nil)
		}

		// Connect brain callbacks to stdout
		b.OnStreamDelta = func(delta string) {
			fmt.Print(delta)
		}
		b.OnStreamDone = func(full string) {
			fmt.Println() 
		}

		// One-shot execution if prompt provided
		if len(args) > 0 {
			prompt := strings.Join(args, " ")
			if directVerbose {
				fmt.Printf("\033[1;32mUser:\033[0m %s\n", prompt)
			}
			_, err := b.Process(context.Background(), brain.Request{Content: prompt})
			if err != nil {
				fmt.Printf("\n\033[31mBRAIN ERROR:\033[0m %v\n", err)
				os.Exit(1)
			}
			return
		}

		if directNonInteractive {
			return
		}

		// REPL Mode
		scanner := bufio.NewScanner(os.Stdin)
		fmt.Println("\033[1;35m--- VibeAuracle Direct REPL ---\033[0m")
		fmt.Println("Type 'exit' to quit, 'clear' to clear screen.")
		if directVerbose {
			fmt.Println("Extremely Verbose Mode: ACTIVE")
		}
		fmt.Println()

		for {
			fmt.Print("\033[1;32m> \033[0m")
			if !scanner.Scan() {
				break
			}
			input := strings.TrimSpace(scanner.Text())
			
			if input == "" {
				continue
			}
			if input == "exit" || input == "quit" {
				break
			}
			if input == "clear" {
				fmt.Print("\033[H\033[2J")
				continue
			}

			// Handle Slash Commands in Direct Mode
			if strings.HasPrefix(input, "/") {
				fmt.Println("\033[33mNote: TUI commands like /shot or /show-tree are disabled in direct mode.\033[0m")
				// We could potentially route these to handleSlashCommand, but many are TUI specific.
				// For now, let the brain handle them as text or ignore.
			}

			_, err := b.Process(context.Background(), brain.Request{Content: input})
			if err != nil {
				fmt.Printf("\n\033[31mBRAIN ERROR:\033[0m %v\n", err)
			}
			fmt.Println()
		}
	},
}

func init() {
	directCmd.Flags().BoolVarP(&directVerbose, "verbose", "v", true, "Enable extremely verbose logging (defaults to true in direct mode)")
	directCmd.Flags().BoolVarP(&directNonInteractive, "non-interactive", "n", false, "Exit after one-shot (if prompt provided)")
	rootCmd.AddCommand(directCmd)
}
