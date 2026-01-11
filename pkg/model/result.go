package model

type Result struct {
	Target         Target
	ResolvedTarget string
	Process        Process
	RestartCount   int
	Ancestry       []Process
	ChildProcesses []Process `json:",omitempty"`
	Source         Source
	Warnings       []string

	// SocketInfo holds socket state details (for port queries)
	SocketInfo *SocketInfo

	// ResourceContext holds resource usage context (macOS)
	ResourceContext *ResourceContext

	// FileContext holds file descriptor and lock info
	FileContext *FileContext
}
