package api

import (
	"fmt"
	"log/slog"
	"net"
	"os/exec"
	"runtime"
	"strconv"
)

func findPort(start int) int {
	for p := start; p < start+10; p++ {
		ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", p))
		if err == nil {
			ln.Close()
			return p
		}
	}
	return start
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		cmd = exec.Command("cmd", "/c", "start", url)
	}
	if err := cmd.Run(); err != nil {
		slog.Warn("cannot open browser", "error", err)
	}
}

func itoa(i int) string {
	return strconv.Itoa(i)
}
