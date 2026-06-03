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

// --- Ping ---

func quickPing(target string, count int, timeout time.Duration) (avgRTT float64, lossPct float64, err error) {
	if ip := net.ParseIP(target); ip == nil {
		// resolve first
		addrs, e := net.LookupHost(target)
		if e != nil || len(addrs) == 0 {
			return 0, 100, fmt.Errorf("cannot resolve %s", target)
		}
		target = addrs[0]
	}

	ctx, cancel := withTimeout(timeout * time.Duration(count))
	defer cancel()

	cmd := exec.CommandContext(ctx, "/sbin/ping", "-c", strconv.Itoa(count), "-t", "2", target)
	out, runErr := cmd.Output()

	// parse output even if some packets lost
	output := string(out)

	// packet loss: "3 packets transmitted, 3 packets received, 0.0% packet loss"
	lossRe := regexp.MustCompile(`(\d+\.?\d*)% packet loss`)
	lossMatch := lossRe.FindStringSubmatch(output)
	if len(lossMatch) >= 2 {
		lossPct, _ = strconv.ParseFloat(lossMatch[1], 64)
	}

	// avg rtt: "round-trip min/avg/max/stddev = 1.234/3.456/7.890/1.234 ms"
	rttRe := regexp.MustCompile(`min/avg/max/\w+ = [\d.]+/([\d.]+)/[\d.]+/[\d.]+`)
	rttMatch := rttRe.FindStringSubmatch(output)
	if len(rttMatch) >= 2 {
		avgRTT, _ = strconv.ParseFloat(rttMatch[1], 64)
	}

	if lossPct == 100 {
		return 0, 100, fmt.Errorf("100%% packet loss")
	}

	// if cmd errored but some packets got through, that's ok
	if runErr != nil && avgRTT == 0 {
		return 0, lossPct, runErr
	}

	return avgRTT, lossPct, nil
}

// --- DNS ---

func dnsResolveSpeed(domain string) (float64, error) {
	t0 := time.Now()
	_, err := net.LookupHost(domain)
	elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
	return elapsed, err
}

// --- Traceroute ---

type traceHop struct {
	Hop int    `json:"hop"`
	IP  string `json:"ip"`
	RTT float64 `json:"rtt_ms"`
}

func doTrace(target string, maxHops int, timeout time.Duration) ([]traceHop, error) {
	ctx, cancel := withTimeout(timeout * time.Duration(maxHops))
	defer cancel()

	cmd := exec.CommandContext(ctx, "/usr/sbin/traceroute", "-m", strconv.Itoa(maxHops), "-q", "1", "-w", "1", target)
	out, err := cmd.Output()
	output := string(out)

	var hops []traceHop
	lines := strings.Split(output, "\n")
	// skip first line (header)
	for _, line := range lines {
		if strings.HasPrefix(line, "traceroute") {
			continue
		}
		// line format: " 1  192.168.1.1  1.234 ms"
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		hop, hErr := strconv.Atoi(fields[0])
		if hErr != nil {
			continue
		}
		th := traceHop{Hop: hop}
		// if * means timeout
		if fields[1] == "*" {
			th.IP = "*"
			hops = append(hops, th)
			continue
		}
		th.IP = fields[1]
		// parse RTT
		if len(fields) >= 3 {
			rttStr := strings.TrimSuffix(fields[2], "ms")
			rttStr = strings.TrimSpace(rttStr)
			if r, parseErr := strconv.ParseFloat(rttStr, 64); parseErr == nil {
				th.RTT = r
			}
		}
		hops = append(hops, th)
	}

	if len(hops) == 0 && err != nil {
		return nil, err
	}
	return hops, nil
}

// --- WiFi ---

type wifiInfo struct {
	SSID      string `json:"ssid"`
	BSSID     string `json:"bssid"`
	RSSI      int    `json:"rssi"`
	Noise     int    `json:"noise"`
	Channel   int    `json:"channel"`
	TxRate    int    `json:"tx_rate"`
	PhyMode   string `json:"phy_mode"`
	Country   string `json:"country_code"`
}

func getWifiInfo() (*wifiInfo, error) {
	// Try system_profiler first (works on all macOS, including 26+ without airport)
	info, err := getWifiFromSystemProfiler()
	if err == nil {
		return info, nil
	}

	// Fallback: try airport (older macOS)
	airportPath := "/System/Library/PrivateFrameworks/Apple80211.framework/Versions/Current/Resources/airport"
	if out, err2 := exec.Command(airportPath, "-I").Output(); err2 == nil {
		return parseAirportOutput(string(out))
	}

	// Fallback: networksetup
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

	// Find "Current Network Information:" section
	curIdx := strings.Index(output, "Current Network Information:")
	if curIdx < 0 {
		return nil, fmt.Errorf("no current network info")
	}
	section := output[curIdx:]

	// First non-empty, non-whitespace-only line after "Current Network Information:"
	// is the SSID (followed by ":")
	lines := strings.Split(section, "\n")
	if len(lines) >= 2 {
		ssidLine := strings.TrimSpace(lines[1])
		ssidLine = strings.TrimSuffix(ssidLine, ":")
		ssidLine = strings.TrimSpace(ssidLine)
		if ssidLine != "" && !strings.HasPrefix(ssidLine, "PHY") && !strings.HasPrefix(ssidLine, "Channel") && !strings.HasPrefix(ssidLine, "Signal") {
			info.SSID = ssidLine
		}
	}

	// Parse the rest of the Current Network section
	sigRe := regexp.MustCompile(`Signal / Noise:\s*(-?\d+)\s*dBm\s*/\s*(-?\d+)\s*dBm`)
	sigMatch := sigRe.FindStringSubmatch(section)
	if len(sigMatch) >= 3 {
		info.RSSI, _ = strconv.Atoi(sigMatch[1])
		info.Noise, _ = strconv.Atoi(sigMatch[2])
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

	// Country Code from "Country Code:" line in section
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
	// output: "Current Wi-Fi Network: MyWiFi"
	re := regexp.MustCompile(`Current Wi-Fi Network:\s*(.+)`)
	match := re.FindStringSubmatch(strings.TrimSpace(string(out)))
	if len(match) >= 2 {
		return &wifiInfo{SSID: match[1]}, nil
	}
	return nil, fmt.Errorf("cannot get wifi info")
}

// --- DNS config ---

func getCurrentDNS() string {
	cmd := exec.Command("scutil", "--dns")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	// extract nameserver lines
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

// --- Network service ---

func getActiveNetworkService() string {
	// get the active network service name
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
		// check if this service is active
		// simplest: try Wi-Fi first, then return first non-disabled
		if strings.Contains(line, "Wi-Fi") {
			return line
		}
	}
	// return first line
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "An asterisk") {
			return line
		}
	}
	return "Wi-Fi"
}

// --- Proxy state ---

func getProxyState(proxyType string) string {
	netName := getActiveNetworkService()
	cmd := exec.Command("networksetup", fmt.Sprintf("-get%s", proxyType), netName)
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	output := strings.ToLower(string(out))
	if strings.Contains(output, "enabled: yes") {
		// get server:port too
		re := regexp.MustCompile(`Server:\s*(.+)`)
		m := re.FindStringSubmatch(string(out))
		if len(m) >= 2 {
			return "✅ " + m[1]
		}
		return "✅ Enabled"
	}
	return "❌ Disabled"
}

// --- Helpers ---

func extractField(output, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

func extractInt(output, pattern string) (int, error) {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		return strconv.Atoi(strings.TrimSpace(match[1]))
	}
	return 0, fmt.Errorf("not found")
}

func withTimeout(d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), d)
}
