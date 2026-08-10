package target

import "errors"

// Sentinel errors that resolvers return (directly or wrapped with %w) so callers
// branch on errors.Is rather than matching error-message text — control flow that
// depends on exact prose silently breaks when a message is reworded or differs
// across platforms.
var (
	// ErrPortNotBound means no process owns a listening TCP or bound UDP
	// socket on the requested local port.
	ErrPortNotBound = errors.New("no process bound to port")

	// ErrSocketOwnerUnknown means a socket is bound to the port but no
	// host-visible process owns it (systemd socket activation, a container
	// runtime, etc.). It triggers the container/socket fallback.
	ErrSocketOwnerUnknown = errors.New("socket found but owning process not detected")

	// ErrUnsupported means the operation is unavailable on this platform
	// (e.g. resolving by file on Windows).
	ErrUnsupported = errors.New("not supported on this platform")
)
