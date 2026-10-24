package record

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"time"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/pkg/model"
)

// SessionSummary summarizes high-level metrics across a recorded snapshot session.
type SessionSummary struct {
	TotalSamples int           `json:"total_samples"`
	StartTime    time.Time     `json:"start_time"`
	EndTime      time.Time     `json:"end_time"`
	Duration     time.Duration `json:"duration"`
	MinProcesses int           `json:"min_processes"`
	MaxProcesses int           `json:"max_processes"`
	AvgProcesses float64       `json:"avg_processes"`
}

// SnapshotOverview provides a per-sample overview entry.
type SnapshotOverview struct {
	Index        int       `json:"index"`
	Timestamp    time.Time `json:"timestamp"`
	ProcessCount int       `json:"process_count"`
	PortCount    int       `json:"port_count"`
}

// SessionReport combines summary metrics with snapshot listing for serialization.
type SessionReport struct {
	Summary   SessionSummary     `json:"summary"`
	Snapshots []SnapshotOverview `json:"snapshots"`
}

// Summarize aggregates statistical metrics across all snapshots.
func Summarize(snapshots []*model.Snapshot) SessionSummary {
	if len(snapshots) == 0 {
		return SessionSummary{}
	}

	start := snapshots[0].Timestamp
	end := snapshots[len(snapshots)-1].Timestamp
	dur := end.Sub(start)
	if dur < 0 {
		dur = 0
	}

	minP := math.MaxInt32
	maxP := 0
	sumP := 0

	for _, s := range snapshots {
		pCount := len(s.Processes)
		if pCount < minP {
			minP = pCount
		}
		if pCount > maxP {
			maxP = pCount
		}
		sumP += pCount
	}

	return SessionSummary{
		TotalSamples: len(snapshots),
		StartTime:    start,
		EndTime:      end,
		Duration:     dur,
		MinProcesses: minP,
		MaxProcesses: maxP,
		AvgProcesses: float64(sumP) / float64(len(snapshots)),
	}
}

// Overviews builds a slice of lightweight sample overviews.
func Overviews(snapshots []*model.Snapshot) []SnapshotOverview {
	list := make([]SnapshotOverview, len(snapshots))
	for i, s := range snapshots {
		list[i] = SnapshotOverview{
			Index:        i,
			Timestamp:    s.Timestamp,
			ProcessCount: len(s.Processes),
			PortCount:    len(s.Ports),
		}
	}
	return list
}

// RenderSummary prints human-readable session summary information.
func RenderSummary(w io.Writer, s SessionSummary, snapshots []*model.Snapshot, colorEnabled bool) {
	cyan := ""
	green := ""
	dim := ""
	reset := ""
	if colorEnabled {
		cyan = string(output.ColorCyan)
		green = string(output.ColorGreen)
		dim = string(output.ColorDim)
		reset = string(output.ColorReset)
	}

	fmt.Fprintf(w, "%s--- Recorded Session Summary ---%s\n", cyan, reset)
	fmt.Fprintf(w, "  Snapshots:     %s%d%s\n", green, s.TotalSamples, reset)
	fmt.Fprintf(w, "  Time Range:    %s to %s (%s)\n",
		s.StartTime.Format("2006-01-02 15:04:05"),
		s.EndTime.Format("2006-01-02 15:04:05"),
		s.Duration.Round(time.Millisecond))
	fmt.Fprintf(w, "  Process Count: min=%d, max=%d, avg=%.1f\n\n",
		s.MinProcesses, s.MaxProcesses, s.AvgProcesses)

	fmt.Fprintf(w, "%sAvailable Snapshots:%s\n", cyan, reset)
	limit := len(snapshots)
	if limit > 20 {
		limit = 20
	}
	for i := 0; i < limit; i++ {
		snap := snapshots[i]
		fmt.Fprintf(w, "  [%d] %s %s(%d processes, %d ports)%s\n",
			i, snap.Timestamp.Format("2006-01-02 15:04:05"),
			dim, len(snap.Processes), len(snap.Ports), reset)
	}
	if len(snapshots) > limit {
		fmt.Fprintf(w, "  %s... and %d more snapshots%s\n", dim, len(snapshots)-limit, reset)
	}
}

// SummaryJSON marshals session metrics and sample overview to JSON.
func SummaryJSON(s SessionSummary, snapshots []*model.Snapshot) (string, error) {
	rep := SessionReport{
		Summary:   s,
		Snapshots: Overviews(snapshots),
	}
	data, err := json.MarshalIndent(rep, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
