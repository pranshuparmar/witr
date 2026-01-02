//go:build linux

package source

import "github.com/SanCognition/witr/pkg/model"

func detectLaunchd(_ []model.Process) *model.Source {
	return nil
}
