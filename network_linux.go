//go:build linux

package main

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// --- Ping (Linux) ---

func quickPing(target string, count int, timeout time.Duration) (avgRTT float64, lossPct float64, err error) {
	if ip := net.ParseIP(target); ip == nil {
		addrs, e := net.LookupHost(target)
		if e != nil || len(addrs) == 0 {
			return 0, 100, fmt.Errorf("cannot resolve %s", target)
		}
		target = addrs[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(count))
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-c", strconv.Itoa(count), "-W", "2", target)
	out, runErr := cmd.Output()
	output := string(out)

	// Same format as macOS: "3 packets transmitted, 3 received, 0% packet loss"
	lossRe := regexp.MustCompile(`(\d+\.?\d*)% packet loss`)
	if m := lossRe.FindStringSubmatch(output); len(m) >= 2 {
		lossPct, _ = strconv.ParseFloat(m[1], 64)
	}
	// Linux: "rtt min/avg/max/mdev = 1.234/3.456/7.890/1.234 ms"
	rttRe := regexp.MustCompile(`(?:rtt |round-trip )?min/avg/max/(?:mdev|stddev) = [\d.]+/([\d.]+)/[\d.]+/[\d.]+`)
	if m := rttRe.FindStringSubmatch(output); len(m) >= 2 {
		avgRTT, _ = strconv.ParseFloat(m[1], 64)
	}

	if lossPct == 100 {
		return 0, 100, fmt.Errorf("100%% packet loss")
	}
	if runErr != nil && avgRTT == 0 {
		return 0, lossPct, runErr
	}
	return avgRTT, lossPct, nil
}

// --- Traceroute (Linux) ---

func doTrace(target string, maxHops int, timeout time.Duration) ([]traceHop, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(maxHops))
	defer cancel()

	// Try traceroute first, fallback to tracepath
	cmd := exec.CommandContext(ctx, "traceroute", "-m", strconv.Itoa(maxHops), "-q", "1", "-w", "1", target)
	out, err := cmd.Output()
	if err != nil {
		// Fallback: try tracepath (no root needed)
		cmd2 := exec.CommandContext(ctx, "tracepath", "-m", strconv.Itoa(maxHops), target)
		out2, err2 := cmd2.Output()
		if err2 != nil {
			return nil, fmt.Errorf("traceroute and tracepath both failed: %w, %w", err, err2)
		}
		return parseTracePathOutput(string(out2))
	}
	return parseTraceOutput(string(out), err)
}

func parseTracePathOutput(output string) ([]traceHop, error) {
	var hops []traceHop
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`^\s*(\d+):\s+(\S+)\s+([\d.]+)ms`)

	for _, line := range lines {
		// Skip header lines
		if strings.HasPrefix(line, "tracepath") || strings.HasPrefix(line, " 1:") == false &&
			!strings.Contains(line, "ms") {
			continue
		}
		m := re.FindStringSubmatch(line)
		if len(m) >= 4 {
			hop, _ := strconv.Atoi(m[1])
			ip := m[2]
			rtt, _ := strconv.ParseFloat(m[3], 64)
			hops = append(hops, traceHop{Hop: hop, IP: ip, RTT: rtt})
		}
	}
	return hops, nil
}

// --- WiFi (Linux - nmcli) ---

func getWifiInfo() (*wifiInfo, error) {
	// Try nmcli first (NetworkManager, most common on desktop Linux)
	if info, err := getWifiFromNmcli(); err == nil {
		return info, nil
	}
	// Fallback: iwconfig (older systems)
	return getWifiFromIwconfig()
}

