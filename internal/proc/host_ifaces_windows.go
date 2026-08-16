//go:build windows

package proc

import (
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/pranshuparmar/witr/pkg/model"
)

// listHostInterfaces prefers `ipconfig /all` (what users expect on Windows),
// then falls back to the Go net package.
func listHostInterfaces() []model.HostInterface {
	if ifaces, src := listHostInterfacesIPConfig(); len(ifaces) > 0 {
		_ = src
		return ifaces
	}
	return listHostInterfacesStdlib()
}

func listHostInterfacesWithSource() ([]model.HostInterface, string) {
	if ifaces, src := listHostInterfacesIPConfig(); len(ifaces) > 0 {
		return ifaces, src
	}
	return listHostInterfacesStdlib(), "net"
}

func listVethMap() map[string]vethInfo {
	return map[string]vethInfo{}
}

func listHostInterfacesIPConfig() ([]model.HostInterface, string) {
	out, err := exec.Command("ipconfig", "/all").Output()
	if err != nil || len(out) == 0 {
		// Some locales ship ipconfig; try without /all.
		out, err = exec.Command("ipconfig").Output()
		if err != nil || len(out) == 0 {
			return nil, ""
		}
		return parseIPConfig(string(out)), "ipconfig"
	}
	return parseIPConfig(string(out)), "ipconfig /all"
}

// adapterHeader matches lines like:
//
//	Ethernet adapter Ethernet:
//	Wireless LAN adapter Wi-Fi:
//	Unknown adapter Local Area Connection* 9:
var (
	reAdapterHeader = regexp.MustCompile(`(?i)^(.+?)\s+adapter\s+(.+):\s*$`)
	reIPConfigKV    = regexp.MustCompile(`(?i)^\s*([^.:]+?)(?:\s*\.\s*)+\s*:\s*(.*)$`)
	reIPv4Pref      = regexp.MustCompile(`(?i)^(\d+\.\d+\.\d+\.\d+)`)
	reIPv6Pref      = regexp.MustCompile(`(?i)^([0-9a-f:]+)`)
)

// parseIPConfig turns ipconfig [/all] text into HostInterface rows.
func parseIPConfig(text string) []model.HostInterface {
	// ipconfig may be UTF-16 on some systems; strip BOM and normalize.
	text = strings.TrimPrefix(text, "\ufeff")
	// If we got UTF-16 LE misread as a string, newlines may be sparse — still try.

	var (
		result  []model.HostInterface
		cur     *model.HostInterface
		pending string // last key for multi-line values (DNS, etc.)
	)

	flush := func() {
		if cur == nil {
			return
		}
		if cur.State == "" {
			// Media disconnected is often the only signal.
			if len(cur.IPv4) == 0 && len(cur.IPv6) == 0 {
				cur.State = "DOWN"
			} else {
				cur.State = "UP"
			}
		}
		if cur.Type == "" {
			cur.Type = detectInterfaceType(cur.Name)
		}
		if cur.MAC == "" {
			cur.MAC = "N/A"
		}
		result = append(result, *cur)
		cur = nil
		pending = ""
	}

	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}

		if m := reAdapterHeader.FindStringSubmatch(line); m != nil {
			flush()
			kind := strings.TrimSpace(m[1])
			name := strings.TrimSpace(m[2])
			cur = &model.HostInterface{
				Name: name,
				Type: classifyWindowsAdapter(kind, name),
			}
			pending = ""
			continue
		}

		// Skip the global "Windows IP Configuration" preamble.
		if cur == nil {
			continue
		}

		// Prefer key/value lines (most adapter properties are indented AND
		// contain "Key . . . : value"). Only treat a line as a multi-value
		// continuation (extra DNS server, etc.) when it is indented and does
		// NOT look like a property row.
		if kv := reIPConfigKV.FindStringSubmatch(line); kv != nil {
			key := normalizeIPConfigKey(kv[1])
			val := strings.TrimSpace(kv[2])
			pending = key
			applyIPConfigValue(cur, key, val)
			continue
		}

		// Fallback for irregular "Key....: value" spacing.
		if key, val, ok := splitIPConfigFallback(line); ok {
			pending = key
			applyIPConfigValue(cur, key, val)
			continue
		}

		// Continuation value for the previous key (e.g. second DNS server).
		if pending != "" && (strings.HasPrefix(line, "   ") || strings.HasPrefix(line, "\t")) {
			val := strings.TrimSpace(line)
			if val == "" {
				continue
			}
			applyIPConfigValue(cur, pending, val)
		}
	}
	flush()

	sort.Slice(result, func(i, j int) bool {
		return result[i].Name < result[j].Name
	})
	return result
}

