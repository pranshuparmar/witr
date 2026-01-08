package target

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

func Resolve(t model.Target) ([]int, error) {
	val := strings.TrimSpace(t.Value)

	switch t.Type {
	case model.TargetPID:
		pid, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid pid")
		}
		return []int{pid}, nil

	case model.TargetPort:
		port, err := strconv.Atoi(val)
		if err != nil {
			return nil, fmt.Errorf("invalid port")
		}
		return ResolvePort(port)

	case model.TargetName:
		return ResolveName(val)

	default:
		return nil, fmt.Errorf("unknown target")
	}
}