func getWifiFromNmcli() (*wifiInfo, error) {
	// Get active WiFi connection
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "connection", "show", "--active")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	output := string(out)

	// Find WiFi connection
	var wifiDev string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		parts := strings.Split(line, ":")
		if len(parts) >= 3 && strings.Contains(parts[1], "wireless") {
			wifiDev = parts[2]
			break
		}
	}
	if wifiDev == "" {
		return nil, fmt.Errorf("no active WiFi connection")
	}

	// Get WiFi details
	cmd2 := exec.Command("nmcli", "-t", "-f", "SSID,BSSID,SIGNAL,FREQ,RATE", "device", "wifi", "list", "--rescan", "no")
	out2, _ := cmd2.Output()
	output2 := string(out2)

	info := &wifiInfo{}

	// nmcli wifi output: lines with "IN-USE:SSID:MODE:CHAN:RATE:SIGNAL:BARS:SECURITY"
	// or from `device wifi list`: lines start with "*:SSID:..."
	for _, line := range strings.Split(output2, "\n") {
		if !strings.HasPrefix(line, "*:") && !strings.Contains(line, ":") {
			continue
		}
		line = strings.TrimPrefix(line, "*:")
		parts := strings.Split(line, ":")
		if len(parts) >= 6 {
			info.SSID = parts[0]
			// Signal is the 6th field (0-indexed: 5)
			if sig, err := strconv.Atoi(parts[4]); err == nil {
				info.RSSI = sig - 100 // nmcli signal is percentage, convert to dBm
			}
			if freq, err := strconv.Atoi(parts[2]); err == nil {
				info.Channel = freqToChannel(freq)
			}
			if rate, err := strconv.Atoi(parts[3]); err == nil {
				info.TxRate = rate
			}
			break
		}
	}

	// Also try nmcli device wifi for connected network
	if info.SSID == "" {
		cmd3 := exec.Command("nmcli", "-t", "-f", "IN-USE,SSID,SIGNAL,CHAN,RATE,BSSID", "device", "wifi")
		out3, _ := cmd3.Output()
		for _, line := range strings.Split(string(out3), "\n") {
			if strings.HasPrefix(line, "*:") {
				parts := strings.Split(strings.TrimPrefix(line, "*:"), ":")
				if len(parts) >= 6 {
					info.SSID = parts[0]
					if sig, err := strconv.Atoi(parts[1]); err == nil {
						info.RSSI = sig - 100
					}
					if ch, err := strconv.Atoi(parts[2]); err == nil {
						info.Channel = ch
					}
					if rate, err := strconv.Atoi(parts[3]); err == nil {
						info.TxRate = rate
					}
					info.BSSID = parts[4]
				}
				break
			}
		}
	}

	if info.SSID == "" {
		return nil, fmt.Errorf("cannot get WiFi info from nmcli")
	}
	return info, nil
}

func getWifiFromIwconfig() (*wifiInfo, error) {
	// Find wireless interface
	cmd := exec.Command("sh", "-c", "iw dev 2>/dev/null | grep Interface | head -1 | awk '{print $2}'")
	out, err := cmd.Output()
	if err != nil || len(out) == 0 {
		return nil, fmt.Errorf("no wireless interface")
	}
	iface := strings.TrimSpace(string(out))

	// Get link info
	cmd2 := exec.Command("iw", "dev", iface, "link")
	out2, err2 := cmd2.Output()
	if err2 != nil {
		return nil, err2
	}
	output := string(out2)

	info := &wifiInfo{}
	info.SSID = extractField(output, `SSID:\s*(.+)`)
	info.BSSID = extractField(output, `Connected to\s+(.+)`)
	if sigStr := extractField(output, `signal:\s*(-?\d+)`); sigStr != "" {
		info.RSSI, _ = strconv.Atoi(sigStr)
	}
	if freqStr := extractField(output, `freq:\s*(\d+)`); freqStr != "" {
		if freq, err := strconv.Atoi(freqStr); err == nil {
			info.Channel = freqToChannel(freq)
		}
	}
	if txStr := extractField(output, `tx bitrate:\s*([\d.]+)`); txStr != "" {
		if rate, err := strconv.ParseFloat(txStr, 64); err == nil {
			info.TxRate = int(rate)
		}
	}

	return info, nil
}

func freqToChannel(freq int) int {
	// 2.4 GHz: channel = (freq - 2412) / 5 + 1
	// 5 GHz: channel = (freq - 5000) / 5
	if freq >= 2412 && freq <= 2484 {
		return (freq-2412)/5 + 1
	}
	if freq >= 5000 && freq <= 6000 {
		return (freq - 5000) / 5
	}
	return freq
}

