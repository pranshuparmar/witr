package model

type SourceType string

const (
	SourceContainer      SourceType = "container"
	SourceSystemd        SourceType = "systemd"
	SourceLaunchd        SourceType = "launchd"
	SourceBsdRc          SourceType = "bsdrc"
	SourceSupervisor     SourceType = "supervisor"
	SourceCron           SourceType = "cron"
	SourceShell          SourceType = "shell"
	SourceWindowsService SourceType = "windows_service"
	SourceInit           SourceType = "init"
	SourceUnknown        SourceType = "unknown"
)

type Source struct {
	Type    SourceType
	Name    string
	Details map[string]string
}
