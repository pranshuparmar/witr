//go:build darwin

package launchd

import (
	"testing"
)

// A StartCalendarInterval dict is parsed by a helper that consumes its own
// closing tag. The caller must balance the depth it counted on entry or every
// root key after the schedule is silently dropped.
func TestParsePlistXML_KeysAfterCalendarInterval(t *testing.T) {
	data := []byte(`<plist version="1.0"><dict>
	<key>Label</key><string>com.example.test</string>
	<key>StartCalendarInterval</key>
	<dict><key>Hour</key><integer>9</integer></dict>
	<key>Comment</key><string>after-schedule</string>
	<key>KeepAlive</key><true/>
	<key>ProgramArguments</key>
	<array><string>/usr/bin/test</string></array>
</dict></plist>`)

	info := &LaunchdInfo{}
	if err := parsePlistXML(data, info); err != nil {
		t.Fatalf("parsePlistXML: %v", err)
	}

	if info.Label != "com.example.test" {
		t.Errorf("Label = %q, want %q", info.Label, "com.example.test")
	}
	if info.Comment != "after-schedule" {
		t.Errorf("Comment after StartCalendarInterval = %q, want %q", info.Comment, "after-schedule")
	}
	if !info.KeepAlive {
		t.Error("KeepAlive after StartCalendarInterval was dropped")
	}
	if len(info.ProgramArguments) != 1 || info.ProgramArguments[0] != "/usr/bin/test" {
		t.Errorf("ProgramArguments after StartCalendarInterval = %q, want [/usr/bin/test]", info.ProgramArguments)
	}
	if info.StartCalendarInterval == "" {
		t.Error("StartCalendarInterval was not parsed")
	}
}
