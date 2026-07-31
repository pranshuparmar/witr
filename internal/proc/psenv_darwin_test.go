//go:build darwin

package proc

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// fakePSScript stands in for /bin/ps and renders lstart, %cpu and %mem the way
// the real ps does under the locale it is invoked with: C uses the 5-token
// English date and a dot decimal separator, ko_KR.UTF-8 a 7-token date, and
// de_DE.UTF-8 a comma decimal separator.
const fakePSScript = `#!/bin/sh
case "$LC_ALL" in
C)
	lstart='Wed Jul 29 15:37:10 2026'
	cpu='1.5'
	mem='0.4'
	;;
ko_KR.UTF-8)
	lstart='2026년  7월 29일 수요일 15시 37분 10초'
	cpu='1.5'
	mem='0.4'
	;;
*)
	lstart='Mi. 29 Juli 15:37:10 2026'
	cpu='1,5'
	mem='0,4'
	;;
esac
case "$*" in
*comm=*)
	printf '%s\n' " 4242 /usr/local/bin/witrtest"
	;;
*'%cpu=,rss='*)
	printf '%s\n' "  $cpu 20480"
	;;
*)
	printf '%s\n' "  PID  PPID USER     STARTED  %CPU   RSS %MEM ARGS"
	printf '%s\n' " 4242     1 testuser $lstart $cpu 20480 $mem /usr/local/bin/witrtest --serve"
	;;
esac
`

func installFakePS(t *testing.T, locale string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "ps"), []byte(fakePSScript), 0o755); err != nil {
		t.Fatalf("write ps script: %v", err)
	}
	t.Setenv("PATH", dir+":"+os.Getenv("PATH"))
	t.Setenv("LC_ALL", locale)
}

func TestListProcessesIgnoresCallerLocale(t *testing.T) {
	wantStarted := time.Date(2026, time.July, 29, 15, 37, 10, 0, time.UTC)

	for _, locale := range []string{"ko_KR.UTF-8", "de_DE.UTF-8"} {
		t.Run(locale, func(t *testing.T) {
			installFakePS(t, locale)

			procs, err := ListProcesses()
			if err != nil {
				t.Fatalf("ListProcesses() error = %v", err)
			}
			if len(procs) != 1 {
				t.Fatalf("ListProcesses() returned %d processes, want 1", len(procs))
			}

			p := procs[0]
			if !p.StartedAt.Equal(wantStarted) {
				t.Errorf("StartedAt = %v, want %v", p.StartedAt, wantStarted)
			}
			if p.CPUPercent != 1.5 {
				t.Errorf("CPUPercent = %v, want 1.5", p.CPUPercent)
			}
			if p.MemoryRSS != 20480*1024 {
				t.Errorf("MemoryRSS = %d, want %d", p.MemoryRSS, 20480*1024)
			}
			if p.MemoryPercent != 0.4 {
				t.Errorf("MemoryPercent = %v, want 0.4", p.MemoryPercent)
			}
			if want := "/usr/local/bin/witrtest --serve"; p.Cmdline != want {
				t.Errorf("Cmdline = %q, want %q", p.Cmdline, want)
			}
		})
	}
}

func TestGetCPUAndMemoryUsageIgnoresCallerLocale(t *testing.T) {
	for _, locale := range []string{"ko_KR.UTF-8", "de_DE.UTF-8"} {
		t.Run(locale, func(t *testing.T) {
			installFakePS(t, locale)

			cpu, mem, err := getCPUAndMemoryUsage(4242)
			if err != nil {
				t.Fatalf("getCPUAndMemoryUsage() error = %v", err)
			}
			if cpu != 1.5 {
				t.Errorf("cpu = %v, want 1.5", cpu)
			}
			if mem != 20480*1024 {
				t.Errorf("mem = %d, want %d", mem, 20480*1024)
			}
		})
	}
}
