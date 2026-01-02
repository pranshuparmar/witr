//go:build darwin

package source

import "github.com/SanCognition/witr/pkg/model"

func detectSystemd(_ []model.Process) *model.Source {
	return nil
}
