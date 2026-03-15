//go:build linux

package target

import (
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"

	procpkg "github.com/pranshuparmar/witr/internal/proc"
)

func ResolveName(name string, exact bool) ([]int, error) {
	var procPIDs []int

	entries, _ := os.ReadDir("/proc")
	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()

	// Build ignored PID set lazily — only resolve ancestry if we actually
	// need to filter matches (avoids walking the chain on every invocation)
	var ignoredPids map[int]bool
	isIgnored := func(pid int) bool {
		if pid == selfPid {
			return true
		}
		if ignoredPids == nil {
			ignoredPids = make(map[int]bool)
			ignoredPids[selfPid] = true
			if ancestry, err := procpkg.ResolveAncestry(selfPid); err == nil {
				for _, p := range ancestry {
					ignoredPids[p.PID] = true
				}
			}
		}
		return ignoredPids[pid]
	}

	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		if lowerName == strconv.Itoa(pid) {
			continue
		}

		if isIgnored(pid) {
			continue
		}

		comm, err := os.ReadFile("/proc/" + e.Name() + "/comm")
		if err == nil {
			commLower := strings.ToLower(strings.TrimSpace(string(comm)))
			var match bool
			if exact {
				match = commLower == lowerName
			} else {
				match = strings.Contains(commLower, lowerName)
			}
			if match {
				if !strings.Contains(commLower, "grep") {
					procPIDs = append(procPIDs, pid)
				}
				continue
			}
		}

		cmdline, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err == nil {
			cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
			cmdLower := strings.ToLower(cmd)
			var match bool
			if exact {
				parts := strings.Fields(cmdLower)
				for _, part := range parts {
					if part == lowerName {
						match = true
						break
					}
				}
			} else {
				match = strings.Contains(cmdLower, lowerName)
			}
			if match && !strings.Contains(cmdLower, "grep") {
				procPIDs = append(procPIDs, pid)
			}
		}
	}

	servicePID, _ := resolveSystemdServiceMainPID(name)

	seen := map[int]bool{}
	var procUnique []int
	for _, pid := range procPIDs {
		if pid == servicePID || seen[pid] {
			continue
		}
		seen[pid] = true
		procUnique = append(procUnique, pid)
	}
	sort.Ints(procUnique)

	var pids []int
	if servicePID > 0 {
		pids = append(pids, servicePID)
	}
	pids = append(pids, procUnique...)

	if len(pids) == 0 {
		return nil, fmt.Errorf("no running process or service named %q", name)
	}
	return pids, nil
}

// resolveSystemdServiceMainPID tries to resolve a systemd service and returns its MainPID if running.
func resolveSystemdServiceMainPID(name string) (int, error) {
	// Accept both foo and foo.service
	svcName := name
	if !strings.HasSuffix(svcName, ".service") {
		svcName += ".service"
	}
	out, err := exec.Command("systemctl", "show", "-p", "MainPID", "--value", "--", svcName).Output()
	if err != nil {
		return 0, err
	}
	pidStr := strings.TrimSpace(string(out))
	pid, err := strconv.Atoi(pidStr)
	if err != nil || pid == 0 {
		return 0, fmt.Errorf("service %q not running", svcName)
	}
	return pid, nil
}
