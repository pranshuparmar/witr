package record

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// Stepper manages stepping through a chronological series of snapshots.
type Stepper struct {
	snapshots []*model.Snapshot
	index     int
}

// NewStepper constructs a Stepper over a non-empty list of snapshots.
func NewStepper(snapshots []*model.Snapshot) (*Stepper, error) {
	if len(snapshots) == 0 {
		return nil, errors.New("cannot create stepper with zero snapshots")
	}
	return &Stepper{
		snapshots: snapshots,
		index:     0,
	}, nil
}

// Count returns the total number of captured snapshots.
func (s *Stepper) Count() int {
	return len(s.snapshots)
}

// CurrentIndex returns the 0-based index of the currently active snapshot.
func (s *Stepper) CurrentIndex() int {
	return s.index
}

// Current returns the currently active snapshot.
func (s *Stepper) Current() *model.Snapshot {
	return s.snapshots[s.index]
}

// Next moves to the next snapshot in the series. Returns false if already at the end.
func (s *Stepper) Next() (*model.Snapshot, bool) {
	if s.index+1 >= len(s.snapshots) {
		return s.Current(), false
	}
	s.index++
	return s.Current(), true
}

// Prev moves to the previous snapshot in the series. Returns false if already at the start.
func (s *Stepper) Prev() (*model.Snapshot, bool) {
	if s.index <= 0 {
		return s.Current(), false
	}
	s.index--
	return s.Current(), true
}

// StepTo jumps to the specified 0-based snapshot index.
func (s *Stepper) StepTo(idx int) (*model.Snapshot, error) {
	if idx < 0 || idx >= len(s.snapshots) {
		return nil, fmt.Errorf("step index %d out of range [0, %d)", idx, len(s.snapshots))
	}
	s.index = idx
	return s.Current(), nil
}

// Snapshots returns the complete list of snapshots.
func (s *Stepper) Snapshots() []*model.Snapshot {
	return s.snapshots
}

// SeekTimestamp selects the snapshot whose timestamp is closest to target.
func (s *Stepper) SeekTimestamp(target time.Time) (*model.Snapshot, int) {
	bestIdx := 0
	minDiff := time.Duration(math.MaxInt64)

	for i, snap := range s.snapshots {
		diff := snap.Timestamp.Sub(target)
		if diff < 0 {
			diff = -diff
		}
		if diff < minDiff {
			minDiff = diff
			bestIdx = i
		}
	}

	s.index = bestIdx
	return s.snapshots[bestIdx], bestIdx
}

// SeekTimestampString parses time string formats and jumps to the closest snapshot.
func (s *Stepper) SeekTimestampString(raw string) (*model.Snapshot, int, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, s.index, errors.New("empty timestamp")
	}

	formats := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.999999999",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
	}

	for _, layout := range formats {
		if t, err := time.Parse(layout, raw); err == nil {
			snap, idx := s.SeekTimestamp(t)
			return snap, idx, nil
		}
	}

	// Try time-of-day layouts matching today or the first snapshot's date
	timeOnlyFormats := []string{"15:04:05.999999999", "15:04:05", "15:04"}
	baseDate := s.snapshots[0].Timestamp
	for _, layout := range timeOnlyFormats {
		if tod, err := time.Parse(layout, raw); err == nil {
			target := time.Date(baseDate.Year(), baseDate.Month(), baseDate.Day(),
				tod.Hour(), tod.Minute(), tod.Second(), tod.Nanosecond(), baseDate.Location())
			snap, idx := s.SeekTimestamp(target)
			return snap, idx, nil
		}
	}

	return nil, s.index, fmt.Errorf("unable to parse timestamp %q", raw)
}

// ResolveTarget finds matching PIDs within the current snapshot.
func (s *Stepper) ResolveTarget(t model.Target, exact bool) ([]int, error) {
	snap := s.Current()
	val := strings.TrimSpace(t.Value)

	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid pid %q", val)
		}
		for _, p := range snap.Processes {
			if p.PID == pid {
				return []int{pid}, nil
			}
		}
		return nil, fmt.Errorf("no process found with pid %d in snapshot", pid)

	case model.TargetName:
		return s.resolveNameInSnapshot(val, exact)

	case model.TargetPort:
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid port %q", val)
		}
		return s.resolvePortInSnapshot(port)

	case model.TargetFile:
		return s.resolveFileInSnapshot(val)

	case model.TargetContainer:
		return s.resolveContainerInSnapshot(val, exact)

	default:
		return nil, fmt.Errorf("unsupported target type %q", t.Type)
	}
}

