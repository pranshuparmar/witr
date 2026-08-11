package gui

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/internal/pipeline"
	procpkg "github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/source"
	"github.com/pranshuparmar/witr/internal/target"
	"github.com/pranshuparmar/witr/pkg/model"
	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct manages Wails application lifecycle and exposes API methods to frontend.
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// Startup is called when the app starts. The context is saved so we can call runtime methods.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
}

// Context returns the application context saved on startup.
func (a *App) Context() context.Context {
	return a.ctx
}

// GUIResult wraps model.Result and adds rich structured fields for the GUI inspector.
type GUIResult struct {
	model.Result
	WhyExplanation   string            `json:"whyExplanation"`
	StartedFormatted string            `json:"startedFormatted"`
	WorkingDir       string            `json:"workingDir"`
	SocketsList      []string          `json:"socketsList"`
	CPUFormatted     string            `json:"cpuFormatted"`
	MemoryVirtual    string            `json:"memoryVirtual"`
	MemoryResident   string            `json:"memoryResident"`
	MemoryPrivate    string            `json:"memoryPrivate"`
	IORead           string            `json:"ioRead"`
	IOWrite          string            `json:"ioWrite"`
	HandlesCount     string            `json:"handlesCount"`
	ThreadCount      string            `json:"threadCount"`
	EnvVars          map[string]string `json:"envVars"`
}

// GUIProcessItem extends model.Process with pre-calculated system, memory, CPU & network flags for 100% table consistency.
type GUIProcessItem struct {
	model.Process
	CPUFormatted  string `json:"cpuFormatted"`
	MemFormatted  string `json:"memFormatted"`
	MemPercentStr string `json:"memPercentStr"`
	StartedStr    string `json:"startedStr"`
	IsSystem      bool   `json:"isSystem"`
	HasSockets    bool   `json:"hasSockets"`
}

// SystemAnalytics contains high-level live process metrics.
type SystemAnalytics struct {
	TotalProcesses  int `json:"totalProcesses"`
	ListeningPorts  int `json:"listeningPorts"`
	SystemProcesses int `json:"systemProcesses"`
	UserProcesses   int `json:"userProcesses"`
}

// isSystemProcess determines if a process is a Windows background system service/component.
func isSystemProcess(p model.Process) bool {
	if p.PID <= 1000 {
		return true
	}
	u := strings.ToUpper(p.User)
	if u == "" || strings.Contains(u, "SYSTEM") || strings.Contains(u, "NT AUTHORITY") || strings.Contains(u, "LOCAL SERVICE") || strings.Contains(u, "NETWORK SERVICE") || strings.HasPrefix(u, "S-1-5-") {
		return true
	}
	cmd := strings.ToLower(p.Command)
	systemCmds := map[string]bool{
		"csrss.exe": true, "services.exe": true, "lsass.exe": true, "svchost.exe": true,
		"smss.exe": true, "wininit.exe": true, "winlogon.exe": true, "spoolsv.exe": true,
		"fontdrvhost.exe": true, "dwm.exe": true, "ctfmon.exe": true, "sihost.exe": true,
		"taskhostw.exe": true, "searchindexer.exe": true, "securityhealthservice.exe": true,
		"system": true, "registry": true, "memory compression": true, "secure system": true,
	}
	return systemCmds[cmd]
}

