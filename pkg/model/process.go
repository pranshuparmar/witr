package model

import "time"

type Process struct {
	PID           int
	PPID          int
	Command       string
	Cmdline       string
	Exe           string
	StartedAt     time.Time
	User          string
	CPUPercent    float64
	MemoryRSS     uint64 // In bytes
	MemoryPercent float64

	WorkingDir string
	GitRepo    string
	GitBranch  string
	Container  string
	Service    string

	// Container runtime identity (from Linux cgroup detection) and the runtime's
	// healthcheck verdict for the target: "", "present", or "absent".
	ContainerID          string `json:",omitempty"`
	ContainerRuntime     string `json:",omitempty"`
	ContainerHealthcheck string `json:",omitempty"`

	// Network context — every socket the process owns (LISTEN, ESTABLISHED,
	// CLOSE_WAIT, etc.). Each entry carries protocol and state.
	Sockets []Socket

	// Health status ("healthy", "zombie", "stopped", "high-cpu", "high-mem")
	Health string

	// Forked status ("forked", "not-forked", "unknown")
	Forked string

	// Session is the process's session ID, or 0 when unknown.
	Session int `json:"-"`

	// ParentExited is set when the process that started this one has exited.
	// Either it was adopted by the process above it in the chain, or, at the
	// top of a chain, its parent PID now names nothing or a newer process.
	ParentExited bool `json:",omitempty"`

	// Environment variables (key=value). Kept out of the full --json report,
	// which would otherwise print the environment of every process in the
	// chain; --env --json prints the target's own through ToEnvJSON.
	Env []string `json:"-"`

	// True if the executable was deleted after the process started
	ExeDeleted bool

	// Linux capabilities (e.g., CAP_NET_BIND_SERVICE, CAP_SYS_ADMIN)
	Capabilities []string `json:",omitempty"`

	// The Linux security module confining the process (AppArmor or SELinux)
	// and its label: a profile such as "/usr/sbin/cupsd (enforce)", or a
	// context such as "system_u:system_r:httpd_t:s0".
	SecurityModule string `json:",omitempty"`
	SecurityLabel  string `json:",omitempty"`

	// Windows integrity level: Untrusted, Low, Medium, High (elevated),
	// System or Protected.
	IntegrityLevel string `json:",omitempty"`

	// Extended information for verbose output
	Memory      MemoryInfo `json:",omitempty"`
	IO          IOStats    `json:",omitempty"`
	FileDescs   []string   `json:",omitempty"`
	FDCount     int        `json:",omitempty"`
	FDLimit     uint64     `json:",omitempty"`
	Children    []int      `json:",omitempty"`
	ThreadCount int        `json:",omitempty"`
}

// MemoryInfo contains detailed memory information
type MemoryInfo struct {
	VMS    uint64  // Virtual memory size in bytes
	RSS    uint64  // Resident set size in bytes
	VMSMB  float64 // Virtual memory in MB
	RSSMB  float64 // Resident memory in MB
	Shared uint64  // Shared memory size in bytes
	Text   uint64  // Code size in bytes
	Lib    uint64  // Library size in bytes
	Data   uint64  // Data + stack size in bytes
	Dirty  uint64  // Dirty pages size in bytes
}

// IOStats contains I/O statistics
type IOStats struct {
	ReadBytes  uint64 // Bytes read from storage
	WriteBytes uint64 // Bytes written to storage
	ReadOps    uint64 // Number of read operations
	WriteOps   uint64 // Number of write operations
}
