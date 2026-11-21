package output

import (
	"fmt"
	"io"
	"net"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// Maximum number of items to display in any list before truncating
const MaxDisplayItems = 10

var detailLabels = map[string]string{
	"type":      "              Type",
	"plist":     "              Plist",
	"triggers":  "              Trigger",
	"keepalive": "              KeepAlive",
}

func formatDetailLabel(key string) string {
	if label, ok := detailLabels[key]; ok {
		return label
	}
	return "              " + key
}

func RenderWarnings(w io.Writer, r model.Result, colorEnabled bool) {
	out := NewPrinter(w)

	proc := r.Process
	if len(r.Ancestry) > 0 {
		proc = r.Ancestry[len(r.Ancestry)-1]
	}

	proc.Command = SanitizeTerminal(proc.Command)

	if colorEnabled {
		out.Printf("%sProcess%s     : %s%s%s (%spid %d%s)\n", ColorBlue, ColorReset, ColorGreen, proc.Command, ColorReset, ColorDim, proc.PID, ColorReset)
		if proc.Cmdline != "" {
			out.Printf("%sCommand%s     : %s\n", ColorBlue, ColorReset, proc.Cmdline)
		} else {
			out.Printf("%sCommand%s     : %s\n", ColorBlue, ColorReset, proc.Command)
		}
	} else {
		out.Printf("Process     : %s (pid %d)\n", proc.Command, proc.PID)
		if proc.Cmdline != "" {
			out.Printf("Command     : %s\n", proc.Cmdline)
		} else {
			out.Printf("Command     : %s\n", proc.Command)
		}
	}

	if len(r.Warnings) == 0 {
		if colorEnabled {
			out.Printf("%sWarnings%s    : %sNo warnings.%s\n", ColorRed, ColorReset, ColorGreen, ColorReset)
		} else {
			out.Println("Warnings    : No warnings.")
		}
		return
	}

	if colorEnabled {
		out.Printf("%sWarnings%s    :\n", ColorRed, ColorReset)
		for _, w := range r.Warnings {
			out.Printf("  • %s\n", SanitizeTerminal(w))
		}
	} else {
		out.Println("Warnings    :")
		for _, w := range r.Warnings {
			out.Printf("  • %s\n", SanitizeTerminal(w))
		}
	}
}

func RenderStandard(w io.Writer, r model.Result, colorEnabled bool, verbose bool) {
	out := NewPrinter(w)
	if len(r.Ancestry) == 0 {
		out.Println("No process information available.")
		return
	}

	target := SanitizeTerminal(r.Ancestry[len(r.Ancestry)-1].Command)
	if colorEnabled {
		out.Printf("%sTarget%s      : %s\n\n", ColorBlue, ColorReset, target)
	} else {
		out.Printf("Target      : %s\n\n", target)
	}

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
		out.Printf("%sProcess%s     : %s%s%s (%spid %d%s)", ColorBlue, ColorReset, ColorGreen, proc.Command, ColorReset, ColorDim, proc.PID, ColorReset)
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
		user := proc.User + integrityNote(proc.IntegrityLevel)
		if colorEnabled {
			out.Printf("%sUser%s        : %s\n", ColorBlue, ColorReset, user)
		} else {
			out.Printf("User        : %s\n", user)
		}
	}
	if label := securityLabel(proc); label != "" {
		if colorEnabled {
			out.Printf("%sSecurity%s    : %s\n", ColorBlue, ColorReset, label)
		} else {
			out.Printf("Security    : %s\n", label)
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
	if c := r.Container; c != nil {
		if image := SanitizeTerminalLine(c.Image); image != "" {
			if colorEnabled {
				out.Printf("%sImage%s       : %s\n", ColorBlue, ColorReset, image)
			} else {
				out.Printf("Image       : %s\n", image)
			}
		}
		if ports := SanitizeTerminalLine(c.Ports); ports != "" {
			if colorEnabled {
				out.Printf("%sPublished%s   : %s\n", ColorBlue, ColorReset, ports)
			} else {
				out.Printf("Published   : %s\n", ports)
			}
		}
		printComposeOrigin(out, c, colorEnabled)
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
			out.Printf("%sCommand%s     : %s\n", ColorBlue, ColorReset, proc.Cmdline)
		} else {
			out.Printf("Command     : %s\n", proc.Cmdline)
		}
	} else {
		if colorEnabled {
			out.Printf("%sCommand%s     : %s\n", ColorBlue, ColorReset, proc.Command)
		} else {
			out.Printf("Command     : %s\n", proc.Command)
		}
	}
	started, dtStr := FormatStartedAt(proc.StartedAt)
	if dtStr != "" {
		started += " (" + dtStr + ")"
	}
	if colorEnabled {
		out.Printf("%sStarted%s     : %s\n", ColorMagenta, ColorReset, started)
	} else {
		out.Printf("Started     : %s\n", started)
	}

	// Restarts by the managing system (systemd's NRestarts, or the container
	// runtime's count), with the container's restart policy.
	policy := ""
	if r.Container != nil {
		policy = r.Container.RestartPolicy
	}
	if v := restartsValue(r.RestartCount, policy); v != "" {
		if colorEnabled {
			out.Printf("%sRestarts%s    : %s\n", ColorMagenta, ColorReset, v)
		} else {
			out.Printf("Restarts    : %s\n", v)
		}
	}

	if schedule, ok := r.Source.Details["schedule"]; ok {
		if colorEnabled {
			out.Printf("%sSchedule%s    : %s\n", ColorMagenta, ColorReset, schedule)
		} else {
			out.Printf("Schedule    : %s\n", schedule)
		}
	}

	// Why It Exists (short chain)
	if colorEnabled {
		out.Printf("\n%sWhy It Exists%s :\n  ", ColorMagenta, ColorReset)
		for i, p := range r.Ancestry {
			name := SanitizeTerminal(ChainName(p))
			if gap := ParentGap(r.Ancestry, i); gap != "" {
				out.Printf("%s%s%s %s→%s ", ColorDimYellow, gap, ColorReset, ColorMagenta, ColorReset)
			}

			nameColor := ansiString("")
			if i == len(r.Ancestry)-1 {
				nameColor = ColorGreen
			}
			out.Printf("%s%s%s (%spid %d%s)", nameColor, name, ColorReset, ColorDim, p.PID, ColorReset)
			if i < len(r.Ancestry)-1 {
				out.Printf(" %s\u2192%s ", ColorMagenta, ColorReset)
			}
		}
		out.Print(ansiString("\n\n"))
	} else {
		out.Printf("\nWhy It Exists :\n  ")
		for i, p := range r.Ancestry {
			name := SanitizeTerminal(ChainName(p))
			if gap := ParentGap(r.Ancestry, i); gap != "" {
				out.Printf("%s → ", gap)
			}
			out.Printf("%s (pid %d)", name, p.PID)
			if i < len(r.Ancestry)-1 {
				out.Printf(" \u2192 ")
			}
		}
		out.Print(ansiString("\n\n"))
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

	// Description
	if r.Source.Description != "" {
		if colorEnabled {
			out.Printf("%sDescription%s : %s\n", ColorCyan, ColorReset, SanitizeTerminal(r.Source.Description))
		} else {
			out.Printf("Description : %s\n", SanitizeTerminal(r.Source.Description))
		}
	}

	// Unit File / Config Source
	if r.Source.UnitFile != "" {
		label := "Unit File"
		switch r.Source.Type {
		case model.SourceLaunchd:
			label = "Plist File"
		case model.SourceWindowsService:
			label = "Registry Key"
		case model.SourceBsdRc:
			label = "Rc Script"
		}

		var pad string
		if len(label) < 12 {
			pad = strings.Repeat(" ", 12-len(label))
		} else {
			pad = " "
		}

		unitFile := SanitizeTerminal(r.Source.UnitFile)
		if colorEnabled {
			out.Printf("%s%s%s%s: %s\n", ColorCyan, label, ColorReset, pad, unitFile)
		} else {
			out.Printf("%s%s: %s\n", label, pad, unitFile)
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
					out.Printf("%s%s%s : %s\n", ColorDim, label, ColorReset, SanitizeTerminal(val))
				} else {
					out.Printf("%s : %s\n", label, SanitizeTerminal(val))
				}
			}
		}
	}

	// Context group
	if colorEnabled {
		if proc.WorkingDir != "" && proc.WorkingDir != "unknown" {
			out.Printf("\n%sWorking Dir%s : %s\n", ColorCyan, ColorReset, proc.WorkingDir)
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

	// Sockets section: listeners, with the connections they accepted folded
	// into a count, then other sockets with the remote end of a connection.
	if rows := socketRows(proc.Sockets); len(rows) > 0 {
		for i, r := range rows {
			if i >= MaxDisplayItems {
				out.Printf("              ... and %d more\n", len(rows)-i)
				break
			}
			line := SanitizeTerminal(formatSocket(r))
			switch {
			case i == 0 && colorEnabled:
				out.Printf("%sSockets%s     : %s\n", ColorGreen, ColorReset, line)
			case i == 0:
				out.Printf("Sockets     : %s\n", line)
			default:
				out.Printf("              %s\n", line)
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
			out.Println(ansiString("\nWarnings    :"))
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
			if colorEnabled {
				if r.ResourceContext.CPUUsage > 70 {
					out.Printf("%sCPU (avg)%s   : %s%.1f%%%s\n", ColorRed, ColorReset, ColorDimYellow, r.ResourceContext.CPUUsage, ColorReset)
				} else {
					out.Printf("%sCPU (avg)%s   : %.1f%%\n", ColorGreen, ColorReset, r.ResourceContext.CPUUsage)
				}
			} else {
				out.Printf("CPU (avg)   : %.1f%%\n", r.ResourceContext.CPUUsage)
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
		if proc.Memory.VMS > 0 || proc.Memory.RSS > 0 {
			if colorEnabled {
				out.Printf("\n%sMemory%s:\n", ColorGreen, ColorReset)
				if proc.Memory.VMS > 0 {
					out.Printf("  Virtual  : %s\n", formatBytes(proc.Memory.VMS))
				}
				out.Printf("  Resident : %s\n", formatBytes(proc.Memory.RSS))
				if r.ResourceContext != nil && r.ResourceContext.MemoryUsage > 0 {
					out.Printf("  Private  : %s\n", formatBytes(r.ResourceContext.MemoryUsage))
				}
				if proc.Memory.Shared > 0 {
					out.Printf("  Shared   : %s\n", formatBytes(proc.Memory.Shared))
				}
			} else {
				out.Printf("\nMemory:\n")
				if proc.Memory.VMS > 0 {
					out.Printf("  Virtual  : %s\n", formatBytes(proc.Memory.VMS))
				}
				out.Printf("  Resident : %s\n", formatBytes(proc.Memory.RSS))
				if r.ResourceContext != nil && r.ResourceContext.MemoryUsage > 0 {
					out.Printf("  Private  : %s\n", formatBytes(r.ResourceContext.MemoryUsage))
				}
				if proc.Memory.Shared > 0 {
					out.Printf("  Shared   : %s\n", formatBytes(proc.Memory.Shared))
				}
			}
		}

		// I/O statistics
		if proc.IO.ReadBytes > 0 || proc.IO.WriteBytes > 0 {
			if colorEnabled {
				out.Printf("\n%sI/O Statistics%s:\n", ColorGreen, ColorReset)
				if proc.IO.ReadBytes > 0 {
					out.Printf("  Read  : %s (%d ops)\n", formatBytes(proc.IO.ReadBytes), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					out.Printf("  Write : %s (%d ops)\n", formatBytes(proc.IO.WriteBytes), proc.IO.WriteOps)
				}
			} else {
				out.Printf("\nI/O Statistics:\n")
				if proc.IO.ReadBytes > 0 {
					out.Printf("  Read  : %s (%d ops)\n", formatBytes(proc.IO.ReadBytes), proc.IO.ReadOps)
				}
				if proc.IO.WriteBytes > 0 {
					out.Printf("  Write : %s (%d ops)\n", formatBytes(proc.IO.WriteBytes), proc.IO.WriteOps)
				}
			}
		}

		// File context (open files, locks)
		if r.FileContext != nil {
			if r.FileContext.OpenFiles > 0 && r.FileContext.FileLimit == 0 {
				if colorEnabled {
					out.Printf("\n%sOpen Files%s  : %d of unlimited\n", ColorGreen, ColorReset, r.FileContext.OpenFiles)
				} else {
					out.Printf("\nOpen Files  : %d of unlimited\n", r.FileContext.OpenFiles)
				}
			}
			if r.FileContext.OpenFiles > 0 && r.FileContext.FileLimit > 0 {
				usagePercent := float64(r.FileContext.OpenFiles) / float64(r.FileContext.FileLimit) * 100
				if colorEnabled {
					if usagePercent > 80 {
						out.Printf("\n%sOpen Files%s  : %s%d of %d (%.0f%%)%s\n", ColorRed, ColorReset, ColorDimYellow, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent, ColorReset)
					} else {
						out.Printf("\n%sOpen Files%s  : %d of %d (%.0f%%)\n", ColorGreen, ColorReset, r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
					}
				} else {
					out.Printf("\nOpen Files  : %d of %d (%.0f%%)\n", r.FileContext.OpenFiles, r.FileContext.FileLimit, usagePercent)
				}
			}
			if len(r.FileContext.LockedFiles) > 0 {
				count := len(r.FileContext.LockedFiles)
				firstLocked := SanitizeTerminal(r.FileContext.LockedFiles[0])

				if colorEnabled {
					out.Printf("%sLocks%s       : %s\n", ColorGreen, ColorReset, firstLocked)
				} else {
					out.Printf("Locks       : %s\n", firstLocked)
				}

				for i, f := range r.FileContext.LockedFiles[1:] {
					if 1+i >= MaxDisplayItems {
						remaining := count - (1 + i)
						out.Printf("              ... and %d more\n", remaining)
						break
					}
					out.Printf("              %s\n", SanitizeTerminal(f))
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

			label, count := fdSummary(proc)
			if colorEnabled {
				out.Printf("\n%s%s%s: %s\n", ColorGreen, label, ColorReset, count)
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= MaxDisplayItems {
					for _, fd := range proc.FileDescs {
						safeFd := SanitizeTerminalLine(fd)
						safeFd = strings.Replace(safeFd, "->", string(ColorMagenta)+"->"+string(ColorReset), 1)
						out.Printf("  %s\n", ansiString(safeFd))
					}
				} else if len(proc.FileDescs) > MaxDisplayItems {
					out.Printf("  Showing first %d of %d descriptors:\n", MaxDisplayItems, len(proc.FileDescs))
					for i := 0; i < MaxDisplayItems; i++ {
						safeFd := SanitizeTerminalLine(proc.FileDescs[i])
						safeFd = strings.Replace(safeFd, "->", string(ColorMagenta)+"->"+string(ColorReset), 1)
						out.Printf("  %s\n", ansiString(safeFd))
					}
					out.Printf("  ... and %d more\n", len(proc.FileDescs)-MaxDisplayItems)
				}
			} else {
				out.Printf("\n%s: %s\n", label, count)
				if len(proc.FileDescs) > 0 && len(proc.FileDescs) <= MaxDisplayItems {
					for _, fd := range proc.FileDescs {
						out.Printf("  %s\n", SanitizeTerminal(fd))
					}
				} else if len(proc.FileDescs) > MaxDisplayItems {
					out.Printf("  Showing first %d of %d descriptors:\n", MaxDisplayItems, len(proc.FileDescs))
					for i := 0; i < MaxDisplayItems; i++ {
						out.Printf("  %s\n", SanitizeTerminal(proc.FileDescs[i]))
					}
					out.Printf("  ... and %d more\n", len(proc.FileDescs)-MaxDisplayItems)
				}
			}
		}

		// Socket state (for port queries)
		if r.SocketInfo != nil {
			state := SanitizeTerminal(r.SocketInfo.State)
			explanation := SanitizeTerminal(r.SocketInfo.Explanation)
			workaround := SanitizeTerminal(r.SocketInfo.Workaround)
			if colorEnabled {
				out.Printf("%sSocket%s      : %s\n", ColorGreen, ColorReset, state)
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
		if len(r.Children) > 0 {
			out.Println("")
			PrintChildren(w, r.Process, r.Children, colorEnabled)
		}
	}
}

// formatBytes renders a byte count with a binary unit (KB/MB/GB/...), keeping
// large values readable. Values below 1 KB are shown in bytes.
func formatBytes(n uint64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := uint64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGTPE"[exp])
}

// fdSummary labels and counts a process's open descriptors: file descriptors
// against their limit, or on Windows handles, which have no per-process limit
// worth showing.
func fdSummary(p model.Process) (label, count string) {
	switch {
	case runtime.GOOS == "windows":
		return "Handles", strconv.Itoa(p.FDCount)
	case p.FDLimit == 0:
		return "File Descriptors", fmt.Sprintf("%d/unlimited", p.FDCount)
	}
	return "File Descriptors", fmt.Sprintf("%d/%d", p.FDCount, p.FDLimit)
}

// integrityNote describes a Windows integrity level other than the normal
// Medium, for the User line: " (elevated)" for High.
func integrityNote(level string) string {
	switch level {
	case "", "Medium":
		return ""
	case "High":
		return " (elevated)"
	}
	return " (" + strings.ToLower(SanitizeTerminalLine(level)) + " integrity)"
}

// securityLabel renders the Linux security module confining a process, or ""
// when the process is unconfined (most are) or no module is active.
func securityLabel(p model.Process) string {
	if p.SecurityLabel == "" || strings.Contains(p.SecurityLabel, "unconfined") {
		return ""
	}
	label := p.SecurityLabel
	if p.SecurityModule != "" {
		label = p.SecurityModule + " " + label
	}
	return SanitizeTerminalLine(label)
}

// formatSocket renders one row of the Sockets section as
// "<address>:<port> (<PROTO> | <STATE>)". A connection adds its remote end
// (" → ", or " ↔ " when both ends are the process's own), and a listener the
// number of connections it accepted.
func formatSocket(r socketRow) string {
	hostPort := net.JoinHostPort(r.Address, strconv.Itoa(r.Port))
	if r.RemoteAddress != "" && r.RemotePort > 0 {
		arrow := " → "
		if r.internal {
			arrow = " ↔ "
		}
		hostPort += arrow + net.JoinHostPort(r.RemoteAddress, strconv.Itoa(r.RemotePort))
	}
	proto := r.Protocol
	if proto == "" {
		proto = "?"
	}
	state := displayState(r.State)
	switch {
	case r.internal:
		state += ", within the process"
	case r.accepted == 1:
		state += ", 1 connection"
	case r.accepted > 1:
		state += fmt.Sprintf(", %d connections", r.accepted)
	}
	return fmt.Sprintf("%s (%s | %s)", hostPort, proto, state)
}

// socketRow is one row of the Sockets section.
type socketRow struct {
	model.Socket
	accepted int  // connections folded into this listener
	internal bool // both ends of the connection belong to the process
}

// socketRows orders a process's sockets for display. A connection whose two
// ends are both the process's own (on Windows, libraries emulate socket pairs
// this way) shows once. Each other established connection on a port the
// process listens on folds into that listener's count, so a busy server's
// listeners aren't buried under its clients. Outbound connections and
// connections in other states stay as rows of their own.
func socketRows(sockets []model.Socket) []socketRow {
	visible := visibleSockets(sockets)
	sortSockets(visible)

	family := func(s model.Socket) string { return strings.TrimSuffix(s.Protocol, "6") }
	endpoint := func(s model.Socket, addr string, port int) string {
		return family(s) + "|" + net.JoinHostPort(addr, strconv.Itoa(port))
	}
	own := make(map[string]bool) // local ends of the process's connections
	for _, s := range visible {
		if s.State == "ESTABLISHED" {
			own[endpoint(s, s.Address, s.Port)] = true
		}
	}
	shown := make(map[string]bool) // far ends of the pairs already shown

	listener := make(map[string]int) // protocol family and port → row index
	key := func(s model.Socket) string {
		return family(s) + "|" + strconv.Itoa(s.Port)
	}
	rows := make([]socketRow, 0, len(visible))
	for _, s := range visible {
		if s.State == "ESTABLISHED" && s.RemotePort > 0 && own[endpoint(s, s.RemoteAddress, s.RemotePort)] {
			if !shown[endpoint(s, s.Address, s.Port)] {
				shown[endpoint(s, s.RemoteAddress, s.RemotePort)] = true
				rows = append(rows, socketRow{Socket: s, internal: true})
			}
			continue
		}
		if s.State == "ESTABLISHED" {
			if i, ok := listener[key(s)]; ok {
				rows[i].accepted++
				continue
			}
		}
		if s.State == "LISTEN" {
			if _, ok := listener[key(s)]; !ok {
				listener[key(s)] = len(rows)
			}
		}
		rows = append(rows, socketRow{Socket: s})
	}
	return rows
}

// displayState pretty-prints socket states. The kernel-style "LISTEN" reads
// awkwardly next to "ESTABLISHED" / "CLOSE_WAIT", so it's expanded here.
func displayState(state string) string {
	switch state {
	case "":
		return "?"
	case "LISTEN":
		return "LISTENING"
	default:
		return state
	}
}

// socketSortRank orders sockets so listeners come first, then connected
// sockets, then everything else. Within a rank, the renderer further sorts by
// port for stable output.
func socketSortRank(state string) int {
	switch state {
	case "LISTEN":
		return 0
	case "OPEN": // bound UDP — the UDP equivalent of LISTEN
		return 1
	case "ESTABLISHED":
		return 2
	default:
		return 3
	}
}

// visibleSockets returns sockets that have enough information to be rendered.
// Anything with a blank address or port-zero is silently dropped.
func visibleSockets(sockets []model.Socket) []model.Socket {
	out := make([]model.Socket, 0, len(sockets))
	for _, s := range sockets {
		if s.Address != "" && s.Port > 0 {
			out = append(out, s)
		}
	}
	return out
}

// sortSockets orders sockets for the Sockets section in place: listeners
// first, then connected sockets, then everything else, so connections can't
// push a listener past the row limit. Within that, addresses are grouped,
// ports ascend and remote ends break ties.
func sortSockets(sockets []model.Socket) {
	sort.SliceStable(sockets, func(i, j int) bool {
		a, b := sockets[i], sockets[j]
		if ra, rb := socketSortRank(a.State), socketSortRank(b.State); ra != rb {
			return ra < rb
		}
		if a.Address != b.Address {
			return a.Address < b.Address
		}
		if a.Port != b.Port {
			return a.Port < b.Port
		}
		if a.RemoteAddress != b.RemoteAddress {
			return a.RemoteAddress < b.RemoteAddress
		}
		return a.RemotePort < b.RemotePort
	})
}
