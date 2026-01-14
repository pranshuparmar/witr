//go:build darwin && (!cgo || internal_witr_cgo_disabled)

package proc

import "github.com/pranshuparmar/witr/pkg/model"

func readDarwinIO(pid int) (model.IOStats, error) {
	return model.IOStats{}, nil
}

func readDarwinTaskInfo(pid int) (model.MemoryInfo, int, error) {
	return model.MemoryInfo{}, 0, nil
}

func readDarwinFDs(pid int) (int, []string, error) {
	return 0, nil, nil
}
