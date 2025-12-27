//go:build linux

package source

import (
	"os"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func detectContainer(ancestry []model.Process) *model.Source {
	for _, p := range ancestry {
		data, err := os.ReadFile("/proc/" + itoa(p.PID) + "/cgroup")
		if err != nil {
			continue
		}
		content := string(data)

		if strings.Contains(content, "docker") ||
			strings.Contains(content, "containerd") ||
			strings.Contains(content, "kubepods") {

			return &model.Source{
				Type:       model.SourceContainer,
				Name:       "container",
				Confidence: 0.9,
			}
		}
	}
	return nil
}

func itoa(n int) string {
	return strconv.Itoa(n)
}
