//go:build windows

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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

	ports, addrs := GetListeningPortsForPID(pid)
	serviceName := detectWindowsServiceSource(pid)

	return model.Process{
		PID:            pid,
		PPID:           info.PPID,
		Command:        name,
		Cmdline:        info.CommandLine,
		Exe:            info.Exe,
		StartedAt:      info.StartedAt,
		User:           readUser(pid),
		WorkingDir:     info.Cwd,
		ListeningPorts: ports,
		BindAddresses:  addrs,
		Health:         "healthy",
		Forked:         "unknown",
		Env:            info.Env,
		Service:        serviceName,
		ExeDeleted:     isWindowsBinaryDeleted(info.Exe),
	}, nil
}

func isWindowsBinaryDeleted(path string) bool {
	if path == "" {
		return false
	}
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// detectWindowsServiceSource checks if a PID belongs to a Windows Service via Get-CimInstance.
// Keeping this as a fallback/auxiliary check for now.
func detectWindowsServiceSource(pid int) string {
	psScript := fmt.Sprintf("Get-CimInstance -ClassName Win32_Service -Filter \"ProcessId=%d\" | Select-Object -ExpandProperty Name", pid)
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", psScript)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(out))
}