// GenerateWhyExplanation builds a clear, plain-English causality statement.
func GenerateWhyExplanation(res *model.Result) string {
	if res == nil {
		return ""
	}
	p := res.Process
	anc := res.Ancestry
	src := res.Source

	var parentName string
	if len(anc) > 1 {
		parentName = anc[len(anc)-2].Command
	}

	targetStr := p.Command
	if targetStr == "" {
		targetStr = res.ResolvedTarget
	}

	switch src.Type {
	case model.SourceContainer:
		img := src.Details["image"]
		if img != "" {
			return fmt.Sprintf("Process '%s' (PID %d) is running inside container '%s' (image: %s).", targetStr, p.PID, src.Name, img)
		}
		return fmt.Sprintf("Process '%s' (PID %d) is running inside container '%s'.", targetStr, p.PID, src.Name)
	case model.SourceSystemd:
		svc := src.Details["Unit"]
		if svc == "" {
			svc = src.Name
		}
		return fmt.Sprintf("Process '%s' (PID %d) was started automatically by systemd service '%s'.", targetStr, p.PID, svc)
	case model.SourceWindowsService:
		return fmt.Sprintf("Process '%s' (PID %d) is running as a background Windows Service managed by services.exe.", targetStr, p.PID)
	case model.SourceShell:
		if parentName != "" {
			return fmt.Sprintf("Process '%s' (PID %d) was launched from command-line shell '%s' by user '%s'.", targetStr, p.PID, parentName, p.User)
		}
		return fmt.Sprintf("Process '%s' (PID %d) was started interactively from a terminal shell.", targetStr, p.PID)
	case model.SourceSSH:
		return fmt.Sprintf("Process '%s' (PID %d) was spawned by a remote SSH connection.", targetStr, p.PID)
	case model.SourceCron:
		return fmt.Sprintf("Process '%s' (PID %d) was triggered by a scheduled cron job.", targetStr, p.PID)
	case model.SourceSupervisor:
		return fmt.Sprintf("Process '%s' (PID %d) is managed by process supervisor '%s'.", targetStr, p.PID, src.Name)
	default:
		if parentName != "" {
			return fmt.Sprintf("Process '%s' (PID %d) was spawned by parent process '%s' (PPID %d).", targetStr, p.PID, parentName, p.PPID)
		}
		if p.User != "" {
			return fmt.Sprintf("Process '%s' (PID %d) is executing under user account '%s'.", targetStr, p.PID, p.User)
		}
		return fmt.Sprintf("Process '%s' (PID %d) is active on the system.", targetStr, p.PID)
	}
}

// EnrichGUIResult populates full extended metrics & environment variables.
func EnrichGUIResult(res *model.Result) *GUIResult {
	if res == nil {
		return nil
	}
	p := res.Process

	// Format Started time
	startedStr := "N/A"
	if !p.StartedAt.IsZero() {
		dur := time.Since(p.StartedAt)
		days := int(dur.Hours() / 24)
		if days > 0 {
			startedStr = fmt.Sprintf("%d days ago (%s)", days, p.StartedAt.Format("Mon 2006-01-02 15:04:05"))
		} else if dur.Hours() >= 1 {
			startedStr = fmt.Sprintf("%d hours ago (%s)", int(dur.Hours()), p.StartedAt.Format("15:04:05"))
		} else {
			startedStr = fmt.Sprintf("%d mins ago (%s)", int(dur.Minutes()), p.StartedAt.Format("15:04:05"))
		}
	}

	// Working dir
	workDir := p.WorkingDir
	if workDir == "" {
		workDir = "N/A"
	}

	// Sockets list
	var socketsList []string
	for _, s := range p.Sockets {
		sockStr := fmt.Sprintf("%s:%d", s.Address, s.Port)
		if s.State != "" {
			sockStr = fmt.Sprintf("%s (%s | %s)", sockStr, s.Protocol, s.State)
		} else if s.Protocol != "" {
			sockStr = fmt.Sprintf("%s (%s)", sockStr, s.Protocol)
		}
		socketsList = append(socketsList, sockStr)
	}
	if res.SocketInfo != nil {
		sockStr := fmt.Sprintf("%s -> %s (%s)", res.SocketInfo.LocalAddr, res.SocketInfo.RemoteAddr, res.SocketInfo.State)
		socketsList = append(socketsList, sockStr)
	}

	// CPU & Memory
	cpuStr := fmt.Sprintf("%.1f%%", p.CPUPercent)
	memVirt := fmt.Sprintf("%.1f MB", p.Memory.VMSMB)
	if p.Memory.VMSMB == 0 && p.MemoryRSS > 0 {
		memVirt = fmt.Sprintf("%.1f MB", float64(p.MemoryRSS)/(1024*1024))
	}
	memRes := fmt.Sprintf("%.1f MB", float64(p.MemoryRSS)/(1024*1024))
	memPriv := fmt.Sprintf("%.1f MB", p.Memory.RSSMB)
	if p.Memory.RSSMB == 0 && p.MemoryRSS > 0 {
		memPriv = fmt.Sprintf("%.1f MB", float64(p.MemoryRSS)/(1024*1024))
	}

	// I/O stats
	ioRead := "N/A"
	if p.IO.ReadBytes > 0 || p.IO.ReadOps > 0 {
		ioRead = fmt.Sprintf("%.1f MB (%d ops)", float64(p.IO.ReadBytes)/(1024*1024), p.IO.ReadOps)
	}
	ioWrite := "N/A"
	if p.IO.WriteBytes > 0 || p.IO.WriteOps > 0 {
		ioWrite = fmt.Sprintf("%.1f MB (%d ops)", float64(p.IO.WriteBytes)/(1024*1024), p.IO.WriteOps)
	}

	// Handles & Threads
	handles := "N/A"
	if p.FDCount > 0 {
		if p.FDLimit > 0 {
			handles = fmt.Sprintf("%d / %d", p.FDCount, p.FDLimit)
		} else {
			handles = fmt.Sprintf("%d", p.FDCount)
		}
	}
	threads := "N/A"
	if p.ThreadCount > 0 {
		threads = fmt.Sprintf("%d", p.ThreadCount)
	}

	// Environment variables map
	envVars := make(map[string]string)
	for _, env := range p.Env {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		} else if len(parts) == 1 {
			envVars[parts[0]] = ""
		}
	}

	return &GUIResult{
		Result:           *res,
		WhyExplanation:   GenerateWhyExplanation(res),
		StartedFormatted: startedStr,
		WorkingDir:       workDir,
		SocketsList:      socketsList,
		CPUFormatted:     cpuStr,
		MemoryVirtual:    memVirt,
		MemoryResident:   memRes,
		MemoryPrivate:    memPriv,
		IORead:           ioRead,
		IOWrite:          ioWrite,
		HandlesCount:     handles,
		ThreadCount:      threads,
		EnvVars:          envVars,
	}
}

