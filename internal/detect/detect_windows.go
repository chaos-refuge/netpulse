//go:build windows

package detect

import (
	"context"
	"fmt"
	"net"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vosskstudio/netpulse/internal/model"
)

func QuickPing(target string, count int, timeout time.Duration) (avgRTT float64, lossPct float64, err error) {
	if ip := net.ParseIP(target); ip == nil {
		addrs, e := net.LookupHost(target)
		if e != nil || len(addrs) == 0 {
			return 0, 100, fmt.Errorf("cannot resolve %s: %w", target, e)
		}
		target = addrs[0]
	}

	timeoutMs := int(timeout.Milliseconds())*count + 2000
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-n", strconv.Itoa(count), "-w", "2000", target)
	out, runErr := cmd.Output()
	output := string(out)

	lossRe := regexp.MustCompile(`Lost = \d+ \((\d+)%`)
	if m := lossRe.FindStringSubmatch(output); len(m) >= 2 {
		lossPct, _ = strconv.ParseFloat(m[1], 64)
	}
	rttRe := regexp.MustCompile(`Average = (\d+)ms`)
	if m := rttRe.FindStringSubmatch(output); len(m) >= 2 {
		avgRTT, _ = strconv.ParseFloat(m[1], 64)
	}

	if lossPct == 100 {
		return 0, 100, fmt.Errorf("100%% packet loss")
	}
	if runErr != nil && avgRTT == 0 {
		return 0, lossPct, fmt.Errorf("ping %s: %w", target, runErr)
	}
	return avgRTT, lossPct, nil
}

func DoTrace(target string, maxHops int, timeout time.Duration) ([]model.TraceHop, error) {
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

func parseTraceOutputWindows(output string, cmdErr error) ([]model.TraceHop, error) {
	var hops []model.TraceHop
	lines := strings.Split(output, "\n")
	re := regexp.MustCompile(`^\s*(\d+)\s+(<?\d+\s*ms|[*])\s+(<?\d+\s*ms|[*])\s+(<?\d+\s*ms|[*])\s+(.+)`)

	for _, line := range lines {
		m := re.FindStringSubmatch(line)
		if len(m) < 6 {
			continue
		}
		hop, _ := strconv.Atoi(m[1])
		ip := strings.TrimSpace(m[5])
		ip = strings.TrimPrefix(ip, "[")
		ip = strings.TrimSuffix(ip, "]")

		th := model.TraceHop{Hop: hop, IP: ip}

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

func GetWifiInfo() (*model.WifiInfo, error) {
	cmd := exec.Command("netsh", "wlan", "show", "interfaces")
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("netsh wlan failed: %w", err)
	}
	output := string(out)

	info := &model.WifiInfo{}

	info.SSID = ExtractField(output, `\s*SSID\s*:\s*(.+)`)
	info.BSSID = ExtractField(output, `\s*BSSID\s*:\s*(.+)`)

	if sigStr := ExtractField(output, `\s*Signal\s*:\s*(\d+)%`); sigStr != "" {
		if pct, err := strconv.Atoi(sigStr); err == nil {
			info.RSSI = (pct / 2) - 100
		}
	}

	if chStr := ExtractField(output, `\s*Channel\s*:\s*(\d+)`); chStr != "" {
		info.Channel, _ = strconv.Atoi(chStr)
	}

	info.PhyMode = ExtractField(output, `\s*Radio type\s*:\s*(.+)`)

	if rateStr := ExtractField(output, `\s*Transmit rate \(Mbps\)\s*:\s*(\d+)`); rateStr != "" {
		info.TxRate, _ = strconv.Atoi(rateStr)
	} else if rateStr = ExtractField(output, `\s*Transmit rate\s*:\s*(\d+)`); rateStr != "" {
		info.TxRate, _ = strconv.Atoi(rateStr)
	}

	state := strings.ToLower(ExtractField(output, `\s*State\s*:\s*(.+)`))
	if !strings.Contains(state, "connected") {
		return info, fmt.Errorf("WiFi not connected")
	}

	return info, nil
}

func GetCurrentDNS() string {
	cmd := exec.Command("ipconfig", "/all")
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	output := string(out)

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

func GetActiveNetworkService() string {
	cmd := exec.Command("netsh", "interface", "show", "interface")
	out, err := cmd.Output()
	if err != nil {
		return "Ethernet"
	}
	output := string(out)

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "Connected") {
			fields := strings.Fields(line)
			if len(fields) >= 4 {
				return fields[3]
			}
		}
	}
	return "Ethernet"
}

func GetProxyState(proxyType string) string {
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
	cmd := exec.Command("reg", "query",
		`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
		"/v", enableKey)
	out, err := cmd.Output()
	if err != nil {
		return "❌ Disabled"
	}
	output := string(out)
	if strings.Contains(output, "0x1") {
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