func normalizeIPConfigKey(k string) string {
	k = strings.ToLower(strings.TrimSpace(k))
	k = strings.Join(strings.Fields(k), " ")
	return k
}

// splitIPConfigFallback handles property lines the main regex misses.
// Uses the first " . :" / ".: " style separator, not the last colon (IPv6).
func splitIPConfigFallback(line string) (key, val string, ok bool) {
	// Find " : " after a run of dots.
	idx := strings.Index(line, " : ")
	if idx < 0 {
		// "....:value" without spaces around colon
		idx = strings.Index(line, ".:")
		if idx < 0 {
			return "", "", false
		}
		// include the dot before colon in the key side
		key = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line[:idx+1]), ". "))
		val = strings.TrimSpace(line[idx+2:])
		if key == "" {
			return "", "", false
		}
		return normalizeIPConfigKey(key), val, true
	}
	key = strings.TrimSpace(strings.TrimRight(strings.TrimSpace(line[:idx]), ". "))
	val = strings.TrimSpace(line[idx+3:])
	if key == "" {
		return "", "", false
	}
	return normalizeIPConfigKey(key), val, true
}

func applyIPConfigValue(cur *model.HostInterface, key, val string) {
	if cur == nil || val == "" || strings.EqualFold(val, "none") {
		// Still record media state from empty gateway? skip empties.
		if val == "" {
			return
		}
	}
	switch {
	case strings.Contains(key, "description"):
		cur.Description = val
	case strings.Contains(key, "physical address"):
		cur.MAC = normalizeMAC(val)
	case strings.Contains(key, "dhcp enabled"):
		cur.DHCP = val
	case strings.Contains(key, "media state"):
		// "Media disconnected"
		if strings.Contains(strings.ToLower(val), "disconnect") {
			cur.State = "DOWN"
		} else {
			cur.State = strings.ToUpper(val)
		}
	case strings.Contains(key, "ipv4 address") || key == "ip address":
		if ip := stripPreferred(val); ip != "" {
			cur.IPv4 = appendUnique(cur.IPv4, ip)
			if cur.State == "" {
				cur.State = "UP"
			}
		}
	case strings.Contains(key, "subnet mask"):
		// Attach mask to the last IPv4 if it has no prefix yet.
		if len(cur.IPv4) > 0 && !strings.Contains(cur.IPv4[len(cur.IPv4)-1], "/") {
			if pfx := subnetMaskToPrefix(val); pfx != "" {
				cur.IPv4[len(cur.IPv4)-1] = cur.IPv4[len(cur.IPv4)-1] + "/" + pfx
			}
		}
	case strings.Contains(key, "ipv6 address") || strings.Contains(key, "temporary ipv6") || strings.Contains(key, "link-local ipv6"):
		if strings.Contains(key, "link-local") {
			return // skip fe80 like Linux path
		}
		if ip := stripPreferred(val); ip != "" {
			cur.IPv6 = appendUnique(cur.IPv6, ip)
			if cur.State == "" {
				cur.State = "UP"
			}
		}
	case strings.Contains(key, "default gateway"):
		if g := strings.TrimSpace(val); g != "" && !strings.EqualFold(g, "none") {
			cur.Gateway = appendUnique(cur.Gateway, g)
		}
	case strings.Contains(key, "dns servers"):
		if d := strings.TrimSpace(val); d != "" {
			cur.DNS = appendUnique(cur.DNS, d)
		}
	case strings.Contains(key, "autconfiguration") || strings.Contains(key, "autoconfiguration"):
		// ignore
	}
}