// ListRunningProcesses returns snapshot of active processes with precomputed IsSystem, CPU, Memory & HasSockets.
func (a *App) ListRunningProcesses() ([]GUIProcessItem, error) {
	procs, err := procpkg.ListProcesses()
	if err != nil || len(procs) == 0 {
		procs, _ = procpkg.ListProcessSnapshot()
	}

	openPorts, _ := procpkg.ListOpenPorts()
	listeningPIDs := make(map[int]bool)
	for _, op := range openPorts {
		if op.PID > 0 && (op.State == "LISTEN" || op.State == "LISTENING" || op.State == "OPEN") {
			listeningPIDs[op.PID] = true
		}
	}

	out := make([]GUIProcessItem, len(procs))
	for i, p := range procs {
		cpuStr := fmt.Sprintf("%.1f%%", p.CPUPercent)
		memMb := float64(p.MemoryRSS) / (1024 * 1024)
		memStr := fmt.Sprintf("%.1f MB", memMb)
		memPctStr := fmt.Sprintf("(%.1f%%)", p.MemoryPercent)

		startedStr := "N/A"
		if !p.StartedAt.IsZero() {
			startedStr = p.StartedAt.Format("15:04:05")
		}

		out[i] = GUIProcessItem{
			Process:       p,
			CPUFormatted:  cpuStr,
			MemFormatted:  memStr,
			MemPercentStr: memPctStr,
			StartedStr:    startedStr,
			IsSystem:      isSystemProcess(p),
			HasSockets:    listeningPIDs[p.PID],
		}
	}
	return out, nil
}

// GetSystemAnalytics returns live overview counts.
func (a *App) GetSystemAnalytics() (*SystemAnalytics, error) {
	procs, err := a.ListRunningProcesses()
	if err != nil {
		return nil, err
	}

	analytics := &SystemAnalytics{
		TotalProcesses: len(procs),
	}

	listeningPIDs := make(map[int]bool)
	for _, p := range procs {
		if p.IsSystem {
			analytics.SystemProcesses++
		} else {
			analytics.UserProcesses++
		}
		if p.HasSockets {
			listeningPIDs[p.PID] = true
		}
	}

	analytics.ListeningPorts = len(listeningPIDs)
	return analytics, nil
}

