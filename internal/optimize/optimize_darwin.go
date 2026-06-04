//go:build darwin

package optimize

import (
	"fmt"
	"os/exec"
	"strings"
)

// SetDNSServers configures DNS servers for the given network service.
func SetDNSServers(networkService string, servers []string) error {
	args := []string{"-setdnsservers", networkService}
	args = append(args, servers...)
	cmd := exec.Command("networksetup", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("networksetup: %s: %w", strings.TrimSpace(string(out)), err)
	}
	return nil
}

// FlushDNSCache clears the local DNS resolver cache.
func FlushDNSCache() error {
	cmds := [][]string{
		{"sudo", "dscacheutil", "-flushcache"},
		{"sudo", "killall", "-HUP", "mDNSResponder"},
		{"sudo", "killall", "-HUP", "mDNSResponderHelper"},
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

// ToggleProxy enables or disables a proxy type on the given network service.
func ToggleProxy(networkService, proxyType string, enable bool) error {
	flag := "off"
	if enable {
		flag = "on"
	}

	proxyArg := map[string]string{
		"http":  "-setwebproxystate",
		"socks": "-setsocksfirewallproxystate",
		"https": "-setsecurewebproxystate",
	}[proxyType]
	if proxyArg == "" {
		return fmt.Errorf("unknown proxy type: %s", proxyType)
	}

	cmd := exec.Command("networksetup", proxyArg, networkService, flag)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("networksetup %s: %s: %w", proxyArg, strings.TrimSpace(string(out)), err)
	}
	return nil
}
