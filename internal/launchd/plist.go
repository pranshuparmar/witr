//go:build darwin

package launchd

import (
	"encoding/xml"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pranshuparmar/witr/internal/proc"
)

// LaunchdInfo contains parsed information about a launchd service
type LaunchdInfo struct {
	Label     string
	PlistPath string
	Domain    string // user, system, or gui/<uid>

	// Triggers
	RunAtLoad             bool
	KeepAlive             bool
	StartInterval         int    // seconds
	StartCalendarInterval string // human-readable schedule
	WatchPaths            []string
	QueueDirectories      []string

	// Program info
	Program          string
	ProgramArguments []string
}

// plistDict represents a plist dictionary for XML parsing
type plistDict struct {
	Keys   []string
	Values []plistValue
}

type plistValue struct {
	String  string
	Integer int
	Bool    *bool
	Array   []string
	Dict    *plistDict
}

// plist search paths in order of precedence
var plistSearchPaths = []string{
	"~/Library/LaunchAgents",
	"/Library/LaunchAgents",
	"/Library/LaunchDaemons",
	"/System/Library/LaunchAgents",
	"/System/Library/LaunchDaemons",
}

// GetServiceLabel uses launchctl blame to get the service label for a PID
func GetServiceLabel(pid int) (string, string, error) {
	// launchctl blame <pid> returns the service that started the process
	out, err := proc.Run("launchctl", "blame", strconv.Itoa(pid))
	if err != nil {
		return "", "", fmt.Errorf("launchctl blame failed: %w", err)
	}

	// Output format varies:
	// - "system/com.apple.example" or "gui/501/com.example.app" (real service)
	// - "speculative", "non-ipc demand", "launch job demand", "ipc (mach)" (blame reasons)
	line := strings.TrimSpace(string(out))
	if line == "" {
		return "", "", fmt.Errorf("no service label found for pid %d", pid)
	}

	// Check if this is a real service path (contains "/" and starts with domain)
	if !strings.Contains(line, "/") {
		// This is a blame reason, not a service label
		// Try to find the service by querying launchctl list
		label, domain := findServiceByPID(pid)
		if label != "" {
			return label, domain, nil
		}
		return "", "", fmt.Errorf("process not managed by a named launchd service: %s", line)
	}

	// Parse domain and label from service path
	parts := strings.SplitN(line, "/", 2)
	if len(parts) < 2 {
		return line, "", nil
	}

	domain := parts[0]
	label := parts[1]

	// Handle gui/501/label format
	if domain == "gui" {
		subParts := strings.SplitN(label, "/", 2)
		if len(subParts) == 2 {
			domain = "gui/" + subParts[0]
			label = subParts[1]
		}
	}

	return label, domain, nil
}

// findServiceByPID queries launchctl list to find a service matching the given PID
func findServiceByPID(pid int) (string, string) {
	// launchctl list shows: PID Status Label
	out, err := proc.Run("launchctl", "list")
	if err != nil {
		return "", ""
	}

	pidStr := strconv.Itoa(pid)
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 3 && fields[0] == pidStr {
			label := fields[2]
			// Determine domain based on label prefix
			domain := "user"
			if strings.HasPrefix(label, "com.apple.") {
				domain = "system"
			}
			return label, domain
		}
	}

	return "", ""
}

// FindPlistPath searches for the plist file for a given service label
func FindPlistPath(label string) string {
	homeDir, _ := os.UserHomeDir()

	for _, searchPath := range plistSearchPaths {
		path := searchPath
		if strings.HasPrefix(path, "~") {
			path = filepath.Join(homeDir, path[1:])
		}

		plistPath := filepath.Join(path, label+".plist")
		if _, err := os.Stat(plistPath); err == nil {
			return plistPath
		}
	}

	return ""
}

// ParsePlist reads and parses a launchd plist file
func ParsePlist(path string) (*LaunchdInfo, error) {
	// Use plutil to convert to XML (handles binary plists)
	out, err := proc.Run("plutil", "-convert", "xml1", "-o", "-", path)
	if err != nil {
		return nil, fmt.Errorf("failed to convert plist: %w", err)
	}

	info := &LaunchdInfo{
		PlistPath: path,
	}

	// Parse the XML plist
	if err := parsePlistXML(out, info); err != nil {
		return nil, err
	}

	return info, nil
}

