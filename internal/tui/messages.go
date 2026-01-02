//go:build linux || darwin

package tui

import (
	"time"

	"github.com/SanCognition/witr/internal/batch"
	"github.com/SanCognition/witr/pkg/model"
)

// tickMsg signals a refresh tick
type tickMsg time.Time

// processesMsg contains refreshed process data
type processesMsg struct {
	processes []batch.ProcessSummary
	err       error
}

// detailsMsg contains detailed info for a specific process
type detailsMsg struct {
	pid      int
	ancestry []model.Process
	ports    []int
	err      error
}

// killResultMsg contains the result of kill operations
type killResultMsg struct {
	success []int
	failed  map[int]error
}

// windowSizeMsg signals a terminal resize
type windowSizeMsg struct {
	width  int
	height int
}
