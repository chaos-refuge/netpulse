//go:build windows

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func setDNSServers(networkService string, servers []string) error {
	if len(servers) == 0 {
		return fmt.Errorf("no DNS servers provided")
	}

	// Windows: netsh interface ip set dns "interface" static <dns>
	cmd := exec.Command("netsh", "interface", "ip", "set", "dns",
		networkService, "static", servers[0])
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}

	// Add additional DNS servers
	for i := 1; i < len(servers); i++ {
		cmd := exec.Command("netsh", "interface", "ip", "add", "dns",
			networkService, servers[i], fmt.Sprintf("index=%d", i+1))
		cmd.CombinedOutput() // ignore errors for add (might already exist)
	}

	return nil
}

func flushDNSCache() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}
	return nil
}

func toggleProxy(networkService, proxyType string, enable bool) error {
	switch proxyType {
	case "http", "https":
		// Use netsh winhttp for system proxy
		if enable {
			// netsh winhttp set proxy proxy-server="http=127.0.0.1:7890;https=127.0.0.1:7890"
			// This is a simplified toggle — full proxy config would need server:port input
			cmd := exec.Command("reg", "add",
				`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
				"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "1", "/f")
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
			}
		} else {
			cmd := exec.Command("reg", "add",
				`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
				"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", "0", "/f")
			out, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
			}
		}
		return nil

	case "socks":
		return fmt.Errorf("SOCKS proxy toggle requires third-party tool on Windows; use GUI settings")

	default:
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}
}