func stripPreferred(val string) string {
	val = strings.TrimSpace(val)
	// "192.168.1.10(Preferred)" or with spaces
	if i := strings.Index(val, "("); i > 0 {
		val = strings.TrimSpace(val[:i])
	}
	if m := reIPv4Pref.FindStringSubmatch(val); m != nil {
		return m[1]
	}
	if m := reIPv6Pref.FindStringSubmatch(val); m != nil {
		return m[1]
	}
	return val
}

func normalizeMAC(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "-", ":")
	return strings.ToLower(s)
}

func appendUnique(slice []string, v string) []string {
	for _, x := range slice {
		if x == v {
			return slice
		}
	}
	return append(slice, v)
}

func subnetMaskToPrefix(mask string) string {
	parts := strings.Split(strings.TrimSpace(mask), ".")
	if len(parts) != 4 {
		return ""
	}
	var total int
	for _, p := range parts {
		var n int
		for _, c := range p {
			if c < '0' || c > '9' {
				return ""
			}
			n = n*10 + int(c-'0')
		}
		if n > 255 {
			return ""
		}
		for n > 0 {
			total += n & 1
			n >>= 1
		}
	}
	if total == 0 {
		return ""
	}
	return itoa(total)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [12]byte
	i := len(b)
	neg := n < 0
	if neg {
		n = -n
	}
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func classifyWindowsAdapter(kind, name string) string {
	k := strings.ToLower(kind)
	n := strings.ToLower(name)
	switch {
	case strings.Contains(k, "wireless") || strings.Contains(n, "wi-fi") || strings.Contains(n, "wifi"):
		return "wireless"
	case strings.Contains(k, "ethernet") || strings.Contains(n, "ethernet"):
		return "physical"
	case strings.Contains(n, "bluetooth"):
		return "bluetooth"
	case strings.Contains(n, "vethernet") || strings.Contains(n, "hyper-v"):
		return "hyper-v"
	case strings.Contains(n, "wsl"):
		return "wsl"
	case strings.Contains(n, "loopback"):
		return "loopback"
	case strings.Contains(k, "tunnel") || strings.Contains(n, "tunnel"):
		return "tunnel"
	case strings.Contains(k, "unknown"):
		return "virtual"
	default:
		return detectInterfaceType(name)
	}
}

func detectInterfaceType(name string) string {
	lower := strings.ToLower(name)
	switch {
	case lower == "lo" || strings.Contains(lower, "loopback") || strings.HasPrefix(lower, "lo"):
		return "loopback"
	case strings.Contains(lower, "bluetooth"):
		return "bluetooth"
	case strings.HasPrefix(lower, "vethernet") || strings.Contains(lower, "vethernet") || strings.Contains(lower, "hyper-v"):
		return "hyper-v"
	case strings.Contains(lower, "wsl"):
		return "wsl"
	case strings.HasPrefix(lower, "eth") || strings.HasPrefix(lower, "en") || strings.Contains(lower, "ethernet"):
		return "physical"
	case strings.HasPrefix(lower, "wl") || strings.Contains(lower, "wi-fi") || strings.Contains(lower, "wifi") || strings.Contains(lower, "wlan"):
		return "wireless"
	case strings.HasPrefix(lower, "veth"):
		return "veth"
	case strings.Contains(lower, "docker") || strings.HasPrefix(lower, "br-"):
		return "bridge"
	case strings.HasPrefix(lower, "wg"):
		return "wireguard"
	case strings.HasPrefix(lower, "tun") || strings.HasPrefix(lower, "tap"):
		return "tun/tap"
	case strings.Contains(lower, "local area connection"):
		return "virtual"
	default:
		return "unknown"
	}
}
