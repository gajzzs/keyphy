// Author @gajzzs
package main

import (
	"fmt"
	"os"
	"github.com/spf13/cobra"
	"keyphy/internal/app"
	"keyphy/internal/config"
)

var rootCmd = &cobra.Command{
	Use:   "keyphy",
	Short: "System access control using external device authentication",
	Long:  "Keyphy blocks apps, websites, and file access until authenticated with external USB device",
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(
		app.NewAddCommand(),
		app.NewUnblockCommand(),
		app.NewResetCommand(),
		app.NewLockCommand(),
		app.NewUnlockCommand(),
		app.NewListCommand(),
		app.NewDeviceCommand(),
		app.NewServiceCommand(),
	)
}

func main() {
	config.InitConfig()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}