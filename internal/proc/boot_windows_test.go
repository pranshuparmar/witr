//go:build windows

package proc

import (
	"testing"
	"time"
)

func TestBootTimeIsInThePast(t *testing.T) {
	if bt := bootTime(); !bt.Before(time.Now()) {
		t.Errorf("bootTime() = %v, expected a time in the past", bt)
	}
}

func TestBootTimeIsAfter2000(t *testing.T) {
	floor := time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)
	if bt := bootTime(); bt.Before(floor) {
		t.Errorf("bootTime() = %v, expected >= %v", bt, floor)
	}
}

func TestBootTimeUptimeReasonable(t *testing.T) {
	uptime := time.Since(bootTime())
	if uptime < 0 {
		t.Errorf("uptime is negative: %v", uptime)
	}
	if uptime > 100*365*24*time.Hour {
		t.Errorf("uptime > 100 years: %v", uptime)
	}
}
