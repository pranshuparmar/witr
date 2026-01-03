package output

import (
	"fmt"
	"time"

	"github.com/pranshuparmar/witr/pkg/model"
)

var (
	colorReset     = "\033[0m"
	colorRed       = "\033[31m"
	colorGreen     = "\033[32m"
	colorBlue      = "\033[34m"
	colorCyan      = "\033[36m"
	colorMagenta   = "\033[35m"
	colorBold      = "\033[2m"
	colorDimYellow = "\033[2;33m"
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
func RenderWarnings(warnings []string, colorEnabled bool) {
	if len(warnings) == 0 {
		if colorEnabled {
			fmt.Printf("%sNo warnings.%s\n", colorGreen, colorReset)
		} else {
			fmt.Println("No warnings.")
		}
		return
	}
	if colorEnabled {
		fmt.Printf("%sWarnings%s:\n", colorRed, colorReset)
		for _, w := range warnings {
			fmt.Printf("  • %s\n", w)
		}
	} else {
		fmt.Println("Warnings:")
		for _, w := range warnings {
			fmt.Printf("  • %s\n", w)
		}
	}
}

func RenderStandard(r model.Result, colorEnabled bool, verbose bool) {
	// Target
	target := "unknown"
	if len(r.Ancestry) > 0 {
		target = r.Ancestry[len(r.Ancestry)-1].Command
	}
	if colorEnabled {
		fmt.Printf("%sTarget%s      : %s\n\n", colorBlue, colorReset, target)
	} else {
		fmt.Printf("Target      : %s\n\n", target)
	}

	// Process
	var proc = r.Ancestry[len(r.Ancestry)-1]
	if colorEnabled {
		fmt.Printf("%sProcess%s     : %s (%spid %d%s)", colorBlue, colorReset, proc.Command, colorBold, proc.PID, colorReset)
	} else {
		fmt.Printf("Process     : %s (pid %d)", proc.Command, proc.PID)
	}
	// Health status
	if proc.Health != "" && proc.Health != "healthy" {
		healthColor := colorRed
		if colorEnabled {
			fmt.Printf(" %s[%s]%s", healthColor, proc.Health, colorReset)
		} else {
			fmt.Printf(" [%s]", proc.Health)
		}
	}
	// Forked status: only display if forked
	if proc.Forked == "forked" {
		forkColor := colorDimYellow
		if colorEnabled {
			fmt.Printf(" %s{forked}%s", forkColor, colorReset)
		} else {
			fmt.Printf(" {forked}")
		}
	}
	fmt.Println("")
	if proc.User != "" && proc.User != "unknown" {
		if colorEnabled {
			fmt.Printf("%sUser%s        : %s\n", colorCyan, colorReset, proc.User)
		} else {
			fmt.Printf("User        : %s\n", proc.User)
		}
	}

	// Container
	if proc.Container != "" {
		if colorEnabled {
			fmt.Printf("%sContainer%s   : %s\n", colorBlue, colorReset, proc.Container)
		} else {
			fmt.Printf("Container   : %s\n", proc.Container)
		}
	}
	// Service
	if proc.Service != "" {
		if colorEnabled {
			fmt.Printf("%sService%s     : %s\n", colorBlue, colorReset, proc.Service)
		} else {
			fmt.Printf("Service     : %s\n", proc.Service)
		}
	}

	if proc.Cmdline != "" {
		if colorEnabled {
			fmt.Printf("%sCommand%s     : %s\n", colorGreen, colorReset, proc.Cmdline)
		} else {
			fmt.Printf("Command     : %s\n", proc.Cmdline)
		}
	} else {
		if colorEnabled {
			fmt.Printf("%sCommand%s     : %s\n", colorGreen, colorReset, proc.Command)
		} else {
			fmt.Printf("Command     : %s\n", proc.Command)
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
		fmt.Printf("%sStarted%s     : %s (%s)\n", colorMagenta, colorReset, rel, dtStr)
	} else {
		fmt.Printf("Started     : %s (%s)\n", rel, dtStr)
	}

	// Restart count
	if r.RestartCount > 0 {
		if colorEnabled {
			fmt.Printf("%sRestarts%s    : %d\n", colorDimYellow, colorReset, r.RestartCount)
		} else {
			fmt.Printf("Restarts    : %d\n", r.RestartCount)
		}
	}

	// Why It Exists (short chain)
	if colorEnabled {
		fmt.Printf("\n%sWhy It Exists%s :\n  ", colorMagenta, colorReset)
		for i, p := range r.Ancestry {
			name := p.Command
			if name == "" && p.Cmdline != "" {
				name = p.Cmdline
			}
			fmt.Printf("%s (%spid %d%s)", name, colorBold, p.PID, colorReset)
			if i < len(r.Ancestry)-1 {
				fmt.Printf(" %s\u2192%s ", colorMagenta, colorReset)
			}
		}
		fmt.Print("\n\n")
	} else {
		fmt.Printf("\nWhy It Exists :\n  ")
		for i, p := range r.Ancestry {
			name := p.Command
			if name == "" && p.Cmdline != "" {
				name = p.Cmdline
			}
			fmt.Printf("%s (pid %d)", name, p.PID)
			if i < len(r.Ancestry)-1 {
				fmt.Printf(" \u2192 ")
			}
		}
		fmt.Print("\n\n")
	}

	// Source
	sourceLabel := string(r.Source.Type)
	if colorEnabled {
		if r.Source.Name != "" && r.Source.Name != sourceLabel {
			fmt.Printf("%sSource%s      : %s (%s)\n", colorCyan, colorReset, r.Source.Name, sourceLabel)
		} else {
			fmt.Printf("%sSource%s      : %s\n", colorCyan, colorReset, sourceLabel)
		}
	} else {
		if r.Source.Name != "" && r.Source.Name != sourceLabel {
			fmt.Printf("Source      : %s (%s)\n", r.Source.Name, sourceLabel)
		} else {
			fmt.Printf("Source      : %s\n", sourceLabel)
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
					fmt.Printf("%s%s%s : %s\n", colorBold, label, colorReset, val)
				} else {
					fmt.Printf("%s : %s\n", label, val)
				}
			}
		}
	}

	// Context group
	if colorEnabled {
		if proc.WorkingDir != "" {
			fmt.Printf("\n%sWorking Dir%s : %s\n", colorGreen, colorReset, proc.WorkingDir)
		}
		if proc.GitRepo != "" {
			if proc.GitBranch != "" {
				fmt.Printf("%sGit Repo%s    : %s (%s)\n", colorCyan, colorReset, proc.GitRepo, proc.GitBranch)
			} else {
				fmt.Printf("%sGit Repo%s    : %s\n", colorCyan, colorReset, proc.GitRepo)
			}
		}
	} else {
		if proc.WorkingDir != "" {
			fmt.Printf("\nWorking Dir : %s\n", proc.WorkingDir)
		}
		if proc.GitRepo != "" {
			if proc.GitBranch != "" {
				fmt.Printf("Git Repo    : %s (%s)\n", proc.GitRepo, proc.GitBranch)
			} else {
				fmt.Printf("Git Repo    : %s\n", proc.GitRepo)
			}
		}
	}

	// Listening section (address:port)
	if len(proc.ListeningPorts) > 0 && len(proc.BindAddresses) == len(proc.ListeningPorts) {
		for i := range proc.ListeningPorts {
			addr := proc.BindAddresses[i]
			port := proc.ListeningPorts[i]
			if addr != "" && port > 0 {
				if colorEnabled {
					if i == 0 {
						fmt.Printf("%sListening%s   : %s:%d\n", colorGreen, colorReset, addr, port)
					} else {
						fmt.Printf("              %s:%d\n", addr, port)
					}
				} else {
					if i == 0 {
						fmt.Printf("Listening   : %s:%d\n", addr, port)
					} else {
						fmt.Printf("              %s:%d\n", addr, port)
					}
				}
			}
		}
	}

	// Socket state (for port queries)
	if r.SocketInfo != nil {
		if colorEnabled {
			fmt.Printf("%sSocket%s      : %s\n", colorCyan, colorReset, r.SocketInfo.State)
			if r.SocketInfo.Explanation != "" {
				fmt.Printf("              %s\n", r.SocketInfo.Explanation)
			}
			if r.SocketInfo.Workaround != "" {
				fmt.Printf("              %s%s%s\n", colorDimYellow, r.SocketInfo.Workaround, colorReset)
			}
		} else {
			fmt.Printf("Socket      : %s\n", r.SocketInfo.State)
			if r.SocketInfo.Explanation != "" {
				fmt.Printf("              %s\n", r.SocketInfo.Explanation)
			}
			if r.SocketInfo.Workaround != "" {
				fmt.Printf("              %s\n", r.SocketInfo.Workaround)
			}
		}
	}

	// Resource context (thermal state, sleep prevention)
	if r.ResourceContext != nil {
		if r.ResourceContext.PreventsSleep {
			if colorEnabled {
				fmt.Printf("%sEnergy%s      : %sPreventing system sleep%s\n", colorRed, colorReset, colorDimYellow, colorReset)
			} else {
				fmt.Printf("Energy      : Preventing system sleep\n")
			}
		}
		if r.ResourceContext.ThermalState != "" {
			if colorEnabled {
				fmt.Printf("%sThermal%s     : %s%s%s\n", colorRed, colorReset, colorDimYellow, r.ResourceContext.ThermalState, colorReset)
			} else {
				fmt.Printf("Thermal     : %s\n", r.ResourceContext.ThermalState)
			}
		}
	}

	// File context (open files, locks)
	if r.FileContext != nil {
		if r.FileContext.OpenFiles > 0 && r.FileContext.FileLimit > 0 {
			usagePercent := float64(r.FileContext.OpenFiles) / float64(r.FileContext.FileLimit) * 100
			if colorEnabled {
				if usagePercent > 80 {
					fmt.Printf("%sOpen Files%s  : %s%d of %d (%.0f%%)%s\n", colorRed, colorReset, colorDimYellow, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent, colorReset)
				} else {
					fmt.Printf("%sOpen Files%s  : %d of %d (%.0f%%)\n", colorCyan, colorReset, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
				}
			} else {
				fmt.Printf("Open Files  : %d of %d (%.0f%%)\n", r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
			}
		}
		if len(r.FileContext.LockedFiles) > 0 {
			if colorEnabled {
				fmt.Printf("%sLocks%s       : %s\n", colorCyan, colorReset, r.FileContext.LockedFiles[0])
				for _, f := range r.FileContext.LockedFiles[1:] {
					fmt.Printf("              %s\n", f)
				}
			} else {
				fmt.Printf("Locks       : %s\n", r.FileContext.LockedFiles[0])
				for _, f := range r.FileContext.LockedFiles[1:] {
					fmt.Printf("              %s\n", f)
				}
			}
		}
	}

	// Warnings
	if len(r.Warnings) > 0 {
		if colorEnabled {
			fmt.Printf("\n%sWarnings%s    :\n", colorRed, colorReset)
			for _, w := range r.Warnings {
				fmt.Printf("  • %s\n", w)
			}
		} else {
			fmt.Println("\nWarnings    :")
			for _, w := range r.Warnings {
				fmt.Printf("  • %s\n", w)
			}
		}
	}

	// Extended information for verbose mode
	if verbose {
		if colorEnabled {
			fmt.Printf("\n%sExtended Information%s:\n", colorMagenta, colorReset)
		} else {
			fmt.Println("\nExtended Information:")
		}

		// Memory information
		if proc.Memory.VMS > 0 {
			if colorEnabled {
				fmt.Printf("\n%sMemory%s:\n", colorGreen, colorReset)
				fmt.Printf("  Virtual: %.1f MB\n", proc.Memory.VMSMB)
				fmt.Printf("  Resident: %.1f MB\n", proc.Memory.RSSMB)
				if proc.Memory.Shared > 0 {
					fmt.Printf("  Shared: %.1f MB\n", float64(proc.Memory.Shared)/(1024*1024))
				}
			} else {
				fmt.Printf("\nMemory:\n")
				fmt.Printf("  Virtual: %.1f MB\n", proc.Memory.VMSMB)
				fmt.Printf("  Resident: %.1f MB\n", proc.Memory.RSSMB)
				if proc.Memory.Shared > 0 {
					fmt.Printf("  Shared: %.1f MB\n", float64(proc.Memory.Shared)/(1024*1024))
				}
			}
		}

		// I/O statistics
		if proc.IO.ReadBytes > 0 || proc.IO.WriteBytes > 0 {
			if colorEnabled {
				fmt.Printf("\n%sI/O Statistics%s:\n", colorGreen, colorReset)
				if proc.IO.ReadBytes > 0 {
					fmt.Printf("  Read: %.1f MB (%d ops)\n", float64(proc.IO.ReadBytes)/(1024*1024), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					fmt.Printf("  Write: %.1f MB (%d ops)\n", float64(proc.IO.WriteBytes)/(1024*1024), proc.IO.WriteOps)
				}
			} else {
				fmt.Printf("\nI/O Statistics:\n")
				if proc.IO.ReadBytes > 0 {
					fmt.Printf("  Read: %.1f MB (%d ops)\n", float64(proc.IO.ReadBytes)/(1024*1024), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					fmt.Printf("  Write: %.1f MB (%d ops)\n", float64(proc.IO.WriteBytes)/(1024*1024), proc.IO.WriteOps)
				}
			}
		}

		// File descriptors
		if proc.FDCount > 0 {
			if colorEnabled {
				if proc.FDLimit == 0 {
					fmt.Printf("\n%sFile Descriptors%s: %d/unlimited\n", colorGreen, colorReset, proc.FDCount)
				} else {
					fmt.Printf("\n%sFile Descriptors%s: %d/%d\n", colorGreen, colorReset, proc.FDCount, proc.FDLimit)
				}
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= 10 {
					for _, fd := range proc.FileDescs {
						fmt.Printf("  %s\n", fd)
					}
				} else if len(proc.FileDescs) > 10 {
					fmt.Printf("  Showing first 10 of %d descriptors:\n", len(proc.FileDescs))
					for i := 0; i < 10; i++ {
						fmt.Printf("  %s\n", proc.FileDescs[i])
					}
					fmt.Printf("  ... and %d more\n", len(proc.FileDescs)-10)
				}
			} else {
				if proc.FDLimit == 0 {
					fmt.Printf("\nFile Descriptors: %d/unlimited\n", proc.FDCount)
				} else {
					fmt.Printf("\nFile Descriptors: %d/%d\n", proc.FDCount, proc.FDLimit)
				}
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= 10 {
					for _, fd := range proc.FileDescs {
						fmt.Printf("  %s\n", fd)
					}
				} else if len(proc.FileDescs) > 10 {
					fmt.Printf("  Showing first 10 of %d descriptors:\n", len(proc.FileDescs))
					for i := 0; i < 10; i++ {
						fmt.Printf("  %s\n", proc.FileDescs[i])
					}
					fmt.Printf("  ... and %d more\n", len(proc.FileDescs)-10)
				}
			}
		}

		// Children and threads
		if proc.ThreadCount > 1 || len(proc.Children) > 0 {
			if colorEnabled {
				fmt.Printf("\n%sProcess Details%s:\n", colorGreen, colorReset)
				if proc.ThreadCount > 1 {
					fmt.Printf("  Threads: %d\n", proc.ThreadCount)
				}
				if len(proc.Children) > 0 {
					fmt.Printf("  Children: %v\n", proc.Children)
				}
			} else {
				fmt.Printf("\nProcess Details:\n")
				if proc.ThreadCount > 1 {
					fmt.Printf("  Threads: %d\n", proc.ThreadCount)
				}
				if len(proc.Children) > 0 {
					fmt.Printf("  Children: %v\n", proc.Children)
				}
			}
		}
	}
}
