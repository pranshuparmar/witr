//go:build linux

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectBsdRc(_ []model.Process) *model.Source {
	// Linux doesn't use FreeBSD rc.d
	return nil
}
