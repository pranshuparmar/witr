//go:build linux || darwin || freebsd || windows

package main

import (
	"github.com/pranshuparmar/witr/internal/app"
)

var (
	version   = ""
	commit    = ""
	buildDate = ""
)

// go build -ldflags "-X main.version=v0.1.0 -X main.commit=$(git rev-parse --short HEAD) -X 'main.buildDate=$(date +%Y-%m-%d)'" -o witr ./cmd/witr

func main() {
	app.SetVersionBuildCommitString(version, commit, buildDate)
	app.Execute()
}
