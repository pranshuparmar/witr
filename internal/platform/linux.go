//go:build linux

package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	proc "github.com/pranshuparmar/witr/internal/linux/proc"
	"github.com/pranshuparmar/witr/pkg/model"
)

func init() {
	Current = &linuxPlatform{}
}

type linuxPlatform struct{}

func (l *linuxPlatform) ReadProcess(pid int) (model.Process, error) {
	return proc.ReadProcess(pid)
}

func (l *linuxPlatform) ListPIDs() ([]int, error) {
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return nil, err
	}

	var pids []int
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}
		pids = append(pids, pid)
	}
	return pids, nil
}

func (l *linuxPlatform) BootTime() time.Time {
	f, err := os.Open("/proc/stat")
	if err != nil {
		return time.Now()
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "btime") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				sec, _ := strconv.ParseInt(parts[1], 10, 64)
				return time.Unix(sec, 0)
			}
		}
	}
	return time.Now()
}

// ResolvePort finds PIDs listening on a port via /proc/net/tcp + socket inode matching.
func (l *linuxPlatform) ResolvePort(port int) ([]int, error) {
	inodes, err := findSocketInodes(port)
	if err != nil {
		return nil, err
	}

	pidSet := make(map[int]bool)
	procEntries, _ := os.ReadDir("/proc")
	for _, entry := range procEntries {
		pid, err := strconv.Atoi(entry.Name())
		if err != nil {
			continue
		}

		fdDir := filepath.Join("/proc", entry.Name(), "fd")
		fds, err := os.ReadDir(fdDir)
		if err != nil {
			continue
		}

		for _, fd := range fds {
			link, err := os.Readlink(filepath.Join(fdDir, fd.Name()))
			if err != nil {
				continue
			}

			if strings.HasPrefix(link, "socket:[") {
				inode := strings.TrimSuffix(strings.TrimPrefix(link, "socket:["), "]")
				if inodes[inode] {
					pidSet[pid] = true
				}
			}
		}
	}

	// Return lowest PID (the main listener)
	var minPID int
	for pid := range pidSet {
		if minPID == 0 || pid < minPID {
			minPID = pid
		}
	}
	if minPID > 0 {
		return []int{minPID}, nil
	}

	return nil, fmt.Errorf("socket found but owning process not detected")
}

func findSocketInodes(port int) (map[string]bool, error) {
	inodes := make(map[string]bool)
	files := []string{"/proc/net/tcp", "/proc/net/tcp6"}
	targetHex := fmt.Sprintf("%04X", port)

	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		lines := strings.Split(string(data), "\n")
		for _, line := range lines[1:] {
			fields := strings.Fields(line)
			if len(fields) < 10 {
				continue
			}

			localAddr := fields[1]
			parts := strings.Split(localAddr, ":")
			if len(parts) != 2 {
				continue
			}

			if parts[1] == targetHex {
				inodes[fields[9]] = true
			}
		}
	}

	if len(inodes) == 0 {
		return nil, fmt.Errorf("no process listening on port %d", port)
	}

	return inodes, nil
}

// ResolveName finds PIDs by process name via /proc scan + systemctl.
func (l *linuxPlatform) ResolveName(name string) ([]int, error) {
	var procPIDs []int
	lowerName := strings.ToLower(name)
	selfPid := os.Getpid()
	parentPid := os.Getppid()

	entries, _ := os.ReadDir("/proc")
	for _, e := range entries {
		pid, err := strconv.Atoi(e.Name())
		if err != nil {
			continue
		}

		// Prevent matching PID itself as name
		if lowerName == strconv.Itoa(pid) {
			continue
		}

		// Exclude self and parent
		if pid == selfPid || pid == parentPid {
			continue
		}

		comm, err := os.ReadFile("/proc/" + e.Name() + "/comm")
		if err == nil {
			if strings.Contains(strings.ToLower(strings.TrimSpace(string(comm))), lowerName) {
				if !strings.Contains(strings.ToLower(string(comm)), "grep") {
					procPIDs = append(procPIDs, pid)
				}
				continue
			}
		}

		cmdline, err := os.ReadFile("/proc/" + e.Name() + "/cmdline")
		if err == nil {
			cmd := strings.ReplaceAll(string(cmdline), "\x00", " ")
			if strings.Contains(strings.ToLower(cmd), lowerName) &&
				!strings.Contains(strings.ToLower(cmd), "grep") &&
				!strings.Contains(strings.ToLower(cmd), "witr") {
				procPIDs = append(procPIDs, pid)
			}
		}
	}

	// Try systemd service
	servicePID, _ := resolveSystemdServiceMainPID(name)

	// Collect unique PIDs
	uniquePIDs := map[int]bool{}
	if servicePID > 0 {
		uniquePIDs[servicePID] = true
	}
	for _, pid := range procPIDs {
		uniquePIDs[pid] = true
	}

	if len(uniquePIDs) > 1 {
		fmt.Printf("Ambiguous target: \"%s\"\n\n", name)
		fmt.Println("The name matches multiple entities:")
		fmt.Println()
		if servicePID > 0 {
			fmt.Printf("[1] PID %d   %s: master process   (service)\n", servicePID, name)
		}
		idx := 2
		for _, pid := range procPIDs {
			if pid == servicePID {
				continue
			}
			fmt.Printf("[%d] PID %d   %s: worker process   (manual)\n", idx, pid, name)
			idx++
		}
		fmt.Println()
		fmt.Println("witr cannot determine intent safely.")
		fmt.Println("Please re-run with an explicit PID:")
		fmt.Println("  witr --pid <pid>")
		os.Exit(1)
	}

	if servicePID > 0 {
		return []int{servicePID}, nil
	}

	if len(procPIDs) > 0 {
		return procPIDs, nil
	}

	return nil, fmt.Errorf("no running process or service named %q", name)
}

func resolveSystemdServiceMainPID(name string) (int, error) {
	svcName := name
	if !strings.HasSuffix(svcName, ".service") {
		svcName += ".service"
	}
	out, err := exec.Command("systemctl", "show", svcName, "-p", "MainPID", "--value").Output()
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
