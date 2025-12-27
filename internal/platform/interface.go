package platform

import (
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ProcessReader reads process information for a given PID.
type ProcessReader interface {
	ReadProcess(pid int) (model.Process, error)
}

// ProcessLister enumerates all PIDs on the system.
type ProcessLister interface {
	ListPIDs() ([]int, error)
}

// PortResolver finds PIDs listening on a given port.
type PortResolver interface {
	ResolvePort(port int) ([]int, error)
}

// NameResolver finds PIDs by process name.
type NameResolver interface {
	ResolveName(name string) ([]int, error)
}

// BootTimeProvider returns the system boot time.
type BootTimeProvider interface {
	BootTime() time.Time
}

// Platform combines all platform-specific operations.
// Implementations are selected at compile time via build tags.
type Platform interface {
	ProcessReader
	ProcessLister
	PortResolver
	NameResolver
	BootTimeProvider
}

// Current is the platform implementation for the current OS.
// Set by init() in platform-specific files (linux.go, darwin.go).
var Current Platform
