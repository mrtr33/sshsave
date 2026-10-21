package cmd

import (
	"fmt"
	"os"
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is set at build time with -ldflags "-X github.com/mrtr33/sshsave/cmd.version=..."
var version string

var rootCmd = &cobra.Command{
	Use:   "sshsave [name]",
	Short: "A lightweight CLI tool for managing SSH connection shortcuts",
	Long: `sshsave is a CLI tool that helps you save, list, and quickly
run SSH connection shortcuts. It supports aliases and shell auto-completion.

"sshsave <name>" is a shortcut for "sshsave run <name>". Commands take
precedence, so an entry named like a command must be run with "run".

Examples:
  sshsave add work ssh user@192.168.10.10
  sshsave work
  sshsave list
  sshsave rm work`,
	Version:           getVersion(),
	Args:              cobra.MaximumNArgs(1),
	ValidArgsFunction: completeEntryNames,
	// Errors are printed once by Execute; usage is only shown on request
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) == 0 {
			return cmd.Help()
		}
		return runEntry(args[0])
	},
}

// getVersion returns the build-time version, or the module version when
// installed with "go install"
func getVersion() string {
	if version != "" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version
	}
	return "dev"
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
