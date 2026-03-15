//go:build windows

package proc

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// ReadExtendedInfo reads extended process information for verbose output.
// Child PID discovery is handled by the caller to avoid redundant process scans.
func ReadExtendedInfo(pid int) (model.MemoryInfo, model.IOStats, []string, int, uint64, int, error) {
	var memInfo model.MemoryInfo
	var ioStats model.IOStats
	var fileDescs []string
	var threadCount int
	var fdCount int
	var fdLimit uint64

	// Use powershell to get process details
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Process -Filter \"ProcessId=%d\" | ForEach-Object { \"HandleCount=$($_.HandleCount)\"; \"ReadOperationCount=$($_.ReadOperationCount)\"; \"ReadTransferCount=$($_.ReadTransferCount)\"; \"ThreadCount=$($_.ThreadCount)\"; \"VirtualSize=$($_.VirtualSize)\"; \"WorkingSetSize=$($_.WorkingSetSize)\"; \"WriteOperationCount=$($_.WriteOperationCount)\"; \"WriteTransferCount=$($_.WriteTransferCount)\" }", pid)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", psScript)
	out, err := cmd.Output()
	if err != nil {
		return memInfo, ioStats, fileDescs, fdCount, fdLimit, threadCount, fmt.Errorf("powershell extended info: %w", err)
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

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, threadCount, nil
}
