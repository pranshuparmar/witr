//go:build darwin

package source

import "github.com/pranshuparmar/witr/pkg/model"

func detectBsdRc(_ []model.Process) *model.Source {
	// macOS doesn't use FreeBSD rc.d
	return nil
}
