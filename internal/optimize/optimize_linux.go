//go:build linux

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

	cmd := exec.Command("resolvectl", "dns", networkService, servers[0])
	_, err := cmd.CombinedOutput()
	if err == nil {
		if len(servers) > 1 {
			args := []string{"dns", networkService}
			args = append(args, servers...)
			cmd2 := exec.Command("resolvectl", args...)
			if out2, err2 := cmd2.CombinedOutput(); err2 != nil {
				return fmt.Errorf("resolvectl: %s: %w", strings.TrimSpace(string(out2)), err2)
			}
		}
		return nil
	}

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
		connName = networkService
	}

	dnsStr := strings.Join(servers, " ")
	cmd3 := exec.Command("nmcli", "connection", "modify", connName, "ipv4.dns", dnsStr)
	out3, err3 := cmd3.CombinedOutput()
	if err3 != nil {
		return fmt.Errorf("nmcli: %s: %w", strings.TrimSpace(string(out3)), err3)
	}

	cmd4 := exec.Command("nmcli", "connection", "up", connName)
	cmd4.CombinedOutput()

	return nil
}

func FlushDNSCache() error {
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
		if out, err := cmd.CombinedOutput(); err != nil {
			lastErr = fmt.Errorf("%s: %s: %w", strings.Join(args, " "), strings.TrimSpace(string(out)), err)
		} else {
			return nil
		}
	}
	return lastErr
}

func ToggleProxy(networkService, proxyType string, enable bool) error {
	if _, err := exec.LookPath("gsettings"); err == nil {
		mode := "none"
		if enable {
			mode = "manual"
		}
		cmd := exec.Command("gsettings", "set", "org.gnome.system.proxy", "mode", mode)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("gsettings: %s: %w", strings.TrimSpace(string(out)), err)
		}
		return nil
	}

	if enable {
		return fmt.Errorf("please set proxy manually: export http_proxy=http://proxy:port")
	}
	return fmt.Errorf("please unset proxy manually: unset http_proxy https_proxy")
}
