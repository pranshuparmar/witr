//go:build freebsd

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectLaunchd(_ []model.Process) *model.Source {
	// FreeBSD doesn't use launchd
	return nil
}
