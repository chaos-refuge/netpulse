//go:build windows

package optimize

import (
	"fmt"
	"os/exec"
	"strings"
)

func SetDNSServers(networkService string, servers []string) error {
	if len(servers) == 0 {
		return fmt.Errorf("no DNS servers provided")
	}

	cmd := exec.Command("netsh", "interface", "ip", "set", "dns",
		networkService, "static", servers[0])
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh dns: %s: %w", strings.TrimSpace(string(out)), err)
	}

	for i := 1; i < len(servers); i++ {
		cmd := exec.Command("netsh", "interface", "ip", "add", "dns",
			networkService, servers[i], fmt.Sprintf("index=%d", i+1))
		cmd.CombinedOutput()
	}

	return nil
}

func FlushDNSCache() error {
	cmd := exec.Command("ipconfig", "/flushdns")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ipconfig flushdns: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

func ToggleProxy(networkService, proxyType string, enable bool) error {
	switch proxyType {
	case "http", "https":
		val := "0"
		if enable {
			val = "1"
		}
		cmd := exec.Command("reg", "add",
			`HKCU\Software\Microsoft\Windows\CurrentVersion\Internet Settings`,
			"/v", "ProxyEnable", "/t", "REG_DWORD", "/d", val, "/f")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("reg proxy: %s: %w", strings.TrimSpace(string(out)), err)
		}
		return nil

	case "socks":
		return fmt.Errorf("SOCKS proxy toggle requires third-party tool on Windows; use GUI settings")

	default:
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}
}
