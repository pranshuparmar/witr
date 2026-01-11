package model

import "time"

type Process struct {
	PID       int
	PPID      int
	Command   string
	Cmdline   string
	Exe       string
	StartedAt time.Time
	User      string

	WorkingDir string
	GitRepo    string
	GitBranch  string
	Container  string
	Service    string

	// Network context
	ListeningPorts []int
	BindAddresses  []string

	// Health status ("healthy", "zombie", "stopped", "high-cpu", "high-mem")
	Health string

	// Forked status ("forked", "not-forked", "unknown")
	Forked string
	// Environment variables (key=value)
	Env []string

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
