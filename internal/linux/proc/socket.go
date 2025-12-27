//go:build linux

package proc

type Socket struct {
	Inode   string
	Port    int
	Address string // 0.0.0.0, 127.0.0.1, ::
}
