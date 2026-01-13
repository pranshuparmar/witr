//go:build windows

package proc

import (
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestGetListeningPortsForPIDWindows(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	netstatOut := `
  TCP    0.0.0.0:8080           0.0.0.0:0              LISTENING       1234
  TCP    [::]:9090              [::]:0                 LISTENING       1234
  TCP    127.0.0.1:443          0.0.0.0:0              ESTABLISHED     1234
  TCP    0.0.0.0:22             0.0.0.0:0              LISTENING       9999
`

	mockExec.EXPECT().Run("netstat", "-ano").Return([]byte(netstatOut), nil)

	ports, addrs := GetListeningPortsForPID(1234)

	if len(ports) != 2 {
		t.Fatalf("Got %d ports, want 2", len(ports))
	}

	// Order isn't guaranteed, but map makes it likely stable if inserted in order
	// Check for 8080 and 9090
	has8080 := false
	has9090 := false
	for _, p := range ports {
		if p == 8080 {
			has8080 = true
		}
		if p == 9090 {
			has9090 = true
		}
	}
	if !has8080 || !has9090 {
		t.Errorf("Ports = %v, want [8080, 9090]", ports)
	}

	if len(addrs) != 2 {
		t.Errorf("Got %d addrs, want 2", len(addrs))
	}
}
