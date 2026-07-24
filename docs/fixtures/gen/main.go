//go:build fixtures

// Command gen renders golden fixtures for the playground using witr's *real*
// output package. The browser engine (docs/js/engine.js) must reproduce
// these byte-for-byte; scripts/check-fixtures.mjs enforces it.
//
// Run from the repo root:
//
//	go run -tags fixtures ./docs/fixtures/gen
//
// It reads docs/worlds/webbox.json, builds model.Result values, renders
// each fixture command, and writes docs/fixtures/*.txt plus _meta.json
// (which pins the clock so the browser check is deterministic).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/pranshuparmar/witr/internal/output"
	"github.com/pranshuparmar/witr/pkg/model"
)

// ---- world schema (subset the fixtures need) --------------------------------

type socket struct {
	Address, Protocol, State string
	Port                     int
}
type source struct {
	Type, Name, Description, UnitFile string
	Details                           map[string]string
}
type proc struct {
	PID, PPID                      int
	Command, Cmdline, User         string
	StartedAgo                     int64
	WorkingDir, GitRepo, GitBranch string
	Forked, Health                 string
	Sockets                        []socket
	Env                            []string
	LockedFiles                    []string
	ThreadCount, FDCount, FDLimit  int
	RestartCount                   int
	Memory                         *struct{ VMS, RSS, Shared uint64 }
	IO                             *struct{ ReadBytes, WriteBytes, ReadOps, WriteOps uint64 }
	Source                         *source
	Warnings                       []string
}
type world struct {
	Processes []proc                `json:"processes"`
	Locks     []lock                `json:"locks"`
	Overrides map[string]socketInfo `json:"socketOverrides"`
}
type lock struct {
	PID  int
	Path string
}
type socketInfo struct {
	State, Explanation, Workaround string
}

// A fixture command.
type fixture struct {
	name  string // output filename stem
	mode  string // standard|verbose|short|tree|warnings|env
	kind  string // name|pid|port|file
	value string
}

func main() {
	root := repoRoot()
	var w world
	readJSON(filepath.Join(root, "docs/worlds/webbox.json"), &w)

	now := time.Now().UTC()
	byPID := map[int]proc{}
	for _, p := range w.Processes {
		byPID[p.PID] = p
	}

	fixtures := []fixture{
		{"node_standard", "standard", "name", "node"},
		{"node_verbose", "verbose", "name", "node"},
		{"port5000_short", "short", "port", "5000"},
		{"pid40141_tree", "tree", "pid", "40141"},
		{"dpkg_file", "standard", "file", "/var/lib/dpkg/lock"},
		{"python_warnings", "warnings", "pid", "8123"},
		{"node_env", "env", "pid", "14233"},
		{"postgres_port_verbose", "verbose", "port", "5432"},
	}

	outDir := filepath.Join(root, "docs/fixtures")
	meta := map[string]any{
		"generatedAtMs": now.UnixMilli(),
		"world":         "webbox",
		"cases":         []map[string]string{},
	}
	var cases []map[string]string

	for _, f := range fixtures {
		pid := resolve(w, byPID, f.kind, f.value)
		if pid == 0 {
			fmt.Fprintf(os.Stderr, "fixture %s: could not resolve %s %s\n", f.name, f.kind, f.value)
			os.Exit(1)
		}
		r := buildResult(w, byPID, pid, now, f)

		for _, color := range []bool{false, true} {
			var buf bytes.Buffer
			render(&buf, r, f.mode, color)
			ext := ".txt"
			if color {
				ext = ".ansi"
			}
			path := filepath.Join(outDir, f.name+ext)
			if err := os.WriteFile(path, buf.Bytes(), 0o644); err != nil {
				panic(err)
			}
		}
		cases = append(cases, map[string]string{
			"name": f.name, "mode": f.mode, "kind": f.kind, "value": f.value,
			"pid": strconv.Itoa(pid),
		})
	}
	meta["cases"] = cases
	b, _ := json.MarshalIndent(meta, "", "  ")
	if err := os.WriteFile(filepath.Join(outDir, "_meta.json"), b, 0o644); err != nil {
		panic(err)
	}
	fmt.Printf("wrote %d fixtures to %s\n", len(fixtures), outDir)
}

func render(buf *bytes.Buffer, r model.Result, mode string, color bool) {
	switch mode {
	case "short":
		output.RenderShort(buf, r, color)
	case "tree":
		output.PrintTree(buf, r.Ancestry, r.Children, color)
	case "warnings":
		output.RenderWarnings(buf, r, color)
	case "env":
		output.RenderEnvOnly(buf, r, color)
	case "verbose":
		output.RenderStandard(buf, r, color, true)
	default:
		output.RenderStandard(buf, r, color, false)
	}
}

