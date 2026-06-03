package main

import (
	"context"
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// --- Shared types ---

type traceHop struct {
	Hop int     `json:"hop"`
	IP  string  `json:"ip"`
	RTT float64 `json:"rtt_ms"`
}

type wifiInfo struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	RSSI    int    `json:"rssi"`
	Noise   int    `json:"noise"`
	Channel int    `json:"channel"`
	TxRate  int    `json:"tx_rate"`
	PhyMode string `json:"phy_mode"`
	Country string `json:"country_code"`
}

// --- DNS resolve (cross-platform) ---

func dnsResolveSpeed(domain string) (float64, error) {
	t0 := time.Now()
	_, err := net.LookupHost(domain)
	elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
	return elapsed, err
}

// --- Trace output parser (macOS format) ---

func parseTraceOutput(output string, cmdErr error) ([]traceHop, error) {
	var hops []traceHop
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "traceroute") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		hop, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		th := traceHop{Hop: hop}
		if fields[1] == "*" {
			th.IP = "*"
			hops = append(hops, th)
			continue
		}
		th.IP = fields[1]
		if len(fields) >= 3 {
			rttStr := strings.TrimSuffix(fields[2], "ms")
			rttStr = strings.TrimSpace(rttStr)
			if r, err := strconv.ParseFloat(rttStr, 64); err == nil {
				th.RTT = r
			}
		}
		hops = append(hops, th)
	}
	if len(hops) == 0 && cmdErr != nil {
		return nil, cmdErr
	}
	return hops, nil
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
