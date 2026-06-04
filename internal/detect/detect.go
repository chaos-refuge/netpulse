// Package detect provides cross-platform network detection utilities.
package detect

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vosskstudio/netpulse/internal/model"
)

// DNSResolveSpeed measures DNS resolution time for a domain.
func DNSResolveSpeed(domain string) (float64, error) {
	t0 := time.Now()
	_, err := net.LookupHost(domain)
	elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
	return elapsed, err
}

// ParseTraceOutput parses macOS/Linux traceroute output into structured hops.
func ParseTraceOutput(output string, cmdErr error) ([]model.TraceHop, error) {
	var hops []model.TraceHop
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
		th := model.TraceHop{Hop: hop}
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

// ExtractField returns the first regexp submatch from output.
func ExtractField(output, pattern string) string {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		return strings.TrimSpace(match[1])
	}
	return ""
}

// ExtractInt returns the first regexp submatch from output as an integer.
func ExtractInt(output, pattern string) (int, error) {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(output)
	if len(match) >= 2 {
		return strconv.Atoi(strings.TrimSpace(match[1]))
	}
	return 0, fmt.Errorf("not found")
}
