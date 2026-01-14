//go:build !linux && !darwin && !freebsd && !windows

package main

import (
	"fmt"
	"os"
)

func main() {
	fmt.Fprintln(
		os.Stderr,
		"witr is only supported on Linux, macOS, Windows, and FreeBSD.\n\nIf you are seeing this message, you are attempting to build or run witr on an unsupported platform.\n\nPlease use Linux, macOS, Windows, or FreeBSD to build and run witr.",
	)
	os.Exit(1)
}
