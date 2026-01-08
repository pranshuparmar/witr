//go:build windows

package proc

import "github.com/pranshuparmar/witr/pkg/model"

func GetFileContext(pid int) *model.FileContext {
	return nil
}
