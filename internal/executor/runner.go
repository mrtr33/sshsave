package executor

import (
	"fmt"
	"os"
	"os/exec"
)

// Run executes the SSH command and passes through stdio for interactivity.
// env replaces the process environment when non-nil.
func Run(command string, env []string) error {
	cmd := exec.Command("bash", "-c", command)
	cmd.Env = env

	// Pass through stdio for SSH interactivity
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit with the same code as the SSH process
			os.Exit(exitErr.ExitCode())
		}
		return fmt.Errorf("failed to execute command: %w", err)
	}

	return nil
}
