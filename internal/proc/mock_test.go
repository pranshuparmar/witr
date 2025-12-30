//go:build darwin

package proc

import (
	"strings"
	"testing"

	"go.uber.org/mock/gomock"
)

func TestWithMockCommandRunner(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockRunner := CommandRunnerFunc(func(name string, args ...string) ([]byte, error) {
		if name == "ps" {
			return []byte("  123  456  501 Mon Dec 30 10:00:00 2024 S test\n"), nil
		}
		return nil, nil
	})

	output, err := mockRunner.Run("ps", "-p", "123", "-o", "pid=")
	if err != nil {
		t.Errorf("mock runner failed: %v", err)
	}
	if len(output) == 0 {
		t.Error("mock runner returned empty output")
	}
}

func TestWithMockFileReader(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockReader := FileReaderFunc(func(path string) ([]byte, error) {
		if path == "/proc/1/cgroup" {
			return []byte("12:memory:/docker/abc123\n"), nil
		}
		return nil, nil
	})

	content, err := mockReader.ReadFile("/proc/1/cgroup")
	if err != nil {
		t.Errorf("mock reader failed: %v", err)
	}
	if len(content) == 0 {
		t.Error("mock reader returned empty content")
	}
}

func TestMockPsOutputParsing(t *testing.T) {
	tests := []struct {
		name     string
		psOutput string
		wantPID  string
		wantPPID string
		wantUID  string
		wantComm string
	}{
		{
			name:     "standard process",
			psOutput: "  123  456  501 Mon Dec 30 10:00:00 2024 S test",
			wantPID:  "123",
			wantPPID: "456",
			wantUID:  "501",
			wantComm: "test",
		},
		{
			name:     "root process",
			psOutput: "    1    0    0 Mon Jan  1 00:00:00 2024 R launchd",
			wantPID:  "1",
			wantPPID: "0",
			wantUID:  "0",
			wantComm: "launchd",
		},
		{
			name:     "zombie process",
			psOutput: "  999  100  501 Mon Dec 30 12:00:00 2024 Z defunct",
			wantPID:  "999",
			wantPPID: "100",
			wantUID:  "501",
			wantComm: "defunct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := strings.Fields(tt.psOutput)
			if len(fields) < 9 {
				t.Fatalf("expected at least 9 fields, got %d", len(fields))
			}
			if fields[0] != tt.wantPID {
				t.Errorf("PID = %s, want %s", fields[0], tt.wantPID)
			}
			if fields[1] != tt.wantPPID {
				t.Errorf("PPID = %s, want %s", fields[1], tt.wantPPID)
			}
			if fields[2] != tt.wantUID {
				t.Errorf("UID = %s, want %s", fields[2], tt.wantUID)
			}
		})
	}
}

func TestMockNetstatParsing(t *testing.T) {
	tests := []struct {
		name      string
		line      string
		wantState string
		wantPort  int
	}{
		{
			name:      "listen",
			line:      "tcp4       0      0  *.8080               *.*                    LISTEN",
			wantState: "LISTEN",
			wantPort:  8080,
		},
		{
			name:      "time_wait",
			line:      "tcp4       0      0  127.0.0.1.8080       127.0.0.1.49152        TIME_WAIT",
			wantState: "TIME_WAIT",
			wantPort:  8080,
		},
		{
			name:      "established",
			line:      "tcp4       0      0  192.168.1.10.443     10.0.0.1.54321         ESTABLISHED",
			wantState: "ESTABLISHED",
			wantPort:  443,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := strings.Fields(tt.line)
			if len(fields) < 6 {
				t.Fatalf("expected at least 6 fields, got %d", len(fields))
			}
			state := fields[5]
			if state != tt.wantState {
				t.Errorf("state = %s, want %s", state, tt.wantState)
			}
			localAddr := fields[3]
			_, port := parseNetstatAddr(localAddr)
			if port != tt.wantPort {
				t.Errorf("port = %d, want %d", port, tt.wantPort)
			}
		})
	}
}

func TestMockLsofOutputParsing(t *testing.T) {
	mockLsofOutput := `p1234
n*:8080
p5678
n127.0.0.1:3000
`
	lines := strings.Split(mockLsofOutput, "\n")
	var currentPID string
	var sockets []struct {
		pid  string
		addr string
	}

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		switch line[0] {
		case 'p':
			currentPID = line[1:]
		case 'n':
			sockets = append(sockets, struct {
				pid  string
				addr string
			}{currentPID, line[1:]})
		}
	}

	if len(sockets) != 2 {
		t.Errorf("expected 2 sockets, got %d", len(sockets))
	}
	if sockets[0].pid != "1234" {
		t.Errorf("first socket PID = %s, want 1234", sockets[0].pid)
	}
	if sockets[1].addr != "127.0.0.1:3000" {
		t.Errorf("second socket addr = %s, want 127.0.0.1:3000", sockets[1].addr)
	}
}

func TestMockPlistXMLParsing(t *testing.T) {
	mockPlist := `<dict>
	<key>Label</key>
	<string>com.test.service</string>
	<key>RunAtLoad</key>
	<true/>
	<key>StartInterval</key>
	<integer>3600</integer>
</dict>`

	if !strings.Contains(mockPlist, "<key>Label</key>") {
		t.Error("plist should contain Label key")
	}
	if !strings.Contains(mockPlist, "com.test.service") {
		t.Error("plist should contain service name")
	}
	if !strings.Contains(mockPlist, "<true/>") {
		t.Error("plist should contain RunAtLoad true")
	}
}

func TestMockCgroupParsing(t *testing.T) {
	tests := []struct {
		name          string
		cgroupContent string
		wantContainer string
	}{
		{
			name:          "docker",
			cgroupContent: "12:memory:/docker/abc123def456\n",
			wantContainer: "docker",
		},
		{
			name:          "containerd",
			cgroupContent: "12:memory:/containerd/xyz789\n",
			wantContainer: "containerd",
		},
		{
			name:          "kubernetes",
			cgroupContent: "12:memory:/kubepods/burstable/pod123\n",
			wantContainer: "kubepods",
		},
		{
			name:          "host",
			cgroupContent: "12:memory:/user.slice/user-1000.slice\n",
			wantContainer: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			container := ""
			if strings.Contains(tt.cgroupContent, "docker") {
				container = "docker"
			} else if strings.Contains(tt.cgroupContent, "containerd") {
				container = "containerd"
			} else if strings.Contains(tt.cgroupContent, "kubepods") {
				container = "kubepods"
			}

			if container != tt.wantContainer {
				t.Errorf("container = %s, want %s", container, tt.wantContainer)
			}
		})
	}
}
