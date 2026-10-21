// Package askpass lets sshsave hand a decrypted secret to ssh without ever
// printing it. sshsave sets itself as SSH_ASKPASS; when ssh needs a password
// or key passphrase it runs sshsave again in askpass mode, which writes the
// secret to ssh's pipe.
package askpass

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	envMode   = "SSHSAVE_ASKPASS"
	envSecret = "SSHSAVE_ASKPASS_SECRET"
)

// IsInvocation reports whether this process was started by ssh as askpass
func IsInvocation() bool {
	return os.Getenv(envMode) == "1"
}

// Env returns the environment for running ssh so that it asks sshsave for
// the secret instead of prompting on the terminal
func Env(secret string) ([]string, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("failed to find sshsave executable: %w", err)
	}

	env := os.Environ()
	env = append(env,
		"SSH_ASKPASS="+exe,
		"SSH_ASKPASS_REQUIRE=force",
		envMode+"=1",
		envSecret+"="+secret,
	)
	return env, nil
}

// isConfirmation reports whether ssh is asking a yes/no question (e.g. an
// unknown host key) rather than for a password
func isConfirmation(prompt string) bool {
	return strings.Contains(prompt, "(yes/no")
}

// Run answers one ssh askpass prompt and returns the process exit code
func Run(args []string) int {
	prompt := strings.Join(args, " ")

	if isConfirmation(prompt) {
		return askTerminal(prompt)
	}

	secret, ok := os.LookupEnv(envSecret)
	if !ok {
		fmt.Fprintln(os.Stderr, "sshsave askpass: no secret available")
		return 1
	}
	fmt.Println(secret)
	return 0
}

// askTerminal shows a confirmation prompt to the user and passes the answer
// back to ssh. The secret must never be used to answer these.
func askTerminal(prompt string) int {
	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintf(os.Stderr, "sshsave askpass: cannot open terminal: %v\n", err)
		return 1
	}
	defer tty.Close()

	fmt.Fprint(tty, prompt)
	answer, err := bufio.NewReader(tty).ReadString('\n')
	if err != nil {
		return 1
	}
	fmt.Println(strings.TrimSpace(answer))
	return 0
}