// --- DNS config (Linux) ---

func getCurrentDNS() string {
	// Try systemd-resolved first (most common on modern Linux)
	cmd := exec.Command("resolvectl", "dns")
	out, err := cmd.Output()
	if err == nil {
		return parseResolvectlDNS(string(out))
	}

	// Fallback: nmcli
	cmd2 := exec.Command("nmcli", "-t", "-f", "IP4.DNS", "device", "show")
	out2, err2 := cmd2.Output()
	if err2 == nil {
		servers := parseNmcliDNS(string(out2))
		if servers != "" {
			return servers
		}
	}

	// Fallback: read /etc/resolv.conf
	cmd3 := exec.Command("cat", "/etc/resolv.conf")
	out3, err3 := cmd3.Output()
	if err3 == nil {
		return parseResolvConf(string(out3))
	}

	return ""
}

func parseResolvectlDNS(output string) string {
	// Find all IP-like entries in the output
	var servers []string
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if ip := net.ParseIP(line); ip != nil {
			servers = append(servers, line)
		}
	}
	return strings.Join(servers, ", ")
}

func parseNmcliDNS(output string) string {
	re := regexp.MustCompile(`IP4\.DNS\[[\d]*\]:\s*(.+)`)
	matches := re.FindAllStringSubmatch(output, -1)
	var servers []string
	seen := map[string]bool{}
	for _, m := range matches {
		if len(m) >= 2 {
			s := strings.TrimSpace(m[1])
			if s != "" && !seen[s] {
				seen[s] = true
				servers = append(servers, s)
			}
		}
	}
	return strings.Join(servers, ", ")
}

func parseResolvConf(output string) string {
	re := regexp.MustCompile(`^\s*nameserver\s+(.+)`)
	var servers []string
	for _, line := range strings.Split(output, "\n") {
		if m := re.FindStringSubmatch(line); len(m) >= 2 {
			servers = append(servers, strings.TrimSpace(m[1]))
		}
	}
	return strings.Join(servers, ", ")
}

// --- Network service (Linux) ---

func getActiveNetworkService() string {
	// nmcli
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "connection", "show", "--active")
	out, err := cmd.Output()
	if err != nil {
		return "eth0"
	}
	// Take the first connected interface name
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) >= 3 {
			return parts[2] // device name
		}
	}
	return "eth0"
}

// --- Proxy state (Linux) ---

func getProxyState(proxyType string) string {
	// Check environment variables first
	envMap := map[string]string{
		"webproxy":          "http_proxy",
		"securewebproxy":    "https_proxy",
		"socksfirewallproxy": "all_proxy",
	}

	envKey, ok := envMap[proxyType]
	if !ok {
		envKey = "http_proxy"
	}

	// Check uppercase and lowercase
	for _, key := range []string{strings.ToUpper(envKey), strings.ToLower(envKey)} {
		cmd := exec.Command("sh", "-c", fmt.Sprintf("echo $%s", key))
		out, err := cmd.Output()
		if err == nil {
			val := strings.TrimSpace(string(out))
			if val != "" && val != " " {
				return "✅ " + val
			}
		}
	}

	// Check gsettings (GNOME)
	cmd := exec.Command("gsettings", "get", "org.gnome.system.proxy", "mode")
	out, err := cmd.Output()
	if err == nil {
		mode := strings.Trim(string(out), "'\n ")
		if mode == "manual" {
			// Get proxy host:port
			cmd2 := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "host")
			cmd3 := exec.Command("gsettings", "get", "org.gnome.system.proxy.http", "port")
			if host, err2 := cmd2.Output(); err2 == nil {
				hostStr := strings.Trim(string(host), "'\n ")
				portStr := ""
				if port, err3 := cmd3.Output(); err3 == nil {
					portStr = ":" + strings.Trim(string(port), "'\n ")
				}
				if hostStr != "" {
					return "✅ " + hostStr + portStr
				}
			}
			return "✅ Enabled"
		}
	}

	return "❌ Disabled"
}
