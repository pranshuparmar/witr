//go:build darwin

package launchd

import (
	"testing"
)

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		seconds int
		want    string
	}{
		{30, "30s"}, {60, "1m"}, {90, "1m"}, {3600, "1h"}, {7200, "2h"},
		{86400, "1d"}, {172800, "2d"}, {0, "0s"}, {59, "59s"}, {3599, "59m"},
	}
	for _, tt := range tests {
		if got := formatDuration(tt.seconds); got != tt.want {
			t.Errorf("formatDuration(%d) = %q, want %q", tt.seconds, got, tt.want)
		}
	}
}

func TestFormatTriggers(t *testing.T) {
	tests := []struct {
		name string
		info LaunchdInfo
		want int
	}{
		{"runAtLoad", LaunchdInfo{RunAtLoad: true}, 1},
		{"interval", LaunchdInfo{StartInterval: 3600}, 1},
		{"multiple", LaunchdInfo{RunAtLoad: true, StartInterval: 300, WatchPaths: []string{"/tmp"}}, 3},
		{"queue", LaunchdInfo{QueueDirectories: []string{"/var/spool"}}, 1},
		{"none", LaunchdInfo{}, 0},
		{"calendar", LaunchdInfo{StartCalendarInterval: "daily"}, 1},
		{"multiWatch", LaunchdInfo{WatchPaths: []string{"/a", "/b"}}, 2},
		{"multiQueue", LaunchdInfo{QueueDirectories: []string{"/a", "/b"}}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := len(tt.info.FormatTriggers()); got != tt.want {
				t.Errorf("FormatTriggers() len = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestDomainDescription(t *testing.T) {
	tests := []struct {
		domain string
		want   string
	}{
		{"system", "Launch Daemon"},
		{"gui/501", "Launch Agent"},
		{"user", "Launch Agent"},
		{"", "launchd service"},
		{"unknown", "launchd service"},
	}
	for _, tt := range tests {
		info := &LaunchdInfo{Domain: tt.domain}
		if got := info.DomainDescription(); got != tt.want {
			t.Errorf("DomainDescription() = %q, want %q", got, tt.want)
		}
	}
}

func TestHandleStringValue(t *testing.T) {
	info := &LaunchdInfo{}
	handleStringValue(info, "Label", "com.test")
	if info.Label != "com.test" {
		t.Error("handleStringValue failed for Label")
	}
	handleStringValue(info, "Program", "/usr/bin/test")
	if info.Program != "/usr/bin/test" {
		t.Error("handleStringValue failed for Program")
	}
	handleStringValue(info, "Unknown", "val")
}

func TestHandleIntValue(t *testing.T) {
	info := &LaunchdInfo{}
	handleIntValue(info, "StartInterval", 3600)
	if info.StartInterval != 3600 {
		t.Error("handleIntValue failed")
	}
	handleIntValue(info, "Unknown", 999)
}

func TestHandleBoolValue(t *testing.T) {
	info := &LaunchdInfo{}
	handleBoolValue(info, "RunAtLoad", true)
	if !info.RunAtLoad {
		t.Error("handleBoolValue failed for RunAtLoad")
	}
	handleBoolValue(info, "KeepAlive", true)
	if !info.KeepAlive {
		t.Error("handleBoolValue failed for KeepAlive")
	}
	handleBoolValue(info, "Unknown", true)
}

func TestHandleArrayValue(t *testing.T) {
	info := &LaunchdInfo{}
	handleArrayValue(info, "ProgramArguments", []string{"/usr/bin/test", "--flag"})
	if len(info.ProgramArguments) != 2 {
		t.Error("handleArrayValue failed for ProgramArguments")
	}
	handleArrayValue(info, "WatchPaths", []string{"/tmp"})
	if len(info.WatchPaths) != 1 {
		t.Error("handleArrayValue failed for WatchPaths")
	}
	handleArrayValue(info, "QueueDirectories", []string{"/var"})
	if len(info.QueueDirectories) != 1 {
		t.Error("handleArrayValue failed for QueueDirectories")
	}
	handleArrayValue(info, "Unknown", []string{"a"})
}

func TestGetServiceLabel(t *testing.T) {
	_, _, _ = GetServiceLabel(1)
	_, _, _ = GetServiceLabel(99999)
}

func TestFindServiceByPID(t *testing.T) {
	_, _ = findServiceByPID(1)
	_, _ = findServiceByPID(99999)
}

func TestFindPlistPath(t *testing.T) {
	_ = FindPlistPath("com.apple.finder")
	_ = FindPlistPath("nonexistent.service.xyz123")
}

func TestParsePlist(t *testing.T) {
	_, _ = ParsePlist("/nonexistent/path.plist")
}

func TestGetLaunchdInfo(t *testing.T) {
	_, _ = GetLaunchdInfo(1)
	_, _ = GetLaunchdInfo(99999)
}

func TestParsePlistXML(t *testing.T) {
	info := &LaunchdInfo{}
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.test.service</string>
	<key>Program</key>
	<string>/usr/bin/test</string>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<false/>
	<key>StartInterval</key>
	<integer>3600</integer>
	<key>ProgramArguments</key>
	<array>
		<string>/usr/bin/test</string>
		<string>--flag</string>
	</array>
</dict>
</plist>`
	err := parsePlistXML([]byte(xml), info)
	if err != nil {
		t.Errorf("parsePlistXML failed: %v", err)
	}
	if info.Label != "com.test.service" {
		t.Errorf("Label = %q, want com.test.service", info.Label)
	}
	if info.Program != "/usr/bin/test" {
		t.Errorf("Program = %q, want /usr/bin/test", info.Program)
	}
	if !info.RunAtLoad {
		t.Error("RunAtLoad should be true")
	}
	if info.StartInterval != 3600 {
		t.Errorf("StartInterval = %d, want 3600", info.StartInterval)
	}
	if len(info.ProgramArguments) != 2 {
		t.Errorf("ProgramArguments len = %d, want 2", len(info.ProgramArguments))
	}
}

func TestParsePlistXMLEmpty(t *testing.T) {
	info := &LaunchdInfo{}
	_ = parsePlistXML([]byte(""), info)
}
