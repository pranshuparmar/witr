//go:build !windows

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectWindowsService(ancestry []model.Process) *model.Source {
	return nil
}
