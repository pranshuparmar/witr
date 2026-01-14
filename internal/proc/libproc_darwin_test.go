//go:build darwin && !internal_witr_cgo_disabled

package proc

import (
	"os"
	"testing"
)

func TestReadDarwinIO(t *testing.T) {
	pid := os.Getpid()
	stats, err := readDarwinIO(pid)
	if err != nil {
		t.Fatalf("readDarwinIO(%d) error: %v", pid, err)
	}
	if stats.ReadBytes == 0 && stats.WriteBytes == 0 {
		t.Log("disk I/O counters are zero; process likely idle")
	}
}
