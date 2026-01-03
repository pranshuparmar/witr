//go:build linux

package proc

import (
	"fmt"
	"os"
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
	fdCount := 0
	var fdLimit uint64

	// Read memory info from /proc/[pid]/statm
	if statmData, err := os.ReadFile(fmt.Sprintf("/proc/%d/statm", pid)); err == nil {
		fields := strings.Fields(string(statmData))
		if len(fields) >= 7 {
			pageSize := uint64(os.Getpagesize())

			// statm fields: total resident shared text lib data dirty
			total, _ := strconv.ParseUint(fields[0], 10, 64)
			resident, _ := strconv.ParseUint(fields[1], 10, 64)
			shared, _ := strconv.ParseUint(fields[2], 10, 64)
			text, _ := strconv.ParseUint(fields[3], 10, 64)
			lib, _ := strconv.ParseUint(fields[4], 10, 64)
			data, _ := strconv.ParseUint(fields[5], 10, 64)
			dirty, _ := strconv.ParseUint(fields[6], 10, 64)

			memInfo = model.MemoryInfo{
				VMS:    total * pageSize,
				RSS:    resident * pageSize,
				VMSMB:  float64(total*pageSize) / (1024 * 1024),
				RSSMB:  float64(resident*pageSize) / (1024 * 1024),
				Shared: shared * pageSize,
				Text:   text * pageSize,
				Lib:    lib * pageSize,
				Data:   data * pageSize,
				Dirty:  dirty * pageSize,
			}
		}
	}

	// Read I/O stats from /proc/[pid]/io
	if ioData, err := os.ReadFile(fmt.Sprintf("/proc/%d/io", pid)); err == nil {
		lines := strings.Split(string(ioData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "read_bytes:") {
				if val, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "read_bytes:")), 10, 64); err == nil {
					ioStats.ReadBytes = val
				}
			} else if strings.HasPrefix(line, "write_bytes:") {
				if val, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "write_bytes:")), 10, 64); err == nil {
					ioStats.WriteBytes = val
				}
			} else if strings.HasPrefix(line, "syscr:") {
				if val, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "syscr:")), 10, 64); err == nil {
					ioStats.ReadOps = val
				}
			} else if strings.HasPrefix(line, "syscw:") {
				if val, err := strconv.ParseUint(strings.TrimSpace(strings.TrimPrefix(line, "syscw:")), 10, 64); err == nil {
					ioStats.WriteOps = val
				}
			}
		}
	}

	// Read file descriptors from /proc/[pid]/fd
	if fdDir, err := os.ReadDir(fmt.Sprintf("/proc/%d/fd", pid)); err == nil {
		fdCount = len(fdDir)
		for _, fdEntry := range fdDir {
			fdPath := fmt.Sprintf("/proc/%d/fd/%s", pid, fdEntry.Name())
			if linkTarget, err := os.Readlink(fdPath); err == nil {
				fileDescs = append(fileDescs, fmt.Sprintf("%s -> %s", fdEntry.Name(), linkTarget))
			}
		}
	}

	// Get file descriptor limit
	if limitsData, err := os.ReadFile(fmt.Sprintf("/proc/%d/limits", pid)); err == nil {
		lines := strings.Split(string(limitsData), "\n")
		for _, line := range lines {
			if strings.Contains(line, "Max open files") {
				fields := strings.Fields(line)
				if len(fields) >= 2 {
					if limit, err := strconv.ParseUint(strings.ReplaceAll(fields[1], "-", "0"), 10, 64); err == nil {
						fdLimit = limit
					}
				}
				break
			}
		}
	}

	// Find child processes
	if procEntries, err := os.ReadDir("/proc"); err == nil {
		for _, entry := range procEntries {
			if childPID, err := strconv.Atoi(entry.Name()); err == nil {
				if statData, err := os.ReadFile(fmt.Sprintf("/proc/%d/stat", childPID)); err == nil {
					fields := strings.Fields(string(statData))
					if len(fields) > 3 {
						ppid, _ := strconv.Atoi(fields[3])
						if ppid == pid {
							children = append(children, childPID)
						}
					}
				}
			}
		}
	}

	// Get thread count from /proc/[pid]/status
	if statusData, err := os.ReadFile(fmt.Sprintf("/proc/%d/status", pid)); err == nil {
		lines := strings.Split(string(statusData), "\n")
		for _, line := range lines {
			if strings.HasPrefix(line, "Threads:") {
				if count, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, "Threads:"))); err == nil {
					threadCount = count
				}
				break
			}
		}
	}

	return memInfo, ioStats, fileDescs, fdCount, fdLimit, children, threadCount, nil
}
