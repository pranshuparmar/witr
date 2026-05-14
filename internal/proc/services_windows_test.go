//go:build windows

package proc

import (
	"testing"
	"time"
)

func TestServiceMapCacheRespectsTTL(t *testing.T) {
	first, err := serviceMapForPIDs()
	if err != nil {
		t.Fatalf("serviceMapForPIDs: %v", err)
	}
	second, err := serviceMapForPIDs()
	if err != nil {
		t.Fatalf("second call: %v", err)
	}
	if len(first) != len(second) {
		t.Errorf("cached map size mismatch: %d vs %d", len(first), len(second))
	}
}

func TestServiceMapCacheRefreshesAfterTTL(t *testing.T) {
	originalTTL := serviceMapCacheTTL
	serviceMapCacheTTL = 1 * time.Millisecond
	defer func() { serviceMapCacheTTL = originalTTL }()

	if _, err := serviceMapForPIDs(); err != nil {
		t.Fatalf("first call: %v", err)
	}
	firstTime := serviceMapCacheTime

	time.Sleep(10 * time.Millisecond)

	if _, err := serviceMapForPIDs(); err != nil {
		t.Fatalf("second call: %v", err)
	}
	if !serviceMapCacheTime.After(firstTime) {
		t.Errorf("cache not refreshed after TTL: firstTime=%v secondTime=%v",
			firstTime, serviceMapCacheTime)
	}
}

func TestUtf16PtrToStringNilSafe(t *testing.T) {
	if got := utf16PtrToString(nil); got != "" {
		t.Errorf("utf16PtrToString(nil) = %q, want empty string", got)
	}
}

func TestDetectWindowsServiceSourceUnknownPIDReturnsEmpty(t *testing.T) {
	if got := detectWindowsServiceSource(0); got != "" {
		t.Errorf("detectWindowsServiceSource(0) = %q, want empty", got)
	}
}

func TestServiceMapInvariants(t *testing.T) {
	services, err := serviceMapForPIDs()
	if err != nil {
		t.Fatalf("serviceMapForPIDs: %v", err)
	}
	if len(services) == 0 {
		t.Skipf("no running services found; nothing to validate")
	}
	for pid, name := range services {
		if pid == 0 {
			t.Errorf("service map contains zero PID entry (name=%q)", name)
		}
		if name == "" {
			t.Errorf("service map contains empty name for PID %d", pid)
		}
	}
}
