package main

import (
	"fmt"
	"os/exec"
	"strings"
)

// --- DNS Optimization ---

func setDNSServers(networkService string, servers []string) error {
	args := []string{"-setdnsservers", networkService}
	args = append(args, servers...)
	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}
	return nil
}

func flushDNSCache() error {
	// try mDNSResponder first (macOS 10.10+)
	cmds := [][]string{
		{"sudo", "dscacheutil", "-flushcache"},
		{"sudo", "killall", "-HUP", "mDNSResponder"},
		{"sudo", "killall", "-HUP", "mDNSResponderHelper"},
	}

	var lastErr error
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		} else {
			return nil
		}
	}
	return lastErr
}

// --- Proxy Optimization ---

func toggleProxy(networkService, proxyType string, enable bool) error {
	var flag string
	if enable {
		flag = "on"
	} else {
		flag = "off"
	}

	var proxyArg string
	switch proxyType {
	case "http":
		proxyArg = "-setwebproxystate"
	case "socks":
		proxyArg = "-setsocksfirewallproxystate"
	case "https":
		proxyArg = "-setsecurewebproxystate"
	default:
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}

	cmd := exec.Command("networksetup", proxyArg, networkService, flag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}
	return nil
}

// --- Preset DNS servers ---

type dnsPreset struct {
	Name    string
	Servers []string
}

var dnsPresets = []dnsPreset{
	{"114DNS", []string{"114.114.114.114", "114.114.115.115"}},
	{"AliDNS", []string{"223.5.5.5", "223.6.6.6"}},
	{"DNSPod", []string{"119.29.29.29", "119.28.28.28"}},
	{"Cloudflare", []string{"1.1.1.1", "1.0.0.1"}},
	{"Google", []string{"8.8.8.8", "8.8.4.4"}},
}

func getDNSPresets() []dnsPreset {
	return dnsPresets
}

// --- Ping target presets ---

var pingTargets = []struct {
	Name string
	Host string
}{
	{"百度", "www.baidu.com"},
	{"Google", "www.google.com"},
	{"GitHub", "www.github.com"},
	{"Cloudflare DNS", "1.1.1.1"},
	{"阿里 DNS", "223.5.5.5"},
}

func getPingTargets() []struct {
	Name string
	Host string
} {
	return pingTargets
}

// ensure imports used
var _ = getDNSPresets
var _ = getPingTargets
