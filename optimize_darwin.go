//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strings"
)

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

func toggleProxy(networkService, proxyType string, enable bool) error {
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
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(out)))
	}
	return nil
}