func resolve(w world, byPID map[int]proc, kind, value string) int {
	switch kind {
	case "pid":
		n, _ := strconv.Atoi(value)
		if _, ok := byPID[n]; ok {
			return n
		}
	case "port":
		n, _ := strconv.Atoi(value)
		var fallback int
		for _, p := range w.Processes {
			for _, s := range p.Sockets {
				if s.Port == n {
					if s.State == "LISTEN" {
						return p.PID
					}
					if fallback == 0 {
						fallback = p.PID
					}
				}
			}
		}
		return fallback
	case "file":
		for _, l := range w.Locks {
			if l.Path == value {
				return l.PID
			}
		}
		for _, p := range w.Processes {
			for _, lf := range p.LockedFiles {
				if strings.TrimSpace(strings.Split(lf, "(")[0]) == value {
					return p.PID
				}
			}
		}
	case "name":
		lower := strings.ToLower(value)
		var hits []int
		for _, p := range w.Processes {
			if strings.Contains(strings.ToLower(p.Command), lower) || strings.Contains(strings.ToLower(p.Cmdline), lower) {
				hits = append(hits, p.PID)
			}
		}
		sort.Ints(hits)
		if len(hits) == 1 {
			return hits[0]
		}
	}
	return 0
}

func toModelProc(p proc, now time.Time) model.Process {
	mp := model.Process{
		PID: p.PID, PPID: p.PPID, Command: p.Command, Cmdline: p.Cmdline,
		User: p.User, WorkingDir: p.WorkingDir, GitRepo: p.GitRepo, GitBranch: p.GitBranch,
		Forked: p.Forked, Health: p.Health, Env: p.Env,
		ThreadCount: p.ThreadCount, FDCount: p.FDCount, FDLimit: uint64(p.FDLimit),
	}
	if p.StartedAgo != 0 {
		mp.StartedAt = now.Add(-time.Duration(p.StartedAgo) * time.Second)
	}
	for _, s := range p.Sockets {
		mp.Sockets = append(mp.Sockets, model.Socket{Address: s.Address, Port: s.Port, Protocol: s.Protocol, State: s.State})
	}
	if p.Memory != nil {
		mp.Memory = model.MemoryInfo{VMS: p.Memory.VMS, RSS: p.Memory.RSS, Shared: p.Memory.Shared}
	}
	if p.IO != nil {
		mp.IO = model.IOStats{ReadBytes: p.IO.ReadBytes, WriteBytes: p.IO.WriteBytes, ReadOps: p.IO.ReadOps, WriteOps: p.IO.WriteOps}
	}
	return mp
}

func buildResult(w world, byPID map[int]proc, pid int, now time.Time, f fixture) model.Result {
	target := byPID[pid]

	// Ancestry: pid1 ... target.
	var chain []model.Process
	seen := map[int]bool{}
	cur := target
	for {
		chain = append(chain, toModelProc(cur, now))
		seen[cur.PID] = true
		parent, ok := byPID[cur.PPID]
		if !ok || cur.PPID == 0 || seen[parent.PID] {
			break
		}
		cur = parent
	}
	for i, j := 0, len(chain)-1; i < j; i, j = i+1, j-1 {
		chain[i], chain[j] = chain[j], chain[i]
	}

	// Children.
	var kids []model.Process
	for _, p := range w.Processes {
		if p.PPID == pid && p.PID != pid {
			kids = append(kids, toModelProc(p, now))
		}
	}
	sort.Slice(kids, func(i, j int) bool { return kids[i].PID < kids[j].PID })

	src := resolveSource(byPID, target)
	// RestartCount comes only from a systemd unit's NRestarts, mirroring
	// pipeline.AnalyzePID.
	restart := 0
	if src.Type == model.SourceSystemd {
		if v, ok := src.Details["NRestarts"]; ok {
			restart, _ = strconv.Atoi(v)
		}
	}
	r := model.Result{
		Process:      toModelProc(target, now),
		Ancestry:     chain,
		Children:     kids,
		Source:       src,
		RestartCount: restart,
		Warnings:     target.Warnings,
	}
	if len(target.LockedFiles) > 0 {
		r.FileContext = &model.FileContext{LockedFiles: target.LockedFiles}
	}
	if f.kind == "port" && f.mode == "verbose" {
		if o, ok := w.Overrides[f.value]; ok {
			r.SocketInfo = &model.SocketInfo{State: displayState(o.State), Explanation: o.Explanation, Workaround: o.Workaround}
		} else {
			for _, s := range target.Sockets {
				if strconv.Itoa(s.Port) == f.value {
					r.SocketInfo = &model.SocketInfo{State: displayState(s.State)}
				}
			}
		}
	}
	return r
}

func resolveSource(byPID map[int]proc, target proc) model.Source {
	p := target
	for {
		if p.Source != nil {
			s := p.Source
			return model.Source{Type: model.SourceType(s.Type), Name: s.Name, Description: s.Description, UnitFile: s.UnitFile, Details: s.Details}
		}
		parent, ok := byPID[p.PPID]
		if !ok || p.PPID == 0 {
			break
		}
		p = parent
	}
	return model.Source{Type: model.SourceUnknown}
}

func displayState(s string) string {
	if s == "LISTEN" {
		return "LISTENING"
	}
	if s == "" {
		return "?"
	}
	return s
}

// ---- helpers ---------------------------------------------------------------

func repoRoot() string {
	dir, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			panic("go.mod not found")
		}
		dir = parent
	}
}

func readJSON(path string, v any) {
	b, err := os.ReadFile(path)
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(b, v); err != nil {
		panic(err)
	}
}
