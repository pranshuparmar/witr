//go:build windows

package proc

import (
	"os"
	"path/filepath"

	"github.com/pranshuparmar/witr/pkg/model"
)

func ReadProcess(pid int) (model.Process, error) {
	info, err := GetProcessDetailedInfo(pid)
	if err != nil {
		return model.Process{}, err
	}

	name := ""
	if info.Exe != "" {
		name = filepath.Base(info.Exe)
	}

	procSockets := GetSocketsForPID(pid)
	serviceName := detectWindowsServiceSource(pid)
	container := detectContainerFromCmdline(info.CommandLine)
	gitRepo, gitBranch := detectGitInfo(info.Cwd)

	return model.Process{
		PID:        pid,
		PPID:       info.PPID,
		Command:    name,
		Cmdline:    info.CommandLine,
		Exe:        info.Exe,
		StartedAt:  info.StartedAt,
		User:       readUser(pid),
		WorkingDir: info.Cwd,
		GitRepo:    gitRepo,
		GitBranch:  gitBranch,
		Sockets:    procSockets,
		Health:     "healthy",
		Forked:     "unknown",
		Env:        info.Env,
		Service:    serviceName,
		Container:  container,
		ExeDeleted: isWindowsBinaryDeleted(info.Exe),
	}, nil
}

func isWindowsBinaryDeleted(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// detectWindowsServiceSource returns the Windows service name that owns the PID, if any.
func detectWindowsServiceSource(pid int) string {
	services, err := serviceMapForPIDs()
	if err != nil {
		return ""
	}
	return services[pid]
}
