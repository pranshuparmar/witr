//go:build windows

package proc

import (
	"context"
	"os/exec"
)

// commandAsOriginalUser is a no-op on Windows.
func commandAsOriginalUser(ctx context.Context, bin string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, bin, args...)
}
