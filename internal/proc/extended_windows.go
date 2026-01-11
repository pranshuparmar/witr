//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ReadExtendedInfo reads extended process information for verbose output
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, []int, int, error) {
	var memInfo model.MemoryInfo
	var ioStats model.IOStats
	var fileDescs []string
	var children []int
	var threadCount int
	var fdCount int
	var fdLimit uint64

	// Use wmic to get process details
	cmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ProcessId=%d", pid), "get", "HandleCount,ReadOperationCount,ReadTransferCount,ThreadCount,VirtualSize,WorkingSetSize,WriteOperationCount,WriteTransferCount", "/format:list")
	out, err := cmd.Output()
	if err != nil {
		return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, fmt.Errorf("wmic extended info: %w", err)
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])

		switch key {
		case "ReadOperationCount":
			ioStats.ReadOps, _ = strconv.ParseUint(val, 10, 64)
		case "ReadTransferCount":
			ioStats.ReadBytes, _ = strconv.ParseUint(val, 10, 64)
		case "WriteOperationCount":
			ioStats.WriteOps, _ = strconv.ParseUint(val, 10, 64)
		case "WriteTransferCount":
			ioStats.WriteBytes, _ = strconv.ParseUint(val, 10, 64)
		case "ThreadCount":
			threadCount, _ = strconv.Atoi(val)
		case "VirtualSize":
			memInfo.VMS, _ = strconv.ParseUint(val, 10, 64)
			memInfo.VMSMB = float64(memInfo.VMS) / (1024 * 1024)
		case "WorkingSetSize":
			memInfo.RSS, _ = strconv.ParseUint(val, 10, 64)
			memInfo.RSSMB = float64(memInfo.RSS) / (1024 * 1024)
		case "HandleCount":
			fdCount, _ = strconv.Atoi(val)
		}
	}

	childCmd := exec.Command("wmic", "process", "where", fmt.Sprintf("ParentProcessId=%d", pid), "get", "ProcessId", "/format:csv")
	if childOut, err := childCmd.Output(); err == nil {
		childLines := strings.Split(string(childOut), "\n")
		for _, line := range childLines {
			line = strings.TrimSpace(line)
			if line == "" || strings.HasPrefix(line, "Node") {
				continue
			}
			parts := strings.Split(line, ",")
			if len(parts) >= 2 {
				if cpid, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1])); err == nil {
					children = append(children, cpid)
				}
			}
		}
	}

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, nil
}
