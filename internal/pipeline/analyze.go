package pipeline

import (
	"sort"
	"strconv"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/pkg/model"
)

type AnalyzeConfig struct {
	PID     int
	Verbose bool
	Tree    bool
	Target  model.Target
}

func AnalyzePID(cfg AnalyzeConfig) (model.Result, error) {
	ancestry, err := procpkg.ResolveAncestry(cfg.PID)
	if err != nil {
		return model.Result{}, err
	}

	src := source.Detect(ancestry)

	var proc model.Process
	resolvedTarget := "unknown"
	if len(ancestry) > 0 {
		proc = ancestry[len(ancestry)-1]
		resolvedTarget = proc.Command
	}

	// Collect child PIDs once and reuse for both extended info and tree output
	var childPIDs []int
	var childProcesses []model.Process
	if (cfg.Verbose || cfg.Tree) && proc.PID > 0 {
		snapshot, err := procpkg.ListProcessSnapshot()
		if err == nil {
			for _, p := range snapshot {
				if p.PPID == proc.PID {
					childPIDs = append(childPIDs, p.PID)
					childProcesses = append(childProcesses, p)
				}
			}
			sort.Slice(childProcesses, func(i, j int) bool {
				return childProcesses[i].PID < childProcesses[j].PID
			})
			sort.Ints(childPIDs)
		}
	}

	if cfg.Verbose && len(ancestry) > 0 {
		memInfo, ioStats, fileDescs, fdCount, fdLimit, threadCount, err := procpkg.ReadExtendedInfo(cfg.PID)
		if err == nil {
			proc.Memory = memInfo
			proc.IO = ioStats
			proc.FileDescs = fileDescs
			proc.FDCount = fdCount
			proc.FDLimit = fdLimit
			proc.Children = childPIDs
			proc.ThreadCount = threadCount
			ancestry[len(ancestry)-1] = proc
		}
	}

	var resCtx *model.ResourceContext
	var fileCtx *model.FileContext
	if cfg.Verbose {
		resCtx = procpkg.GetResourceContext(cfg.PID)
		fileCtx = procpkg.GetFileContext(cfg.PID)
	}

	restartCount := 0
	if src.Type == model.SourceSystemd {
		if v, ok := src.Details["NRestarts"]; ok {
			if count, err := strconv.Atoi(v); err == nil {
				restartCount = count
			}
		}
	}

	res := model.Result{
		Target:          cfg.Target,
		ResolvedTarget:  resolvedTarget,
		Process:         proc,
		RestartCount:    restartCount,
		Ancestry:        ancestry,
		Source:          src,
		Warnings:        source.Warnings(ancestry, src.Type),
		ResourceContext: resCtx,
		FileContext:     fileCtx,
		Children:        childProcesses,
	}

	return res, nil
}
