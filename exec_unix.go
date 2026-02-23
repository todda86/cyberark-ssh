//go:build !windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
)

// execInto replaces the current process with the given command (Unix).
// This gives the child full control of the terminal (important for SSH).
func execInto(binName string, args []string) {
	binPath, err := exec.LookPath(binName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %s not found in PATH: %v\n", binName, err)
		os.Exit(1)
	}

	if err := syscall.Exec(binPath, args, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "error: exec %s: %v\n", binName, err)
		os.Exit(1)
	}
}
