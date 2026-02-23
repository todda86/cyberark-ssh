//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
)

// execInto launches the given command and waits for it to finish (Windows).
// Windows does not support syscall.Exec, so we spawn a child process and
// proxy stdin/stdout/stderr instead.
func execInto(binName string, args []string) {
	binPath, err := exec.LookPath(binName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s not found in PATH: %v\n", binName, err)
		os.Exit(1)
	}

	cmd := exec.Command(binPath, args[1:]...) // args[0] is the program name
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			os.Exit(exitErr.ExitCode())
		}
		fmt.Fprintf(os.Stderr, "error: %s: %v\n", binName, err)
		os.Exit(1)
	}
}
