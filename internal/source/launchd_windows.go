//go:build windows

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectLaunchd(ancestry []model.Process) *model.Source {
	return nil
}
