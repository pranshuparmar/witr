//go:build darwin && cgo && !internal_witr_cgo_disabled

package proc

/*
#cgo CFLAGS: -mmacosx-version-min=11.0
#cgo LDFLAGS: -mmacosx-version-min=11.0
#include <libproc.h>
#include <sys/resource.h>
#include <sys/proc_info.h>
#include <sys/socket.h>
#include <arpa/inet.h>
#include <errno.h>
#include <string.h>
#include <stdio.h>

static int witr_proc_pid_rusage(int pid, int flavor, struct rusage_info_v4 *usage) {
    int rv = proc_pid_rusage(pid, flavor, (rusage_info_t)usage);
    if (rv != 0) {
        return errno;
    }
    return 0;
}

static int witr_proc_pidtaskinfo(int pid, struct proc_taskinfo *info) {
    int rv = proc_pidinfo(pid, PROC_PIDTASKINFO, 0, info, PROC_PIDTASKINFO_SIZE);
    if (rv < 0) {
        return errno;
    }
    if (rv < PROC_PIDTASKINFO_SIZE) {
        return EIO;
    }
    return 0;
}

static int witr_proc_pidlistfds(int pid, struct proc_fdinfo *fds, int bufsize, int *bytes_used) {
    int rv = proc_pidinfo(pid, PROC_PIDLISTFDS, 0, fds, bufsize);
    if (rv < 0) {
        if (bytes_used) {
            *bytes_used = 0;
        }
        return errno;
    }
    if (bytes_used) {
        *bytes_used = rv;
    }
    return 0;
}

static int witr_proc_pidfdinfo_vnode(int pid, int fd, struct vnode_fdinfowithpath *info) {
    int rv = proc_pidfdinfo(pid, fd, PROC_PIDFDVNODEPATHINFO, info, PROC_PIDFDVNODEPATHINFO_SIZE);
    if (rv < 0) {
        return errno;
    }
    if (rv < PROC_PIDFDVNODEPATHINFO_SIZE) {
        return EIO;
    }
    return 0;
}

static int witr_proc_pidfdinfo_socket(int pid, int fd, struct socket_fdinfo *info) {
    int rv = proc_pidfdinfo(pid, fd, PROC_PIDFDSOCKETINFO, info, PROC_PIDFDSOCKETINFO_SIZE);
    if (rv < 0) {
        return errno;
    }
    if (rv < PROC_PIDFDSOCKETINFO_SIZE) {
        return EIO;
    }
    return 0;
}

static void witr_format_socket(const struct socket_fdinfo *info, char *buf, size_t buf_len) {
    if (buf_len == 0) {
        return;
    }
    buf[0] = '\0';
    if (!info) {
        return;
    }

    const struct in_sockinfo *ini = &info->psi.soi_proto.pri_tcp.tcpsi_ini;
    char laddr[INET6_ADDRSTRLEN];
    char faddr[INET6_ADDRSTRLEN];
    laddr[0] = '\0';
    faddr[0] = '\0';
    uint16_t lport = ntohs((uint16_t)ini->insi_lport);
    uint16_t fport = ntohs((uint16_t)ini->insi_fport);

    if (ini->insi_vflag & INI_IPV4) {
        inet_ntop(AF_INET, &ini->insi_laddr.ina_46.i46a_addr4, laddr, sizeof(laddr));
        inet_ntop(AF_INET, &ini->insi_faddr.ina_46.i46a_addr4, faddr, sizeof(faddr));
    } else if (ini->insi_vflag & INI_IPV6) {
        inet_ntop(AF_INET6, &ini->insi_laddr.ina_6, laddr, sizeof(laddr));
        inet_ntop(AF_INET6, &ini->insi_faddr.ina_6, faddr, sizeof(faddr));
    }

    if (laddr[0] == '\0') {
        strlcpy(laddr, "?", sizeof(laddr));
    }
    if (faddr[0] == '\0') {
        strlcpy(faddr, "?", sizeof(faddr));
    }

    snprintf(buf, buf_len, "%s:%u -> %s:%u", laddr, lport, faddr, fport);
}
*/
import "C"

import (
	"errors"
	"fmt"
	"unsafe"

	"github.com/pranshuparmar/witr/pkg/model"
)

func readDarwinIO(pid int) (model.IOStats, error) {
	var stats model.IOStats
	var usage C.struct_rusage_info_v4
	errno := C.witr_proc_pid_rusage(C.int(pid), C.RUSAGE_INFO_V4, &usage)
	if errno != 0 {
		switch errno {
		case C.ESRCH, C.EPERM:
			return stats, nil
		default:
			return stats, fmt.Errorf("proc_pid_rusage: %d", errno)
		}
	}

	stats.ReadBytes = uint64(usage.ri_diskio_bytesread)
	stats.WriteBytes = uint64(usage.ri_diskio_byteswritten)
	return stats, nil
}

