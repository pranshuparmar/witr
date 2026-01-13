package output

import (
	"fmt"
	"io"
	"net"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

// formatDetailLabel formats a detail key into a padded label for display
func formatDetailLabel(key string) string {
	labels := map[string]string{
		"type":      "              Type",
		"plist":     "              Plist",
		"triggers":  "              Trigger",
		"keepalive": "              KeepAlive",
	}
	if label, ok := labels[key]; ok {
		return label
	}
	return "              " + key
}

// RenderWarnings prints only the warnings, with color if enabled
func RenderWarnings(w io.Writer, warnings []string, colorEnabled bool) {
	out := NewPrinter(w)
	if len(warnings) == 0 {
		if colorEnabled {
			out.Printf("%sNo warnings.%s\n", ColorGreen, ColorReset)
		} else {
			out.Println("No warnings.")
		}
		return
	}
	if colorEnabled {
		out.Printf("%sWarnings%s:\n", ColorRed, ColorReset)
		for _, w := range warnings {
			out.Printf("  • %s\n", SanitizeTerminal(w))
		}
	} else {
		out.Println("Warnings:")
		for _, w := range warnings {
			out.Printf("  • %s\n", SanitizeTerminal(w))
		}
	}
}

func RenderStandard(w io.Writer, r model.Result, colorEnabled bool, verbose bool) {
	out := NewPrinter(w)
	// Target
	target := "unknown"
	if len(r.Ancestry) > 0 {
		target = SanitizeTerminal(r.Ancestry[len(r.Ancestry)-1].Command)
	}
	if colorEnabled {
		out.Printf("%sTarget%s      : %s\n\n", ColorBlue, ColorReset, target)
	} else {
		out.Printf("Target      : %s\n\n", target)
	}

	// Process
	var proc = r.Ancestry[len(r.Ancestry)-1]
	proc.Command = SanitizeTerminal(proc.Command)
	proc.Cmdline = SanitizeTerminal(proc.Cmdline)
	proc.User = SanitizeTerminal(proc.User)
	proc.Container = SanitizeTerminal(proc.Container)
	proc.Service = SanitizeTerminal(proc.Service)
	proc.WorkingDir = SanitizeTerminal(proc.WorkingDir)
	proc.GitRepo = SanitizeTerminal(proc.GitRepo)
	proc.GitBranch = SanitizeTerminal(proc.GitBranch)
	if colorEnabled {
		out.Printf("%sProcess%s     : %s%s%s (%spid %d%s)", ColorBlue, ColorReset, ColorGreen, proc.Command, ColorReset, ColorBold, proc.PID, ColorReset)
	} else {
		out.Printf("Process     : %s (pid %d)", proc.Command, proc.PID)
	}
	// Health status
	if proc.Health != "" && proc.Health != "healthy" {
		health := SanitizeTerminal(proc.Health)
		healthColor := ColorRed
		if colorEnabled {
			out.Printf(" %s[%s]%s", healthColor, health, ColorReset)
		} else {
			out.Printf(" [%s]", health)
		}
	}
	// Forked status: only display if forked
	if proc.Forked == "forked" {
		forkColor := ColorDimYellow
		if colorEnabled {
			out.Printf(" %s{forked}%s", forkColor, ColorReset)
		} else {
			out.Printf(" {forked}")
		}
	}
	out.Println("")
	if proc.User != "" && proc.User != "unknown" {
		if colorEnabled {
			out.Printf("%sUser%s        : %s\n", ColorCyan, ColorReset, proc.User)
		} else {
			out.Printf("User        : %s\n", proc.User)
		}
	}

	// Container
	if proc.Container != "" {
		if colorEnabled {
			out.Printf("%sContainer%s   : %s\n", ColorBlue, ColorReset, proc.Container)
		} else {
			out.Printf("Container   : %s\n", proc.Container)
		}
	}
	// Service
	if proc.Service != "" {
		if colorEnabled {
			out.Printf("%sService%s     : %s\n", ColorBlue, ColorReset, proc.Service)
		} else {
			out.Printf("Service     : %s\n", proc.Service)
		}
	}

	if proc.Cmdline != "" {
		if colorEnabled {
			out.Printf("%sCommand%s     : %s\n", ColorGreen, ColorReset, proc.Cmdline)
		} else {
			out.Printf("Command     : %s\n", proc.Cmdline)
		}
	} else {
		if colorEnabled {
			out.Printf("%sCommand%s     : %s\n", ColorGreen, ColorReset, proc.Command)
		} else {
			out.Printf("Command     : %s\n", proc.Command)
		}
	}
	// Format as: 2 days ago (Mon 2025-02-02 11:42:10 +0530)
	startedAt := proc.StartedAt
	now := time.Now()
	dur := now.Sub(startedAt)
	var rel string
	switch {
	case dur.Hours() >= 48:
		days := int(dur.Hours()) / 24
		rel = fmt.Sprintf("%d days ago", days)
	case dur.Hours() >= 24:
		rel = "1 day ago"
	case dur.Hours() >= 2:
		hours := int(dur.Hours())
		rel = fmt.Sprintf("%d hours ago", hours)
	case dur.Minutes() >= 60:
		rel = "1 hour ago"
	default:
		mins := int(dur.Minutes())
		if mins > 0 {
			rel = fmt.Sprintf("%d min ago", mins)
		} else {
			rel = "just now"
		}
	}
	dtStr := startedAt.Format("Mon 2006-01-02 15:04:05 -07:00")
	if colorEnabled {
		out.Printf("%sStarted%s     : %s (%s)\n", ColorMagenta, ColorReset, rel, dtStr)
	} else {
		out.Printf("Started     : %s (%s)\n", rel, dtStr)
	}

	// Restart count
	if r.RestartCount > 0 {
		if colorEnabled {
			out.Printf("%sRestarts%s    : %d\n", ColorDimYellow, ColorReset, r.RestartCount)
		} else {
			out.Printf("Restarts    : %d\n", r.RestartCount)
		}
	}

	// Why It Exists (short chain)
	if colorEnabled {
		out.Printf("\n%sWhy It Exists%s :\n  ", ColorMagenta, ColorReset)
		for i, p := range r.Ancestry {
			name := p.Command
			if name == "" && p.Cmdline != "" {
				name = p.Cmdline
			}
			name = SanitizeTerminal(name)

			nameColor := ansiString("")
			if i == len(r.Ancestry)-1 {
				nameColor = ColorGreen
			}
			out.Printf("%s%s%s (%spid %d%s)", nameColor, name, ColorReset, ColorBold, p.PID, ColorReset)
			if i < len(r.Ancestry)-1 {
				out.Printf(" %s\u2192%s ", ColorMagenta, ColorReset)
			}
		}
		out.Print("\n\n")
	} else {
		out.Printf("\nWhy It Exists :\n  ")
		for i, p := range r.Ancestry {
			name := p.Command
			if name == "" && p.Cmdline != "" {
				name = p.Cmdline
			}
			name = SanitizeTerminal(name)
			out.Printf("%s (pid %d)", name, p.PID)
			if i < len(r.Ancestry)-1 {
				out.Printf(" \u2192 ")
			}
		}
		out.Print("\n\n")
	}

	// Source
	sourceLabel := string(r.Source.Type)
	sourceName := SanitizeTerminal(r.Source.Name)
	if colorEnabled {
		if r.Source.Name != "" && r.Source.Name != sourceLabel {
			out.Printf("%sSource%s      : %s (%s)\n", ColorCyan, ColorReset, sourceName, sourceLabel)
		} else {
			out.Printf("%sSource%s      : %s\n", ColorCyan, ColorReset, sourceLabel)
		}
	} else {
		if r.Source.Name != "" && r.Source.Name != sourceLabel {
			out.Printf("Source      : %s (%s)\n", sourceName, sourceLabel)
		} else {
			out.Printf("Source      : %s\n", sourceLabel)
		}
	}

	// Source details (launchd triggers, plist path, etc.)
	if len(r.Source.Details) > 0 {
		// Display in consistent order
		detailKeys := []string{"type", "plist", "triggers", "keepalive"}
		for _, key := range detailKeys {
			if val, ok := r.Source.Details[key]; ok {
				label := formatDetailLabel(key)
				if colorEnabled {
					out.Printf("%s%s%s : %s\n", ColorBold, label, ColorReset, SanitizeTerminal(val))
				} else {
					out.Printf("%s : %s\n", label, SanitizeTerminal(val))
				}
			}
		}
	}

	// Context group
	if colorEnabled {
		if proc.WorkingDir != "" && proc.WorkingDir != "unknown" {
			out.Printf("\n%sWorking Dir%s : %s\n", ColorGreen, ColorReset, proc.WorkingDir)
		}
		if proc.GitRepo != "" {
			if proc.GitBranch != "" {
				out.Printf("%sGit Repo%s    : %s (%s)\n", ColorCyan, ColorReset, proc.GitRepo, proc.GitBranch)
			} else {
				out.Printf("%sGit Repo%s    : %s\n", ColorCyan, ColorReset, proc.GitRepo)
			}
		}
	} else {
		if proc.WorkingDir != "" && proc.WorkingDir != "unknown" {
			out.Printf("\nWorking Dir : %s\n", proc.WorkingDir)
		}
		if proc.GitRepo != "" {
			if proc.GitBranch != "" {
				out.Printf("Git Repo    : %s (%s)\n", proc.GitRepo, proc.GitBranch)
			} else {
				out.Printf("Git Repo    : %s\n", proc.GitRepo)
			}
		}
	}

	// Listening section (address:port)
	if len(proc.ListeningPorts) > 0 && len(proc.BindAddresses) == len(proc.ListeningPorts) {
		for i := range proc.ListeningPorts {
			addr := proc.BindAddresses[i]
			port := proc.ListeningPorts[i]
			if addr != "" && port > 0 {
				hostPort := net.JoinHostPort(addr, strconv.Itoa(port))
				safeHostPort := SanitizeTerminal(hostPort)
				if colorEnabled {
					if i == 0 {
						out.Printf("%sListening%s   : %s\n", ColorGreen, ColorReset, safeHostPort)
					} else {
						out.Printf("              %s\n", safeHostPort)
					}
				} else {
					if i == 0 {
						out.Printf("Listening   : %s\n", safeHostPort)
					} else {
						out.Printf("              %s\n", safeHostPort)
					}
				}

			}
		}
	}

	// Warnings
	if len(r.Warnings) > 0 {
		if colorEnabled {
			out.Printf("\n%sWarnings%s    :\n", ColorRed, ColorReset)
			for _, w := range r.Warnings {
				out.Printf("  • %s\n", SanitizeTerminal(w))
			}
		} else {
			out.Println("\nWarnings    :")
			for _, w := range r.Warnings {
				out.Printf("  • %s\n", SanitizeTerminal(w))
			}
		}
	}

	// Extended information for verbose mode
	if verbose {
		out.Println()
		// Resource context (thermal state, sleep prevention, CPU)
		if r.ResourceContext != nil {
			if r.ResourceContext.CPUUsage > 0 {
				if colorEnabled {
					if r.ResourceContext.CPUUsage > 70 {
						out.Printf("%sCPU%s         : %s%.1f%%%s\n", ColorRed, ColorReset, ColorDimYellow, r.ResourceContext.CPUUsage, ColorReset)
					} else {
						out.Printf("%sCPU%s         : %.1f%%\n", ColorGreen, ColorReset, r.ResourceContext.CPUUsage)
					}
				} else {
					out.Printf("CPU         : %.1f%%\n", r.ResourceContext.CPUUsage)
				}
			}

			if r.ResourceContext.PreventsSleep {
				if colorEnabled {
					out.Printf("%sEnergy%s      : %sPreventing system sleep%s\n", ColorRed, ColorReset, ColorDimYellow, ColorReset)
				} else {
					out.Printf("Energy      : Preventing system sleep\n")
				}
			}

			if r.ResourceContext.ThermalState != "" {
				thermalState := SanitizeTerminal(r.ResourceContext.ThermalState)
				if colorEnabled {
					out.Printf("%sThermal%s     : %s%s%s\n", ColorRed, ColorReset, ColorDimYellow, thermalState, ColorReset)
				} else {
					out.Printf("Thermal     : %s\n", thermalState)
				}
			}
		}

		// Memory information
		if proc.Memory.VMS > 0 {
			if colorEnabled {
				out.Printf("\n%sMemory%s:\n", ColorGreen, ColorReset)
				out.Printf("  Virtual  : %.1f MB\n", proc.Memory.VMSMB)
				out.Printf("  Resident : %.1f MB\n", proc.Memory.RSSMB)
				if r.ResourceContext != nil && r.ResourceContext.MemoryUsage > 0 {
					out.Printf("  Private  : %.1f MB\n", float64(r.ResourceContext.MemoryUsage)/(1024*1024))
				}
				if proc.Memory.Shared > 0 {
					out.Printf("  Shared   : %.1f MB\n", float64(proc.Memory.Shared)/(1024*1024))
				}
			} else {
				out.Printf("\nMemory:\n")
				out.Printf("  Virtual  : %.1f MB\n", proc.Memory.VMSMB)
				out.Printf("  Resident : %.1f MB\n", proc.Memory.RSSMB)
				if r.ResourceContext != nil && r.ResourceContext.MemoryUsage > 0 {
					out.Printf("  Private  : %.1f MB\n", float64(r.ResourceContext.MemoryUsage)/(1024*1024))
				}
				if proc.Memory.Shared > 0 {
					out.Printf("  Shared   : %.1f MB\n", float64(proc.Memory.Shared)/(1024*1024))
				}
			}
		}

		// I/O statistics
		if proc.IO.ReadBytes > 0 || proc.IO.WriteBytes > 0 {
			if colorEnabled {
				out.Printf("\n%sI/O Statistics%s:\n", ColorGreen, ColorReset)
				if proc.IO.ReadBytes > 0 {
					out.Printf("  Read  : %.1f MB (%d ops)\n", float64(proc.IO.ReadBytes)/(1024*1024), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					out.Printf("  Write : %.1f MB (%d ops)\n", float64(proc.IO.WriteBytes)/(1024*1024), proc.IO.WriteOps)
				}
			} else {
				out.Printf("\nI/O Statistics:\n")
				if proc.IO.ReadBytes > 0 {
					out.Printf("  Read  : %.1f MB (%d ops)\n", float64(proc.IO.ReadBytes)/(1024*1024), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					out.Printf("  Write : %.1f MB (%d ops)\n", float64(proc.IO.WriteBytes)/(1024*1024), proc.IO.WriteOps)
				}
			}
		}

		// File context (open files, locks)
		if r.FileContext != nil {
			if r.FileContext.OpenFiles > 0 && r.FileContext.FileLimit > 0 {
				usagePercent := float64(r.FileContext.OpenFiles) / float64(r.FileContext.FileLimit) * 100
				if colorEnabled {
					if usagePercent > 80 {
						out.Printf("%sOpen Files%s  : %s%d of %d (%.0f%%)%s\n", ColorRed, ColorReset, ColorDimYellow, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent, ColorReset)
					} else {
						out.Printf("%sOpen Files%s  : %d of %d (%.0f%%)\n", ColorGreen, ColorReset, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
					}
				} else {
					out.Printf("Open Files  : %d of %d (%.0f%%)\n", r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
				}
			}
			if len(r.FileContext.LockedFiles) > 0 {
				firstLocked := SanitizeTerminal(r.FileContext.LockedFiles[0])
				if colorEnabled {
					out.Printf("%sLocks%s       : %s\n", ColorCyan, ColorReset, firstLocked)
					for _, f := range r.FileContext.LockedFiles[1:] {
						out.Printf("              %s\n", SanitizeTerminal(f))
					}
				} else {
					out.Printf("Locks       : %s\n", firstLocked)
					for _, f := range r.FileContext.LockedFiles[1:] {
						out.Printf("              %s\n", SanitizeTerminal(f))
					}
				}
			}
		}

		// File descriptors
		if proc.FDCount > 0 {
			// Sort file descriptors numerically
			sort.Slice(proc.FileDescs, func(i, j int) bool {
				fdI := proc.FileDescs[i]
				fdJ := proc.FileDescs[j]
				idxI := strings.Index(fdI, " ")
				idxJ := strings.Index(fdJ, " ")

				if idxI == -1 || idxJ == -1 {
					return fdI < fdJ
				}

				numI, errI := strconv.Atoi(fdI[:idxI])
				numJ, errJ := strconv.Atoi(fdJ[:idxJ])

				if errI == nil && errJ == nil {
					return numI < numJ
				}
				return fdI < fdJ
			})

			if colorEnabled {
				if proc.FDLimit == 0 {
					out.Printf("\n%sFile Descriptors%s: %d/unlimited\n", ColorGreen, ColorReset, proc.FDCount)
				} else {
					out.Printf("\n%sFile Descriptors%s: %d/%d\n", ColorGreen, ColorReset, proc.FDCount, proc.FDLimit)
				}
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= 10 {
					for _, fd := range proc.FileDescs {
						safeFd := SanitizeTerminal(fd)
						safeFd = strings.Replace(safeFd, "->", string(ColorMagenta)+"->"+string(ColorReset), 1)
						out.Printf("  %s\n", ansiString(safeFd))
					}
				} else if len(proc.FileDescs) > 10 {
					out.Printf("  Showing first 10 of %d descriptors:\n", len(proc.FileDescs))
					for i := 0; i < 10; i++ {
						safeFd := SanitizeTerminal(proc.FileDescs[i])
						safeFd = strings.Replace(safeFd, "->", string(ColorMagenta)+"->"+string(ColorReset), 1)
						out.Printf("  %s\n", ansiString(safeFd))
					}
					out.Printf("  ... and %d more\n", len(proc.FileDescs)-10)
				}
			} else {
				if proc.FDLimit == 0 {
					out.Printf("\nFile Descriptors: %d/unlimited\n", proc.FDCount)
				} else {
					out.Printf("\nFile Descriptors: %d/%d\n", proc.FDCount, proc.FDLimit)
				}
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= 10 {
					for _, fd := range proc.FileDescs {
						out.Printf("  %s\n", SanitizeTerminal(fd))
					}
				} else if len(proc.FileDescs) > 10 {
					out.Printf("  Showing first 10 of %d descriptors:\n", len(proc.FileDescs))
					for i := 0; i < 10; i++ {
						out.Printf("  %s\n", SanitizeTerminal(proc.FileDescs[i]))
					}
					out.Printf("  ... and %d more\n", len(proc.FileDescs)-10)
				}
			}
		}

		// Socket state (for port queries)
		if r.SocketInfo != nil {
			state := SanitizeTerminal(r.SocketInfo.State)
			explanation := SanitizeTerminal(r.SocketInfo.Explanation)
			workaround := SanitizeTerminal(r.SocketInfo.Workaround)
			if colorEnabled {
				out.Printf("%sSocket%s      : %s\n", ColorCyan, ColorReset, state)
				if explanation != "" {
					out.Printf("              %s\n", explanation)
				}
				if workaround != "" {
					out.Printf("              %s%s%s\n", ColorDimYellow, workaround, ColorReset)
				}
			} else {
				out.Printf("Socket      : %s\n", state)
				if explanation != "" {
					out.Printf("              %s\n", explanation)
				}
				if workaround != "" {
					out.Printf("              %s\n", workaround)
				}
			}
		}

		// Threads
		if proc.ThreadCount > 1 {
			if colorEnabled {
				out.Printf("\n%sThreads%s: %d\n", ColorGreen, ColorReset, proc.ThreadCount)
			} else {
				out.Printf("\nThreads: %d\n", proc.ThreadCount)
			}
		}

		// Child processes
		if len(r.ChildProcesses) > 0 {
			out.Println("")
			PrintChildren(w, r.Process, r.ChildProcesses, colorEnabled)
		}
	}
}
