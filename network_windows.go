//go:build windows

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

// --- Ping (Windows) ---

func quickPing(target string, count int, timeout time.Duration) (avgRTT float64, lossPct float64, err error) {
	if ip := net.ParseIP(target); ip == nil {
		addrs, e := net.LookupHost(target)
		if e != nil || len(addrs) == 0 {
			return 0, 100, fmt.Errorf("cannot resolve %s", target)
		}
		target = addrs[0]
	}

	timeoutMs := int(timeout.Milliseconds())*count + 2000
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), "-w", "2000", target)
	out, runErr := cmd.Output()
	output := string(out)

	// Windows: "Lost = 0 (0% loss)"
	lossRe := regexp.MustCompile(`Lost = \d+ \((\d+)%`)
	if m := lossRe.FindStringSubmatch(output); len(m) >= 2 {
		lossPct, _ = strconv.ParseFloat(m[1], 64)
	}

	// Windows: "Average = 3ms" or "Average = 0ms" inside round-trip section
	rttRe := regexp.MustCompile(`Average = (\d+)ms`)
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

// --- Traceroute (Windows - tracert) ---

func doTrace(target string, maxHops int, timeout time.Duration) ([]traceHop, error) {
	timeoutMs := int(timeout.Milliseconds()) * maxHops * 3
	if timeoutMs < 30000 {
		timeoutMs = 30000
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "tracert", "-h", strconv.Itoa(maxHops), "-w", "1000", target)
	out, err := cmd.Output()
	output := string(out)

	return parseTraceOutputWindows(output, err)
}

func parseTraceOutputWindows(output string, cmdErr error) ([]traceHop, error) {
	var hops []traceHop
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`^\s*(\d+)\s+(<?\d+\s*ms|[*])\s+(<?\d+\s*ms|[*])\s+(<?\d+\s*ms|[*])\s+(.+)`)

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) < 6 {
			continue
		}
		hop, _ := strconv.Atoi(m[1])
		ip := strings.TrimSpace(m[5])
		// remove brackets from "[192.168.1.1]"
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")

		th := traceHop{Hop: hop, IP: ip}

		// Try to extract RTT from the first time column
		rttRe := regexp.MustCompile(`<?(\d+)\s*ms`)
		if rttM := rttRe.FindStringSubmatch(m[2]); len(rttM) >= 2 {
			if rtt, err := strconv.ParseFloat(rttM[1], 64); err == nil {
				th.RTT = rtt
			}
		}
		hops = append(hops, th)
	}

	if len(hops) == 0 && cmdErr != nil {
		return nil, cmdErr
	}
	return hops, nil
}

// --- WiFi (Windows - netsh wlan) ---

func getWifiInfo() (*wifiInfo, error) {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netsh wlan failed: %w", err)
	}
	output := string(out)

	info := &wifiInfo{}

	info.SSID = extractField(output, `\s*SSID\s*:\s*(.+)`)
	info.BSSID = extractField(output, `\s*BSSID\s*:\s*(.+)`)

	// Signal: "Signal                 : 85%"
	if sigStr := extractField(output, `\s*Signal\s*:\s*(\d+)%`); sigStr != "" {
		if pct, err := strconv.Atoi(sigStr); err == nil {
			// Convert percentage to approximate dBm: dBm ≈ (pct/2) - 100
			info.RSSI = (pct / 2) - 100
		}
	}

	// Channel
	if chStr := extractField(output, `\s*Channel\s*:\s*(\d+)`); chStr != "" {
		info.Channel, _ = strconv.Atoi(chStr)
	}

	// Radio type (protocol)
	info.PhyMode = extractField(output, `\s*Radio type\s*:\s*(.+)`)

	// Tx rate
	if rateStr := extractField(output, `\s*Transmit rate \(Mbps\)\s*:\s*(\d+)`); rateStr != "" {
		info.TxRate, _ = strconv.Atoi(rateStr)
	} else if rateStr = extractField(output, `\s*Transmit rate\s*:\s*(\d+)`); rateStr != "" {
		info.TxRate, _ = strconv.Atoi(rateStr)
	}

	// State check - if not connected, SSID might be empty
	state := strings.ToLower(extractField(output, `\s*State\s*:\s*(.+)`))
	if !strings.Contains(state, "connected") {
		return info, fmt.Errorf("WiFi not connected")
	}

	return info, nil
}

// --- DNS config (Windows) ---

func getCurrentDNS() string {
	// Try ipconfig /all first - parse DNS Servers lines
	cmd := exec.Command("ipconfig", "/all")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	output := string(out)

	// Find DNS Servers entries
	re := regexp.MustCompile(`DNS Servers[ .]*: (.+)`)
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
	if len(servers) > 0 {
		return strings.Join(servers, ", ")
	}
	return ""
}

// --- Network service (Windows) ---

func getActiveNetworkService() string {
	// Find connected interface name via netsh
	cmd := exec.Command("netsh", "interface", "show", "interface")
	out, err := cmd.Output()
	if err != nil {
		return "Ethernet"
	}
	output := string(out)

	// Format: "Enabled   Connected   Dedicated   Wi-Fi"
	// Look for "Connected" status
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Connected") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				return fields[3] // interface name
			}
		}
	}
	return "Ethernet"
}

// --- Proxy state (Windows) ---

func getProxyState(proxyType string) string {
	switch proxyType {
	case "webproxy":
		return getWinHTTPProxy()
	case "socksfirewallproxy":
		return getIEOption("ProxyEnable", "ProxyServer")
	case "securewebproxy":
		return getIEOption("ProxyEnable", "ProxyServer")
	default:
		return "unknown"
	}
}

func getWinHTTPProxy() string {
	cmd := exec.Command("netsh", "winhttp", "show", "proxy")
	out, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	output := string(out)
	if strings.Contains(output, "Direct access") || strings.Contains(output, "直接访问") {
		return "❌ Disabled"
	}
	re := regexp.MustCompile(`Proxy Server\(s\)?\s*:\s*(.+)`)
	if m := re.FindStringSubmatch(output); len(m) >= 2 {
		return "✅ " + strings.TrimSpace(m[1])
	}
	return "❌ Disabled"
}

func getIEOption(enableKey, serverKey string) string {
	// Query IE proxy settings from registry
	cmd := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", enableKey)
	out, err := cmd.Output()
	if err != nil {
		return "❌ Disabled"
	}
	output := string(out)
	if strings.Contains(output, "0x1") {
		// Proxy is enabled, check server
		cmd2 := exec.Command("reg", "query",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", serverKey)
		if out2, err2 := cmd2.Output(); err2 == nil {
			if m := regexp.MustCompile(`REG_SZ\s+(.+)`).FindStringSubmatch(string(out2)); len(m) >= 2 {
				return "✅ " + strings.TrimSpace(m[1])
			}
		}
		return "✅ Enabled"
	}
	return "❌ Disabled"
}
