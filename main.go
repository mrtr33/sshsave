package main

import (
	"os"

	"github.com/mrtr33/sshsave/cmd"
	"github.com/mrtr33/sshsave/internal/askpass"
)

func main() {
	// ssh runs sshsave as SSH_ASKPASS when a saved secret is used
	if askpass.IsInvocation() {
		os.Exit(askpass.Run(os.Args[1:]))
	}

	cmd.Execute()
}