func (s *Stepper) resolveNameInSnapshot(name string, exact bool) ([]int, error) {
	snap := s.Current()
	var exactMatches []int
	var partialMatches []int

	lowerName := strings.ToLower(name)

	for _, p := range snap.Processes {
		cmdMatch := p.Command == name
		tokenMatch := matchesToken(p.Cmdline, name)

		if cmdMatch || tokenMatch {
			exactMatches = append(exactMatches, p.PID)
			continue
		}

		if !exact {
			if strings.Contains(strings.ToLower(p.Command), lowerName) ||
				strings.Contains(strings.ToLower(p.Cmdline), lowerName) {
				partialMatches = append(partialMatches, p.PID)
			}
		}
	}

	if len(exactMatches) > 0 {
		slices.Sort(exactMatches)
		return exactMatches, nil
	}
	if len(partialMatches) > 0 {
		slices.Sort(partialMatches)
		return partialMatches, nil
	}
	return nil, fmt.Errorf("no running process found matching %q", name)
}

func (s *Stepper) resolvePortInSnapshot(port int) ([]int, error) {
	snap := s.Current()
	var pids []int
	seen := make(map[int]bool)

	// Check Ports table
	for _, op := range snap.Ports {
		if op.Port == port && op.PID > 0 {
			if !seen[op.PID] {
				seen[op.PID] = true
				pids = append(pids, op.PID)
			}
		}
	}

	// Also check sockets embedded in processes
	for _, p := range snap.Processes {
		for _, sock := range p.Sockets {
			if sock.Port == port {
				if !seen[p.PID] {
					seen[p.PID] = true
					pids = append(pids, p.PID)
				}
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process listening on port %d in snapshot", port)
	}
	slices.Sort(pids)
	return pids, nil
}

func (s *Stepper) resolveFileInSnapshot(path string) ([]int, error) {
	snap := s.Current()
	var pids []int
	seen := make(map[int]bool)

	for _, lf := range snap.LockedFiles {
		if lf.Path == path || strings.HasSuffix(lf.Path, path) {
			if !seen[lf.PID] {
				seen[lf.PID] = true
				pids = append(pids, lf.PID)
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no process holding file %q in snapshot", path)
	}
	slices.Sort(pids)
	return pids, nil
}

func (s *Stepper) resolveContainerInSnapshot(name string, exact bool) ([]int, error) {
	snap := s.Current()
	var pids []int
	seen := make(map[int]bool)
	lower := strings.ToLower(name)

	for _, c := range snap.Containers {
		matched := false
		if exact {
			matched = c.Name == name || c.ID == name || strings.HasPrefix(c.ID, name)
		} else {
			matched = strings.Contains(strings.ToLower(c.Name), lower) ||
				strings.Contains(strings.ToLower(c.ID), lower)
		}
		if matched {
			for _, p := range snap.Processes {
				if p.ContainerID == c.ID || p.Container == c.Name {
					if !seen[p.PID] {
						seen[p.PID] = true
						pids = append(pids, p.PID)
					}
				}
			}
		}
	}

	if len(pids) == 0 {
		return nil, fmt.Errorf("no container process found matching %q in snapshot", name)
	}
	slices.Sort(pids)
	return pids, nil
}

// ResolveAncestry walks the parent chain for a PID using only data in the current snapshot.
func (s *Stepper) ResolveAncestry(pid int) ([]model.Process, error) {
	snap := s.Current()
	procMap := make(map[int]model.Process, len(snap.Processes))
	for _, p := range snap.Processes {
		procMap[p.PID] = p
	}

	var chain []model.Process
	seen := make(map[int]bool)
	current := pid

	for current > 0 {
		if seen[current] {
			break // Loop detected
		}
		seen[current] = true

		p, exists := procMap[current]
		if !exists {
			if len(chain) > 0 {
				chain[len(chain)-1].ParentExited = true
			}
			break
		}

		if len(chain) > 0 {
			child := &chain[len(chain)-1]
			if !p.StartedAt.IsZero() && !child.StartedAt.IsZero() && p.StartedAt.After(child.StartedAt) {
				child.ParentExited = true
				break
			}
		}

		chain = append(chain, p)
		if p.PPID == 0 || p.PID == 1 {
			break
		}
		current = p.PPID
	}

	if len(chain) == 0 {
		return nil, fmt.Errorf("process %d does not exist in snapshot", pid)
	}

	// Reverse chain: root ancestor first, target process last
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	return chain, nil
}

func matchesToken(cmdline, name string) bool {
	for _, part := range strings.Fields(cmdline) {
		if part == name {
			return true
		}
		normalized := strings.ReplaceAll(part, "\\", "/")
		for _, seg := range strings.Split(normalized, "/") {
			if seg == name {
				return true
			}
		}
	}
	return false
}
