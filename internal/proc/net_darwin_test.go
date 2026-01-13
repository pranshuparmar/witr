//go:build darwin

package proc

import (
	"errors"
	"testing"

	"github.com/pranshuparmar/witr/internal/proc/mocks"
	"go.uber.org/mock/gomock"
)

func TestReadListeningSockets(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	// lsof output format:
	// p<pid>
	// n<address>
	lsofOut := "p123\nn*:8080\np456\nn127.0.0.1:9090\n"

	mockExec.EXPECT().Run("lsof", "-i", "TCP", "-s", "TCP:LISTEN", "-n", "-P", "-F", "pn").
		Return([]byte(lsofOut), nil)

	sockets, err := readListeningSockets()
	if err != nil {
		t.Fatalf("readListeningSockets failed: %v", err)
	}

	if len(sockets) != 2 {
		t.Errorf("Got %d sockets, want 2", len(sockets))
	}

	// Check socket 1
	inode1 := "123:8080"
	if s, ok := sockets[inode1]; !ok {
		t.Errorf("Socket %s not found", inode1)
	} else {
		if s.Port != 8080 {
			t.Errorf("Socket 1 port = %d, want 8080", s.Port)
		}
		if s.Address != "0.0.0.0" {
			t.Errorf("Socket 1 address = %q, want 0.0.0.0", s.Address)
		}
	}

	// Check socket 2
	inode2 := "456:9090"
	if s, ok := sockets[inode2]; !ok {
		t.Errorf("Socket %s not found", inode2)
	} else {
		if s.Port != 9090 {
			t.Errorf("Socket 2 port = %d, want 9090", s.Port)
		}
		if s.Address != "127.0.0.1" {
			t.Errorf("Socket 2 address = %q, want 127.0.0.1", s.Address)
		}
	}
}

func TestReadListeningSocketsFallbackToNetstat(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	mockExec := mocks.NewMockExecutor(ctrl)
	SetExecutor(mockExec)
	defer ResetExecutor()

	// lsof fails
	mockExec.EXPECT().Run("lsof", gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, errors.New("lsof failed"))

	// netstat fallback
	// Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)
	// tcp4       0      0  127.0.0.1.3306         *.*                    LISTEN
	netstatOut := "Proto Recv-Q Send-Q  Local Address          Foreign Address        (state)\n" +
		"tcp4       0      0  127.0.0.1.3306         *.*                    LISTEN\n"

	mockExec.EXPECT().Run("netstat", "-an", "-p", "tcp").
		Return([]byte(netstatOut), nil)

	sockets, err := readListeningSockets()
	if err != nil {
		t.Fatalf("readListeningSockets failed: %v", err)
	}

	if len(sockets) != 1 {
		t.Errorf("Got %d sockets, want 1", len(sockets))
	}

	// Check socket
	// For netstat, inode is "netstat:" + localAddr
	inode := "netstat:127.0.0.1.3306"
	if s, ok := sockets[inode]; !ok {
		t.Errorf("Socket %s not found", inode)
	} else {
		if s.Port != 3306 {
			t.Errorf("Socket port = %d, want 3306", s.Port)
		}
		if s.Address != "127.0.0.1" {
			t.Errorf("Socket address = %q, want 127.0.0.1", s.Address)
		}
	}
}
