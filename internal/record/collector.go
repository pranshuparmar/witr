package record

import (
	"fmt"
	"os"
	"time"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

// CollectorFunc defines a function signature for capturing system snapshots.
type CollectorFunc func(index int) (*model.Snapshot, error)

// DefaultCollector captures real system state across processes, ports, containers, and locks.
func DefaultCollector(index int) (*model.Snapshot, error) {
	procs, err := procpkg.ListProcesses()
	if err != nil {
		return nil, fmt.Errorf("list processes: %w", err)
	}

	ports, _ := procpkg.ListOpenPorts()
	containers := procpkg.ListAllContainers()
	locks := procpkg.ListLockedFiles()
	hostname, _ := os.Hostname()

	return &model.Snapshot{
		Version:     CurrentSnapshotVersion,
		Index:       index,
		Timestamp:   time.Now().UTC(),
		Hostname:    hostname,
		Processes:   procs,
		Ports:       ports,
		Containers:  containers,
		LockedFiles: locks,
	}, nil
}
