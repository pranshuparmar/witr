package output

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/SanCognition/witr/internal/batch"
)

// Color codes (reusing from standard.go)
var (
	tableColorReset  = "\033[0m"
	tableColorRed    = "\033[31m"
	tableColorGreen  = "\033[32m"
	tableColorBlue   = "\033[34m"
	tableColorYellow = "\033[33m"
	tableColorCyan   = "\033[36m"
)

// TableRenderer handles streaming table output
type TableRenderer struct {
	writer       *tabwriter.Writer
	colorEnabled bool
	rowCount     int
	rows         []batch.ProcessSummary // Buffer for sorting
	sortBy       string
}

// NewTableRenderer creates a new streaming table renderer
func NewTableRenderer(colorEnabled bool, sortBy string) *TableRenderer {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	return &TableRenderer{
		writer:       w,
		colorEnabled: colorEnabled,
		sortBy:       sortBy,
		rows:         make([]batch.ProcessSummary, 0),
	}
}

// PrintHeader outputs the table header
func (t *TableRenderer) PrintHeader() {
	header := " PID\tCPU\tMEM\tAGE\tSOURCE\tSCRIPT\tWORKDIR\tREPO"
	if t.colorEnabled {
		fmt.Fprintf(t.writer, "%s%s%s\n", tableColorBlue, header, tableColorReset)
	} else {
		fmt.Fprintln(t.writer, header)
	}

	// Separator line
	fmt.Fprintln(t.writer, " ────\t───\t───\t───\t──────\t──────\t───────\t────")
	t.writer.Flush()
}

// AddRow buffers a row (for sorting) or prints immediately (no sorting)
func (t *TableRenderer) AddRow(p batch.ProcessSummary) {
	if p.Error != nil {
		return // Skip failed analyses
	}

	if t.sortBy != "" {
		// Buffer for sorting later
		t.rows = append(t.rows, p)
	} else {
		// Stream immediately
		t.printRow(p)
	}
	t.rowCount++
}

func (t *TableRenderer) printRow(p batch.ProcessSummary) {
	cpu := fmt.Sprintf("%.0f%%", p.CPU)
	mem := formatMemory(p.MemoryMB)
	// Clean workdir of any newlines
	workDir := strings.TrimSpace(batch.ShortenPath(p.WorkDir))
	script := batch.Truncate(p.Script, 20)

	// Color high CPU/memory
	if t.colorEnabled {
		if p.CPU > 50 {
			cpu = tableColorRed + cpu + tableColorReset
		} else if p.CPU > 20 {
			cpu = tableColorYellow + cpu + tableColorReset
		}
		if p.MemoryMB > 512 {
			mem = tableColorYellow + mem + tableColorReset
		}
		if p.MemoryMB > 1024 {
			mem = tableColorRed + mem + tableColorReset
		}
	}

	// Truncate workdir for display
	if len(workDir) > 25 {
		workDir = "..." + workDir[len(workDir)-22:]
	}

	fmt.Fprintf(t.writer, " %d\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
		p.PID, cpu, mem, p.Age, p.Source, script, workDir, p.GitRepo)
	t.writer.Flush()
}

// Flush sorts (if needed) and prints all buffered rows
func (t *TableRenderer) Flush() {
	if t.sortBy != "" && len(t.rows) > 0 {
		sort.Slice(t.rows, func(i, j int) bool {
			switch t.sortBy {
			case "cpu":
				return t.rows[i].CPU > t.rows[j].CPU // Descending
			case "mem":
				return t.rows[i].MemoryMB > t.rows[j].MemoryMB
			case "age":
				return t.rows[i].StartedAt.Before(t.rows[j].StartedAt) // Oldest first
			case "pid":
				return t.rows[i].PID < t.rows[j].PID
			default:
				return false
			}
		})
		for _, row := range t.rows {
			t.printRow(row)
		}
	}
}

// GetRows returns the buffered rows (for JSON output)
func (t *TableRenderer) GetRows() []batch.ProcessSummary {
	return t.rows
}

// PrintFooter outputs the summary line
func (t *TableRenderer) PrintFooter(total int, errors int, elapsed time.Duration) {
	fmt.Println()
	if t.colorEnabled {
		fmt.Printf("%sFound %d processes%s", tableColorGreen, total, tableColorReset)
	} else {
		fmt.Printf("Found %d processes", total)
	}
	if errors > 0 {
		if t.colorEnabled {
			fmt.Printf(" (%s%d errors%s)", tableColorYellow, errors, tableColorReset)
		} else {
			fmt.Printf(" (%d errors)", errors)
		}
	}
	fmt.Printf(" (%.1fs)\n", elapsed.Seconds())
}

// PrintBatchJSON outputs batch results as JSON
func PrintBatchJSON(rows []batch.ProcessSummary) {
	type jsonRow struct {
		PID      int     `json:"pid"`
		CPU      float64 `json:"cpu"`
		MemoryMB int     `json:"memory_mb"`
		Age      string  `json:"age"`
		Source   string  `json:"source"`
		Script   string  `json:"script"`
		WorkDir  string  `json:"workdir"`
		GitRepo  string  `json:"repo"`
		Command  string  `json:"command"`
		Cmdline  string  `json:"cmdline"`
		User     string  `json:"user"`
		Health   string  `json:"health"`
	}

	output := make([]jsonRow, 0, len(rows))
	for _, r := range rows {
		output = append(output, jsonRow{
			PID:      r.PID,
			CPU:      r.CPU,
			MemoryMB: r.MemoryMB,
			Age:      r.Age,
			Source:   r.Source,
			Script:   r.Script,
			WorkDir:  r.WorkDir,
			GitRepo:  r.GitRepo,
			Command:  r.Command,
			Cmdline:  r.Cmdline,
			User:     r.User,
			Health:   r.Health,
		})
	}

	enc, _ := json.MarshalIndent(output, "", "  ")
	fmt.Println(string(enc))
}

// formatMemory converts MB to human-readable format
func formatMemory(mb int) string {
	if mb >= 1024 {
		return fmt.Sprintf("%.1fG", float64(mb)/1024)
	}
	return fmt.Sprintf("%dM", mb)
}
