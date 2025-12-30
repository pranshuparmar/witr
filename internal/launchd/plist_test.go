//go:build darwin

package launchd

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc"
	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
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
		want []string
	}{
		{"runAtLoad", LaunchdInfo{RunAtLoad: true}, []string{"RunAtLoad (starts at login/boot)"}},
		{"interval", LaunchdInfo{StartInterval: 3600}, []string{"StartInterval (every 1h)"}},
		{
			"multiple",
			LaunchdInfo{RunAtLoad: true, StartInterval: 300, WatchPaths: []string{"/tmp"}},
			[]string{
				"RunAtLoad (starts at login/boot)",
				"StartInterval (every 5m)",
				"WatchPaths: /tmp",
			},
		},
		{"queue", LaunchdInfo{QueueDirectories: []string{"/var/spool"}}, []string{"QueueDirectories: /var/spool"}},
		{"none", LaunchdInfo{}, nil},
		{"calendar", LaunchdInfo{StartCalendarInterval: "daily"}, []string{"StartCalendarInterval (daily)"}},
		{"multiWatch", LaunchdInfo{WatchPaths: []string{"/a", "/b"}}, []string{"WatchPaths: /a", "WatchPaths: /b"}},
		{"multiQueue", LaunchdInfo{QueueDirectories: []string{"/a", "/b"}}, []string{"QueueDirectories: /a", "QueueDirectories: /b"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.FormatTriggers(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("FormatTriggers() = %v, want %v", got, tt.want)
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
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	// 1. Success case
	// launchctl blame <pid> -> returns "domain service reason"
	mockExec.EXPECT().Run("launchctl", "blame", "123").Return([]byte("system/com.apple.finder semantic"), nil)

	_, _, err := GetServiceLabel(123)
	if err != nil {
		t.Errorf("GetServiceLabel failed: %v", err)
	}

	// 2. Failure case
	mockExec.EXPECT().Run("launchctl", "blame", "99999").Return(nil, errors.New("process not found"))
	_, _, err = GetServiceLabel(99999)
	if err == nil {
		t.Error("GetServiceLabel should fail for nonexistent PID")
	}
}

func TestFindServiceByPID(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	// Mock list output
	output := `PID	Status	Label
123	0	com.test.service
456	0	com.other.service
`
	// 1. Found
	mockExec.EXPECT().Run("launchctl", "list").Return([]byte(output), nil)
	label, _ := findServiceByPID(123)
	if label != "com.test.service" {
		t.Errorf("findServiceByPID(123) = %q, want com.test.service", label)
	}

	// 2. Not found
	mockExec.EXPECT().Run("launchctl", "list").Return([]byte(output), nil)
	label, _ = findServiceByPID(999)
	if label != "" {
		t.Errorf("findServiceByPID(999) found %q, want empty", label)
	}

	// 3. Error case
	mockExec.EXPECT().Run("launchctl", "list").Return(nil, errors.New("failed"))
	findServiceByPID(1)
}

func TestFindPlistPath(t *testing.T) {
	dir := t.TempDir()
	origPaths := plistSearchPaths
	plistSearchPaths = []string{dir}
	defer func() { plistSearchPaths = origPaths }()

	label := "com.test.service"
	path := filepath.Join(dir, label+".plist")
	if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	if got := FindPlistPath(label); got != path {
		t.Fatalf("FindPlistPath = %q, want %q", got, path)
	}
	if got := FindPlistPath("nonexistent.service.xyz123"); got != "" {
		t.Fatalf("FindPlistPath for missing label = %q, want empty", got)
	}
}

func TestParsePlist(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	// 1. Success
	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>com.test.plist</string>
</dict>
</plist>`
	mockExec.EXPECT().Run("plutil", "-convert", "xml1", "-o", "-", "/path/to/test.plist").Return([]byte(xml), nil)

	info, err := ParsePlist("/path/to/test.plist")
	if err != nil {
		t.Errorf("ParsePlist failed: %v", err)
	}
	if info.Label != "com.test.plist" {
		t.Errorf("ParsePlist label = %q, want com.test.plist", info.Label)
	}

	// 2. Failure
	mockExec.EXPECT().Run("plutil", "-convert", "xml1", "-o", "-", "/nonexistent.plist").Return(nil, errors.New("file not found"))
	_, err = ParsePlist("/nonexistent.plist")
	if err == nil {
		t.Error("ParsePlist should fail for nonexistent file")
	}
}

func TestGetLaunchdInfo(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	proc.SetExecutor(mockExec)
	defer proc.ResetExecutor()

	dir := t.TempDir()
	origPaths := plistSearchPaths
	plistSearchPaths = []string{dir}
	defer func() { plistSearchPaths = origPaths }()

	label := "com.test.service"
	path := filepath.Join(dir, label+".plist")
	if err := os.WriteFile(path, []byte("dummy"), 0o644); err != nil {
		t.Fatalf("WriteFile failed: %v", err)
	}

	xml := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>` + label + `</string>
</dict>
</plist>`

	gomock.InOrder(
		mockExec.EXPECT().Run("launchctl", "blame", "123").Return([]byte("system/"+label), nil),
		mockExec.EXPECT().Run("plutil", "-convert", "xml1", "-o", "-", path).Return([]byte(xml), nil),
	)

	info, err := GetLaunchdInfo(123)
	if err != nil {
		t.Fatalf("GetLaunchdInfo failed: %v", err)
	}
	if info.Label != label {
		t.Fatalf("GetLaunchdInfo label = %q, want %q", info.Label, label)
	}
	if info.Domain != "system" {
		t.Fatalf("GetLaunchdInfo domain = %q, want system", info.Domain)
	}
	if info.PlistPath != path {
		t.Fatalf("GetLaunchdInfo plist path = %q, want %q", info.PlistPath, path)
	}
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
	if err := parsePlistXML([]byte(""), info); err != nil {
		t.Fatalf("parsePlistXML empty returned error: %v", err)
	}
	if info.Label != "" {
		t.Fatalf("parsePlistXML empty label = %q, want empty", info.Label)
	}
}