// parsePlistXML parses XML plist data into LaunchdInfo
func parsePlistXML(data []byte, info *LaunchdInfo) error {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))

	var currentKey string
	var dictDepth int // Track dict nesting depth (1 = root dict)

	for {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			switch t.Name.Local {
			case "dict":
				dictDepth++
				// Skip nested dicts by clearing currentKey
				if dictDepth > 1 {
					currentKey = ""
				}
			case "key":
				// Only capture keys at root dict level
				if dictDepth == 1 {
					var key string
					decoder.DecodeElement(&key, &t)
					currentKey = key
				}
			case "string":
				if dictDepth == 1 && currentKey != "" {
					var val string
					decoder.DecodeElement(&val, &t)
					handleStringValue(info, currentKey, val)
					currentKey = ""
				}
			case "integer":
				if dictDepth == 1 && currentKey != "" {
					var val string
					decoder.DecodeElement(&val, &t)
					if i, err := strconv.Atoi(val); err == nil {
						handleIntValue(info, currentKey, i)
					}
					currentKey = ""
				}
			case "true":
				if dictDepth == 1 && currentKey != "" {
					handleBoolValue(info, currentKey, true)
					currentKey = ""
				}
			case "false":
				if dictDepth == 1 && currentKey != "" {
					handleBoolValue(info, currentKey, false)
					currentKey = ""
				}
			case "array":
				if dictDepth == 1 && currentKey != "" {
					arr := parseArray(decoder)
					handleArrayValue(info, currentKey, arr)
					currentKey = ""
				}
			}
		case xml.EndElement:
			if t.Name.Local == "dict" {
				dictDepth--
			}
		}
	}

	return nil
}

func parseArray(decoder *xml.Decoder) []string {
	var result []string
	depth := 1

	for depth > 0 {
		token, err := decoder.Token()
		if err != nil {
			break
		}

		switch t := token.(type) {
		case xml.StartElement:
			if t.Name.Local == "array" {
				depth++
			} else if t.Name.Local == "string" {
				var val string
				decoder.DecodeElement(&val, &t)
				result = append(result, val)
			}
		case xml.EndElement:
			if t.Name.Local == "array" {
				depth--
			}
		}
	}

	return result
}

func handleStringValue(info *LaunchdInfo, key, val string) {
	switch key {
	case "Label":
		info.Label = val
	case "Program":
		info.Program = val
	}
}

func handleIntValue(info *LaunchdInfo, key string, val int) {
	switch key {
	case "StartInterval":
		info.StartInterval = val
	}
}

func handleBoolValue(info *LaunchdInfo, key string, val bool) {
	switch key {
	case "RunAtLoad":
		info.RunAtLoad = val
	case "KeepAlive":
		info.KeepAlive = val
	}
}

func handleArrayValue(info *LaunchdInfo, key string, val []string) {
	switch key {
	case "ProgramArguments":
		info.ProgramArguments = val
	case "WatchPaths":
		info.WatchPaths = val
	case "QueueDirectories":
		info.QueueDirectories = val
	}
}

// GetLaunchdInfo retrieves full launchd information for a process
func GetLaunchdInfo(pid int) (*LaunchdInfo, error) {
	label, domain, err := GetServiceLabel(pid)
	if err != nil {
		return nil, err
	}

	plistPath := FindPlistPath(label)
	if plistPath == "" {
		// Return basic info even if we can't find the plist
		return &LaunchdInfo{
			Label:  label,
			Domain: domain,
		}, nil
	}

	info, err := ParsePlist(plistPath)
	if err != nil {
		// Return basic info on parse error
		return &LaunchdInfo{
			Label:     label,
			Domain:    domain,
			PlistPath: plistPath,
		}, nil
	}

	info.Domain = domain
	return info, nil
}

// FormatTriggers returns a human-readable description of what triggers the service
func (info *LaunchdInfo) FormatTriggers() []string {
	var triggers []string

	if info.RunAtLoad {
		triggers = append(triggers, "RunAtLoad (starts at login/boot)")
	}

	if info.StartInterval > 0 {
		triggers = append(triggers, fmt.Sprintf("StartInterval (every %s)", formatDuration(info.StartInterval)))
	}

	if info.StartCalendarInterval != "" {
		triggers = append(triggers, fmt.Sprintf("StartCalendarInterval (%s)", info.StartCalendarInterval))
	}

	if len(info.WatchPaths) > 0 {
		for _, p := range info.WatchPaths {
			triggers = append(triggers, fmt.Sprintf("WatchPaths: %s", p))
		}
	}

	if len(info.QueueDirectories) > 0 {
		for _, p := range info.QueueDirectories {
			triggers = append(triggers, fmt.Sprintf("QueueDirectories: %s", p))
		}
	}

	return triggers
}

func formatDuration(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%ds", seconds)
	}
	if seconds < 3600 {
		return fmt.Sprintf("%dm", seconds/60)
	}
	if seconds < 86400 {
		return fmt.Sprintf("%dh", seconds/3600)
	}
	return fmt.Sprintf("%dd", seconds/86400)
}

// DomainDescription returns a human-readable description of the domain
func (info *LaunchdInfo) DomainDescription() string {
	switch {
	case info.Domain == "system":
		return "Launch Daemon"
	case strings.HasPrefix(info.Domain, "gui/"):
		return "Launch Agent"
	case info.Domain == "user":
		return "Launch Agent"
	default:
		return "launchd service"
	}
}
