//go:build linux || darwin

package batch

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	procpkg "github.com/SanCognition/witr/internal/proc"
	"github.com/SanCognition/witr/internal/source"
	"github.com/SanCognition/witr/pkg/model"
)

// AnalyzeAsync runs full process analysis concurrently.
// Results stream to the returned channel as they complete.
func AnalyzeAsync(pids []int, concurrency int) <-chan ProcessSummary {
	results := make(chan ProcessSummary)
	semaphore := make(chan struct{}, concurrency) // Limit concurrent workers
	var wg sync.WaitGroup

	for _, pid := range pids {
		wg.Add(1)
		go func(pid int) {
			defer wg.Done()
			semaphore <- struct{}{}        // Acquire
			defer func() { <-semaphore }() // Release

			summary := analyzeProcess(pid)
			results <- summary
		}(pid)
	}

	go func() {
		wg.Wait()
		close(results)
	}()

	return results
}

// analyzeProcess does full analysis for a single PID
func analyzeProcess(pid int) ProcessSummary {
	summary := ProcessSummary{PID: pid}

	// 1. Get full ancestry (reuse existing)
	ancestry, err := procpkg.ResolveAncestry(pid)
	if err != nil {
		summary.Error = err
		return summary
	}

	// 2. Extract target process (last in chain)
	if len(ancestry) > 0 {
		proc := ancestry[len(ancestry)-1]
		summary.Command = proc.Command
		summary.Cmdline = proc.Cmdline
		summary.User = proc.User
		summary.StartedAt = proc.StartedAt
		summary.Age = formatAge(proc.StartedAt)
		summary.WorkDir = strings.TrimSpace(proc.WorkingDir) // Clean newlines
		summary.GitRepo = extractRepoName(proc.GitRepo)
		summary.Health = proc.Health

		// Detect npm script
		summary.Script = DetectNpmScript(proc.Cmdline, proc.WorkingDir)
	}

	// 3. Detect source (reuse existing)
	src := source.Detect(ancestry)
	summary.Source = formatSource(src, ancestry)

	// 4. Get CPU/memory
	summary.CPU, summary.MemoryMB = getCPUMemory(pid)

	return summary
}

// getCPUMemory retrieves CPU percentage and memory usage for a process
func getCPUMemory(pid int) (float64, int) {
	out, err := exec.Command("ps", "-p", strconv.Itoa(pid), "-o", "pcpu=,rss=").Output()
	if err != nil {
		return 0, 0
	}

	fields := strings.Fields(strings.TrimSpace(string(out)))
	if len(fields) < 2 {
		return 0, 0
	}

	cpu, _ := strconv.ParseFloat(fields[0], 64)
	rssKB, _ := strconv.ParseFloat(fields[1], 64)
	memMB := int(rssKB / 1024)

	return cpu, memMB
}

// formatAge converts a start time to a human-readable age string
func formatAge(startedAt time.Time) string {
	if startedAt.IsZero() {
		return "-"
	}

	dur := time.Since(startedAt)
	days := int(dur.Hours() / 24)
	hours := int(dur.Hours()) % 24
	mins := int(dur.Minutes()) % 60

	switch {
	case days > 0:
		return strconv.Itoa(days) + "d"
	case hours > 0:
		return strconv.Itoa(hours) + "h " + strconv.Itoa(mins) + "m"
	case mins > 0:
		return strconv.Itoa(mins) + "m"
	default:
		return "<1m"
	}
}

// formatSource simplifies source type for table display
func formatSource(src model.Source, ancestry []model.Process) string {
	switch src.Type {
	case model.SourceLaunchd:
		return "launchd"
	case model.SourceSystemd:
		return "systemd"
	case model.SourceContainer:
		return "container"
	case model.SourceSupervisor:
		// Check for specific supervisors
		if src.Name != "" {
			return src.Name
		}
		return "supervisor"
	case model.SourceCron:
		return "cron"
	case model.SourceShell:
		// Check if started by npm/yarn/pnpm or vscode
		for _, p := range ancestry {
			cmd := strings.ToLower(p.Command)
			if strings.Contains(cmd, "npm") {
				return "npm"
			}
			if strings.Contains(cmd, "yarn") {
				return "yarn"
			}
			if strings.Contains(cmd, "pnpm") {
				return "pnpm"
			}
			if strings.Contains(cmd, "code") || strings.Contains(p.Cmdline, "Visual Studio Code") {
				return "vscode"
			}
			if strings.Contains(cmd, "cursor") {
				return "cursor"
			}
			if strings.Contains(cmd, "idea") || strings.Contains(cmd, "webstorm") {
				return "jetbrains"
			}
		}
		return "shell"
	default:
		// Check ancestry for IDE/package manager patterns even if source is unknown
		for _, p := range ancestry {
			cmd := strings.ToLower(p.Command)
			if strings.Contains(cmd, "npm") {
				return "npm"
			}
			if strings.Contains(cmd, "yarn") {
				return "yarn"
			}
			if strings.Contains(cmd, "pnpm") {
				return "pnpm"
			}
			if strings.Contains(cmd, "code") || strings.Contains(p.Cmdline, "Visual Studio Code") {
				return "vscode"
			}
		}
		return "-"
	}
}

// extractRepoName extracts just the repo folder name from a full path
func extractRepoName(gitRepo string) string {
	if gitRepo == "" {
		return "-"
	}
	return filepath.Base(gitRepo)
}

// ShortenPath replaces home directory with ~ for display
func ShortenPath(path string) string {
	if path == "" {
		return "-"
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return path
	}
	if strings.HasPrefix(path, home) {
		return "~" + path[len(home):]
	}
	return path
}

// Truncate shortens a string to maxLen characters
func Truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
