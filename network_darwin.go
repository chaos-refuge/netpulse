//go:build darwin

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

// --- Ping (macOS) ---

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

	cmd := exec.CommandContext(ctx, "/sbin/ping", "-c", strconv.Itoa(count), "-t", "2", target)
	out, runErr := cmd.Output()
	output := string(out)

	lossRe := regexp.MustCompile(`(\d+\.?\d*)% packet loss`)
	if m := lossRe.FindStringSubmatch(output); len(m) >= 2 {
		lossPct, _ = strconv.ParseFloat(m[1], 64)
	}
	rttRe := regexp.MustCompile(`min/avg/max/\w+ = [\d.]+/([\d.]+)/[\d.]+/[\d.]+`)
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

// --- Traceroute (macOS) ---

func doTrace(target string, maxHops int, timeout time.Duration) ([]traceHop, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout*time.Duration(maxHops))
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/sbin/traceroute", "-m", strconv.Itoa(maxHops), "-q", "1", "-w", "1", target)
	out, err := cmd.Output()
	output := string(out)

	return parseTraceOutput(output, err)
}

// --- WiFi (macOS) ---

func getWifiInfo() (*wifiInfo, error) {
	info, err := getWifiFromSystemProfiler()
	if err == nil {
		return info, nil
	}
	// fallback: try airport
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	if out, err2 := exec.Command(airportPath, "-I").Output(); err2 == nil {
		return parseAirportOutput(string(out))
	}
	return getWifiFallback()
}

func getWifiFromSystemProfiler() (*wifiInfo, error) {
	cmd := exec.Command("system_profiler", "SPAirPortDataType")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	output := string(out)
	info := &wifiInfo{}

	curIdx := strings.Index(output, "Current Network Information:")
	if curIdx < 0 {
		return nil, fmt.Errorf("no current network info")
	}
	section := output[curIdx:]

	lines := strings.Split(section, "\n")
	if len(lines) >= 2 {
		ssidLine := strings.TrimSpace(lines[1])
		ssidLine = strings.TrimSuffix(ssidLine, ":")
		ssidLine = strings.TrimSpace(ssidLine)
		if ssidLine != "" && !strings.HasPrefix(ssidLine, "PHY") &&
			!strings.HasPrefix(ssidLine, "Channel") && !strings.HasPrefix(ssidLine, "Signal") {
			info.SSID = ssidLine
		}
	}

	sigRe := regexp.MustCompile(`Signal / Noise:\s*(-?\d+)\s*dBm\s*/\s*(-?\d+)\s*dBm`)
	if m := sigRe.FindStringSubmatch(section); len(m) >= 3 {
		info.RSSI, _ = strconv.Atoi(m[1])
		info.Noise, _ = strconv.Atoi(m[2])
	}
	phyRe := regexp.MustCompile(`PHY Mode:\s*(.+)`)
	if m := phyRe.FindStringSubmatch(section); len(m) >= 2 {
		info.PhyMode = strings.TrimSpace(m[1])
	}
	chRe := regexp.MustCompile(`Channel:\s*(\d+)`)
	if m := chRe.FindStringSubmatch(section); len(m) >= 2 {
		info.Channel, _ = strconv.Atoi(m[1])
	}
	rateRe := regexp.MustCompile(`Transmit Rate:\s*(\d+)`)
	if m := rateRe.FindStringSubmatch(section); len(m) >= 2 {
		info.TxRate, _ = strconv.Atoi(m[1])
	}
	ccRe := regexp.MustCompile(`Country Code:\s*(\S+)`)
	if m := ccRe.FindStringSubmatch(section); len(m) >= 2 {
		info.Country = m[1]
	}
	return info, nil
}

func parseAirportOutput(output string) (*wifiInfo, error) {
	info := &wifiInfo{}
	info.SSID = extractField(output, `\s+SSID:\s*(.+)`)
	info.BSSID = extractField(output, `\s+BSSID:\s*(.+)`)
	if rssi, err := extractInt(output, `\s+agrCtlRSSI:\s*(-?\d+)`); err == nil {
		info.RSSI = rssi
	}
	if noise, err := extractInt(output, `\s+agrCtlNoise:\s*(-?\d+)`); err == nil {
		info.Noise = noise
	}
	if ch, err := extractInt(output, `\s+channel:\s*(\d+)`); err == nil {
		info.Channel = ch
	}
	if tx, err := extractInt(output, `\s+lastTxRate:\s*(\d+)`); err == nil {
		info.TxRate = tx
	}
	info.PhyMode = extractField(output, `\s+PHY Mode:\s*(.+)`)
	info.Country = extractField(output, `\s+country code:\s*(.+)`)
	return info, nil
}

func getWifiFallback() (*wifiInfo, error) {
	netName := getActiveNetworkService()
	if netName == "" {
		return nil, fmt.Errorf("no active network service")
	}
	cmd := exec.Command("networksetup", "-getairportnetwork", netName)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	re := regexp.MustCompile(`Current Wi-Fi Network:\s*(.+)`)
	match := re.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(match) >= 2 {
		return &wifiInfo{SSID: match[1]}, nil
	}
	return nil, fmt.Errorf("cannot get wifi info")
}

// --- DNS config (macOS) ---

func getCurrentDNS() string {
	cmd := exec.Command("scutil", "--dns")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	re := regexp.MustCompile(`nameserver\[[\d]*\] : (.+)`)
	matches := re.FindAllStringSubmatch(string(out), -1)
	var servers []string
	for _, m := range matches {
		if len(m) >= 2 {
			servers = append(servers, m[1])
		}
	}
	return strings.Join(servers, ", ")
}

// --- Network service (macOS) ---

func getActiveNetworkService() string {
	cmd := exec.Command("networksetup", "-listallnetworkservices")
	out, err := cmd.Output()
	if err != nil {
		return "Wi-Fi"
	}
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "An asterisk") {
			continue
		}
		if strings.Contains(line, "Wi-Fi") {
			return line
		}
	}
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "An asterisk") {
			return line
		}
	}
	return "Wi-Fi"
}

// --- Proxy state (macOS) ---

func getProxyState(proxyType string) string {
	netName := getActiveNetworkService()
	cmd := exec.Command("networksetup", fmt.Sprintf("-get%s", proxyType), netName)
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	output := strings.ToLower(string(out))
	if strings.Contains(output, "enabled: yes") {
		re := regexp.MustCompile(`Server:\s*(.+)`)
		if m := re.FindStringSubmatch(string(out)); len(m) >= 2 {
			return "✅ " + m[1]
		}
		return "✅ Enabled"
	}
	return "❌ Disabled"
}
