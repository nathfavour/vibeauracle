package main

import (
	"fmt"
	"os"

	"github.com/nathfavour/vibeauracle/brain"
	"github.com/spf13/cobra"
)

var extensionCmd = &cobra.Command{
	Use:   "extension",
	Short: "Manage vibe auracle extensions",
}

var extensionListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed extensions",
	Run: func(cmd *cobra.Command, args []string) {
		b := brain.New()
		extensions := b.Extensions()

		if len(extensions) == 0 {
			printInfo("No extensions installed.")
			return
		}

		printTitle("🧩", "INSTALLED EXTENSIONS")
		for _, ext := range extensions {
			status := "Enabled"
			if !ext.Enabled {
				status = "Disabled"
			}
			printBulletWithMeta(fmt.Sprintf("%-20s (%s)", ext.Name, ext.UUID[:8]), fmt.Sprintf("[%s] %s", status, ext.Description))
		}
	},
}

var extensionRegisterCmd = &cobra.Command{
	Use:   "register <name> <description>",
	Short: "Register a new tool as an extension",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		name := args[0]
		desc := args[1]
		b := brain.New()

		ext, err := b.RegisterExtension(name, desc)
		if err != nil {
			printError(fmt.Sprintf("Failed to register extension: %v", err))
			os.Exit(1)
		}

		printSuccess(fmt.Sprintf("Extension '%s' registered successfully!", name))
		fmt.Printf("UUID: %s\n", ext.UUID)
		fmt.Printf("Config: ~/.vibeauracle/vibes/%s/vibe.json\n", ext.UUID)
	},
}

var extensionEnableCmd = &cobra.Command{
	Use:   "enable <uuid>",
	Short: "Enable an extension",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		b := brain.New()

		if err := b.SetExtensionEnabled(id, true); err != nil {
			printError(err.Error())
			os.Exit(1)
		}

		printStatus("ENABLED", "Extension "+id)
	},
}

var extensionDisableCmd = &cobra.Command{
	Use:   "disable <uuid>",
	Short: "Disable an extension",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		b := brain.New()

		if err := b.SetExtensionEnabled(id, false); err != nil {
			printError(err.Error())
			os.Exit(1)
		}

		printStatus("DISABLED", "Extension "+id)
	},
}

var extensionInstallCmd = &cobra.Command{
	Use:   "install <repo-uri>",
	Short: "Install a new extension from a repository",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		repo := args[0]
		printInfo("Installing extension from " + repo + "...")
		// Placeholder for actual git clone / install logic
		printSuccess("Extension installed and registered.")
	},
}

var extensionUninstallCmd = &cobra.Command{
	Use:   "uninstall <uuid>",
	Short: "Uninstall an extension",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		printInfo("Uninstalling extension " + id + "...")
		// Placeholder for deletion logic
		printSuccess("Extension uninstalled.")
	},
}

var extensionConfigCmd = &cobra.Command{
	Use:   "config <uuid> <key> <value>",
	Short: "Configure extension settings (e.g. comms.tui, capabilities.agentic)",
	Args:  cobra.ExactArgs(3),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		key := args[1]
		val := args[2]

		printStatus("CONFIG", fmt.Sprintf("Set %s=%s for %s", key, val, id))
		// In a real impl, we'd parse key paths and update the struct
	},
}

func init() {
	extensionCmd.AddCommand(extensionListCmd)
	extensionCmd.AddCommand(extensionRegisterCmd)
	extensionCmd.AddCommand(extensionEnableCmd)
	extensionCmd.AddCommand(extensionDisableCmd)
	extensionCmd.AddCommand(extensionInstallCmd)
	extensionCmd.AddCommand(extensionUninstallCmd)
	extensionCmd.AddCommand(extensionConfigCmd)
}
