//go:build windows

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectBsdRc(_ []model.Process) *model.Source {
	// windows doesn't use FreeBSD rc.d
	return nil
}
