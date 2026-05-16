//go:build !windows

package proc

import (
	"context"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"syscall"
)

// commandAsOriginalUser builds an *exec.Cmd that, when invoked under sudo,
// runs as the original (non-root) user. This matters for tools like rootless
// podman and nerdctl whose state lives in $HOME/.local — invisible to root.
//
// When not running as root, or when SUDO_UID/SUDO_GID/SUDO_USER aren't set,
// it behaves identically to exec.CommandContext.
func commandAsOriginalUser(ctx context.Context, bin string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, bin, args...)
	if os.Geteuid() != 0 {
		return cmd
	}

	uidStr := os.Getenv("SUDO_UID")
	gidStr := os.Getenv("SUDO_GID")
	sudoUser := os.Getenv("SUDO_USER")
	if uidStr == "" || gidStr == "" || sudoUser == "" {
		return cmd
	}

	uid, err1 := strconv.Atoi(uidStr)
	gid, err2 := strconv.Atoi(gidStr)
	if err1 != nil || err2 != nil || uid <= 0 {
		return cmd
	}

	u, err := user.Lookup(sudoUser)
	if err != nil {
		return cmd
	}

	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(uid),
			Gid: uint32(gid),
		},
	}

	// Rootless container tools resolve sockets and state under HOME and
	// XDG_RUNTIME_DIR. Re-derive both so the child sees the original user's
	// environment, not root's.
	env := os.Environ()
	env = append(env, "HOME="+u.HomeDir)
	env = append(env, "USER="+sudoUser)
	env = append(env, "LOGNAME="+sudoUser)
	env = append(env, "XDG_RUNTIME_DIR=/run/user/"+uidStr)
	cmd.Env = env

	return cmd
}
