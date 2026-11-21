package model

import "time"

// Snapshot represents a point-in-time system capture containing running
// processes, open ports, containers, and locked files.
type Snapshot struct {
	Version     int               `json:"version"`
	Index       int               `json:"index"`
	Timestamp   time.Time         `json:"timestamp"`
	Hostname    string            `json:"hostname,omitempty"`
	Processes   []Process         `json:"processes"`
	Ports       []OpenPort        `json:"ports,omitempty"`
	Containers  []*ContainerMatch `json:"containers,omitempty"`
	LockedFiles []*LockedFile     `json:"locked_files,omitempty"`
}
