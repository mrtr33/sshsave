package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "sshsave",
	Short: "A lightweight CLI tool for managing SSH connection shortcuts",
	Long: `SSHLauncher (sshsave) is a CLI tool that helps you save, list, and quickly
run SSH connection shortcuts. It supports aliases and shell auto-completion.

Examples:
  sshsave add work ssh user@192.168.10.10
  sshsave run work
  sshsave list
  sshsave rm work`,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags can be added here if needed
}
