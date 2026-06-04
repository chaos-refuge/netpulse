// NetPulse — macOS/Linux/Windows network diagnostics & optimization tool.
package main

import (
	"embed"
	"fmt"
	"log/slog"
	"os"

	"github.com/vosskstudio/netpulse/internal/api"
	"github.com/vosskstudio/netpulse/internal/config"
)

//go:embed frontend/css/* frontend/js/* frontend/*
var frontend embed.FS

func main() {
	cfg := config.Default()

	if len(os.Args) > 1 {
		for i := 1; i < len(os.Args); i++ {
			switch os.Args[i] {
			case "-h", "--help":
				fmt.Println("NetPulse — Network Diagnostics & Optimization")
				fmt.Println("Usage: netpulse [options]")
				fmt.Println("  -h, --help    Show this help")
				fmt.Println("  -p, --port N  Set preferred port (default: 8080)")
				os.Exit(0)
			case "-p", "--port":
				if i+1 < len(os.Args) {
					var port int
					if _, err := fmt.Sscanf(os.Args[i+1], "%d", &port); err == nil {
						cfg.Port = port
						cfg.StartPort = port
					}
					i++
				}
			}
		}
	}

	srv := api.NewServer(cfg, frontend)
	if err := api.Run(srv, cfg.BrowserOpen); err != nil {
		slog.Error("server error", "error", err)
		os.Exit(1)
	}
}
