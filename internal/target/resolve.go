package target

import (
	"fmt"
	"strconv"

	"github.com/pranshuparmar/witr/internal/platform"
	"github.com/pranshuparmar/witr/pkg/model"
)

func Resolve(t model.Target) ([]int, error) {
	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(t.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid pid")
		}
		return []int{pid}, nil

	case model.TargetPort:
		port, err := strconv.Atoi(t.Value)
		if err != nil {
			return nil, fmt.Errorf("invalid port")
		}
		return platform.Current.ResolvePort(port)

	case model.TargetName:
		return platform.Current.ResolveName(t.Value)

	default:
		return nil, fmt.Errorf("unknown target")
	}
}
