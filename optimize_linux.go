//go:build linux

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

	// Try systemd-resolved first (modern Linux: resolvectl)
	cmd := exec.Command("resolvectl", "dns", networkService, servers[0])
	_, err := cmd.CombinedOutput()
	if err == nil {
		// Additional servers (resolvectl supports multiple in one call)
		if len(servers) > 1 {
			args := []string{"dns", networkService}
			args = append(args, servers...)
			cmd2 := exec.Command("resolvectl", args...)
			if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
				return fmt.Errorf("resolvectl: %s: %s", err2.Error(), strings.TrimSpace(string(out2)))
			}
		}
		return nil
	}

	// Try nmcli (NetworkManager)
	// Get connection name from device
	cmd2 := exec.Command("nmcli", "-t", "-f", "NAME,DEVICE", "connection", "show", "--active")
	out2, _ := cmd2.Output()
	var connName string
	for _, line := range strings.Split(string(out2), "\n") {
		parts := strings.Split(line, ":")
		if len(parts) >= 2 && parts[1] == networkService {
			connName = parts[0]
			break
		}
	}
	if connName == "" {
		// Try using device name directly
		connName = networkService
	}

	dnsStr := strings.Join(servers, " ")
	cmd3 := exec.Command("nmcli", "connection", "modify", connName, "ipv4.dns", dnsStr)
	out3, err3 := cmd3.CombinedOutput()
	if err3 != nil {
		return fmt.Errorf("nmcli: %s: %s", err3.Error(), strings.TrimSpace(string(out3)))
	}

	// Apply changes
	cmd4 := exec.Command("nmcli", "connection", "up", connName)
	cmd4.CombinedOutput()

	return nil
}

func flushDNSCache() error {
	// Try systemd-resolved
	cmds := [][]string{
		{"sudo", "resolvectl", "flush-caches"},
		{"sudo", "systemd-resolve", "--flush-caches"},
		{"sudo", "systemctl", "restart", "systemd-resolved"},
		{"sudo", "service", "nscd", "restart"},
		{"sudo", "service", "dnsmasq", "restart"},
	}

	var lastErr error
	for _, args := range cmds {
		cmd := exec.Command(args[0], args[1:]...)
		// Ignore errors for optional services
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%s: %s", strings.Join(args, " "), strings.TrimSpace(string(out)))
		} else {
			return nil
		}
	}
	return lastErr
}

func toggleProxy(networkService, proxyType string, enable bool) error {
	// Linux: Use gsettings (GNOME) or environment variables
	// GNOME proxy settings
	if _, err := exec.LookPath("gsettings"); err == nil {
		mode := "none"
		if enable {
			mode = "manual"
		}
		cmd := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", mode)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("gsettings: %s: %s", err.Error(), strings.TrimSpace(string(out)))
		}
		return nil
	}

	// Fallback: print instructions
	if enable {
		return fmt.Errorf("please set proxy manually: export http_proxy=http://proxy:port")
	}
	return fmt.Errorf("please unset proxy manually: unset http_proxy https_proxy")
}