// Search queries witr causality engine for a given input query and target type.
// targetType can be "auto", "pid", "port", "file", "container", "name".
func (a *App) Search(query string, targetType string, exact bool) (*GUIResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return nil, fmt.Errorf("search query cannot be empty")
	}

	var t model.Target

	switch strings.ToLower(targetType) {
	case "pid":
		t = model.Target{Type: model.TargetPID, Value: query}
	case "port":
		t = model.Target{Type: model.TargetPort, Value: query}
	case "file":
		t = model.Target{Type: model.TargetFile, Value: query}
	case "container":
		t = model.Target{Type: model.TargetContainer, Value: query}
	case "name":
		t = model.Target{Type: model.TargetName, Value: query}
	default:
		// Auto-detect target type from query string format
		if _, err := strconv.Atoi(query); err == nil && len(query) <= 5 {
			// Could be PID or Port; check PID first
			if pids, err := target.Resolve(model.Target{Type: model.TargetPID, Value: query}, exact); err == nil && len(pids) > 0 {
				t = model.Target{Type: model.TargetPID, Value: query}
			} else {
				t = model.Target{Type: model.TargetPort, Value: query}
			}
		} else if strings.HasPrefix(query, ":") || strings.Contains(query, ".") || strings.Contains(query, "/") || strings.Contains(query, "\\") {
			if strings.HasPrefix(query, ":") {
				t = model.Target{Type: model.TargetPort, Value: strings.TrimPrefix(query, ":")}
			} else {
				t = model.Target{Type: model.TargetFile, Value: query}
			}
		} else {
			t = model.Target{Type: model.TargetName, Value: query}
		}
	}

	if t.Type == model.TargetContainer {
		matches := procpkg.ResolveContainer(t.Value, exact)
		if len(matches) > 0 {
			match := matches[0]
			procpkg.EnrichContainer(match)
			pid := procpkg.ResolveContainerHostPID(match.Runtime, match.ID)
			if pid > 0 {
				res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
					PID:     pid,
					Verbose: true,
					Tree:    true,
					Target:  t,
				})
				if err == nil {
					return EnrichGUIResult(&res), nil
				}
			}
		}
	}

	pids, err := target.Resolve(t, exact)
	if err != nil {
		return nil, err
	}
	if len(pids) == 0 {
		return nil, fmt.Errorf("no matching process found for %q", query)
	}

	pid := pids[0]

	var systemdService string
	if t.Type == model.TargetPort && pid == 1 && source.IsSystemdRunning() {
		if portNum, err := strconv.Atoi(t.Value); err == nil {
			if svc, err := procpkg.ResolveSystemdService(portNum); err == nil && svc != "" {
				systemdService = svc
			}
		}
	}

	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID:     pid,
		Verbose: true,
		Tree:    true,
		Target:  t,
	})
	if err != nil {
		return nil, err
	}

	if systemdService != "" {
		res.ResolvedTarget = strings.TrimSuffix(systemdService, ".service")
	}

	if t.Type == model.TargetPort {
		if portNum, err := strconv.Atoi(t.Value); err == nil && portNum > 0 {
			res.SocketInfo = procpkg.GetSocketStateForPort(portNum)
			source.EnrichSocketInfo(res.SocketInfo)
		}
	}

	return EnrichGUIResult(&res), nil
}

// GetProcessDetails fetches deep info for a specific PID.
func (a *App) GetProcessDetails(pid int) (*GUIResult, error) {
	res, err := pipeline.AnalyzePID(pipeline.AnalyzeConfig{
		PID:     pid,
		Verbose: true,
		Tree:    true,
		Target:  model.Target{Type: model.TargetPID, Value: strconv.Itoa(pid)},
	})
	if err != nil {
		return nil, err
	}
	return EnrichGUIResult(&res), nil
}

// MinimizeToTray hides the application window.
func (a *App) MinimizeToTray() {
	if a.ctx != nil {
		runtime.WindowHide(a.ctx)
	}
}

// QuitApp terminates application.
func (a *App) QuitApp() {
	if a.ctx != nil {
		runtime.Quit(a.ctx)
	}
}
