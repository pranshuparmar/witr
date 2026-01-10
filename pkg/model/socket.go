package model

type Socket struct {
	Inode   string
	Port    int
	Address string // 0.0.0.0, 127.0.0.1, ::
}

// SocketInfo holds information about a socket's state
type SocketInfo struct {
	Port        int
	State       string // LISTEN, TIME_WAIT, CLOSE_WAIT, ESTABLISHED, etc.
	LocalAddr   string
	RemoteAddr  string
	Explanation string // Human-readable explanation of the state
	Workaround  string // Suggested workaround if applicable
}