func readDarwinTaskInfo(pid int) (model.MemoryInfo, int, error) {
	var info C.struct_proc_taskinfo
	if errno := C.witr_proc_pidtaskinfo(C.int(pid), &info); errno != 0 {
		switch errno {
		case C.ESRCH, C.EPERM:
			return model.MemoryInfo{}, 0, nil
		default:
			return model.MemoryInfo{}, 0, fmt.Errorf("proc_pidinfo taskinfo: %d", errno)
		}
	}

	mem := model.MemoryInfo{
		VMS:   uint64(info.pti_virtual_size),
		RSS:   uint64(info.pti_resident_size),
		VMSMB: float64(info.pti_virtual_size) / (1024 * 1024),
		RSSMB: float64(info.pti_resident_size) / (1024 * 1024),
	}

	return mem, int(info.pti_threadnum), nil
}

func readDarwinFDs(pid int) (int, []string, error) {
	const bytesPerEntry = int(C.sizeof_struct_proc_fdinfo)
	entries := make([]C.struct_proc_fdinfo, 256)

	for {
		var used C.int
		errno := C.witr_proc_pidlistfds(C.int(pid), &entries[0], C.int(len(entries)*bytesPerEntry), &used)
		if errno == C.EINVAL && len(entries) < 16384 {
			entries = make([]C.struct_proc_fdinfo, len(entries)*2)
			continue
		}
		if errno != 0 {
			switch errno {
			case C.ESRCH, C.EPERM:
				return 0, nil, nil
			default:
				return 0, nil, fmt.Errorf("proc_pidinfo listfds: %d", errno)
			}
		}
		bytesUsed := int(used)
		if bytesUsed%bytesPerEntry != 0 {
			return 0, nil, errors.New("listfds returned partial record")
		}
		count := bytesUsed / bytesPerEntry
		return count, formatFDEntries(pid, entries[:count]), nil
	}
}

func formatFDEntries(pid int, entries []C.struct_proc_fdinfo) []string {
	var out []string
	for _, entry := range entries {
		if len(out) >= 10 {
			break
		}
		fd := int(entry.proc_fd)
		label := fdTypeLabel(entry.proc_fdtype)
		switch entry.proc_fdtype {
		case C.PROX_FDTYPE_VNODE:
			var vnode C.struct_vnode_fdinfowithpath
			if errno := C.witr_proc_pidfdinfo_vnode(C.int(pid), C.int(fd), &vnode); errno == 0 {
				path := C.GoString(&vnode.pvip.vip_path[0])
				if path == "" {
					path = "<anonymous>"
				}
				out = append(out, fmt.Sprintf("%d -> %s", fd, path))
				continue
			}
		case C.PROX_FDTYPE_SOCKET:
			var sock C.struct_socket_fdinfo
			if errno := C.witr_proc_pidfdinfo_socket(C.int(pid), C.int(fd), &sock); errno == 0 {
				buf := make([]byte, 128)
				C.witr_format_socket(&sock, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
				desc := C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
				out = append(out, fmt.Sprintf("%d -> %s", fd, desc))
				continue
			}
		}
		out = append(out, fmt.Sprintf("%d (%s)", fd, label))
	}
	return out
}

func fdTypeLabel(fdType C.uint32_t) string {
	switch fdType {
	case C.PROX_FDTYPE_VNODE:
		return "vnode"
	case C.PROX_FDTYPE_SOCKET:
		return "socket"
	case C.PROX_FDTYPE_PIPE:
		return "pipe"
	case C.PROX_FDTYPE_KQUEUE:
		return "kqueue"
	case C.PROX_FDTYPE_FSEVENTS:
		return "fsevents"
	case C.PROX_FDTYPE_NEXUS:
		return "nexus"
	case C.PROX_FDTYPE_NETPOLICY:
		return "netpolicy"
	default:
		return fmt.Sprintf("fdtype-%d", fdType)
	}
}

func describeSocket(info *C.struct_socket_fdinfo) string {
	buf := make([]byte, 128)
	C.witr_format_socket(info, (*C.char)(unsafe.Pointer(&buf[0])), C.size_t(len(buf)))
	desc := C.GoString((*C.char)(unsafe.Pointer(&buf[0])))
	if desc == "" {
		return "socket"
	}
	return desc
}
