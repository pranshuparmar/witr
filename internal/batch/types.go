//go:build linux || darwin

package batch

import "time"

// ProcessSummary is a lightweight view for table display
type ProcessSummary struct {
	PID       int
	Command   string  // Short command name
	Cmdline   string  // Full command line
	User      string
	CPU       float64 // Percentage
	MemoryMB  int     // RSS in MB
	StartedAt time.Time
	Age       string // "2h 15m", "5d", etc.
	Source    string // Simplified: "npm", "launchd", "shell", "vscode"
	Script    string // npm script name OR entry file (e.g., "dev", "server.js")
	WorkDir   string // Working directory (shortened with ~)
	GitRepo   string // Git repo name (just folder name)
	Health    string // "healthy", "high-cpu", etc.
	Error     error  // If analysis failed
}

// BatchResult holds the complete batch operation result
type BatchResult struct {
	Pattern string
	Matches []ProcessSummary
	Total   int
	Elapsed time.Duration
	Errors  int
}
