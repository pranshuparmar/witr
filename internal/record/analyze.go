package record

import (
	"fmt"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/internal/pipeline"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/pkg/model"
)

// AnalyzeSnapshot produces a complete model.Result for target PID within the given snapshot.
func AnalyzeSnapshot(snap *model.Snapshot, pid int, cfg pipeline.AnalyzeConfig) (model.Result, error) {
	if snap == nil {
		return model.Result{}, fmt.Errorf("snapshot is nil")
	}

	stepper, err := NewStepper([]*model.Snapshot{snap})
	if err != nil {
		return model.Result{}, err
	}

	ancestry, err := stepper.ResolveAncestry(pid)
	if err != nil {
		return model.Result{}, err
	}

	src := source.Detect(ancestry)

	var targetProc model.Process
	resolvedTarget := "unknown"
	if len(ancestry) > 0 {
		targetProc = ancestry[len(ancestry)-1]
		resolvedTarget = targetProc.Command
	}

	// Match container details if present in snapshot
	var containerMatch *model.ContainerMatch
	if targetProc.ContainerID != "" && len(snap.Containers) > 0 {
		for _, c := range snap.Containers {
			if c.ID == targetProc.ContainerID || c.Name == targetProc.ContainerID {
				containerMatch = c
				break
			}
		}
	} else if targetProc.Container != "" && len(snap.Containers) > 0 {
		for _, c := range snap.Containers {
			if c.Name == targetProc.Container {
				containerMatch = c
				break
			}
		}
	}

	// Gather child processes from the snapshot
	var childProcesses []model.Process
	var childPIDs []int
	for _, p := range snap.Processes {
		if p.PPID == targetProc.PID {
			childProcesses = append(childProcesses, p)
			childPIDs = append(childPIDs, p.PID)
		}
	}
	sort.Slice(childProcesses, func(i, j int) bool {
		return childProcesses[i].PID < childProcesses[j].PID
	})
	slices.Sort(childPIDs)

	if cfg.Verbose {
		targetProc.Children = childPIDs
		if len(ancestry) > 0 {
			ancestry[len(ancestry)-1].Children = childPIDs
		}
	}

	if containerMatch != nil {
		targetProc.Container = output.FormatContainerLine(containerMatch)
		if len(ancestry) > 0 {
			ancestry[len(ancestry)-1].Container = targetProc.Container
		}
	}

	restartCount := 0
	if src.Type == model.SourceSystemd {
		if v, ok := src.Details["NRestarts"]; ok {
			if count, convErr := strconv.Atoi(v); convErr == nil {
				restartCount = count
			}
		}
	} else if src.Type == model.SourceContainer && containerMatch != nil {
		restartCount = containerMatch.RestartCount
	}

	warnings := source.Warnings(ancestry, restartCount, src.Type)

	var socketInfo *model.SocketInfo
	if cfg.Target.Type == model.TargetPort {
		if portNum, convErr := strconv.Atoi(cfg.Target.Value); convErr == nil {
			for _, op := range snap.Ports {
				if op.Port == portNum && op.PID == pid {
					socketInfo = &model.SocketInfo{
						Port:       op.Port,
						State:      op.State,
						LocalAddr:  op.Address,
						RemoteAddr: op.RemoteAddress,
					}
					break
				}
			}
		}
	}

	res := model.Result{
		Target:         cfg.Target,
		ResolvedTarget: strings.TrimSpace(resolvedTarget),
		Process:        targetProc,
		RestartCount:   restartCount,
		Ancestry:       ancestry,
		Source:         src,
		Warnings:       warnings,
		Container:      containerMatch,
		SocketInfo:     socketInfo,
		Children:       childProcesses,
	}

	return res, nil
}
