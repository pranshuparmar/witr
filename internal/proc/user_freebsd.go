//go:build freebsd

package proc

import (
	"os/user"
	"strconv"
)

func readUser(pid int) string {
	// On FreeBSD, we get the UID from ps in ReadProcess and resolve it here
	// This function is a fallback that just returns unknown
	return "unknown"
}

func readUserByUID(uid int) string {
	return resolveUID(uid)
}

func resolveUID(uid int) string {
	if uid == 0 {
		return "root"
	}

	// Try to resolve username using os/user package (works on FreeBSD)
	u, err := user.LookupId(strconv.Itoa(uid))
	if err == nil {
		return u.Username
	}

	// Fallback to UID as string
	return strconv.Itoa(uid)
}
