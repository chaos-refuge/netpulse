package main

import (
	"embed"
	"fmt"
	"log"
	"net"
	"net/http"
	"os/exec"
	"runtime"
	"time"
)

//go:embed frontend/*
var frontend embed.FS

func main() {
	mux := http.NewServeMux()

	// serve frontend
	mux.HandleFunc("/", serveFrontend)

	// quick status
	mux.HandleFunc("/api/status", handleStatus)

	// detect
	mux.HandleFunc("/api/detect/ping", handlePing)
	mux.HandleFunc("/api/detect/dns", handleDnsSpeed)
	mux.HandleFunc("/api/detect/trace", handleTrace)
	mux.HandleFunc("/api/detect/wifi", handleWifi)
	mux.HandleFunc("/api/detect/config", handleConfig)

	// diagnose
	mux.HandleFunc("/api/diagnose", handleDiagnose)

	// optimize
	mux.HandleFunc("/api/optimize/dns", handleSetDNS)
	mux.HandleFunc("/api/optimize/flush", handleFlushDNS)
	mux.HandleFunc("/api/optimize/proxy", handleProxy)

	port := findPort(8080)
	addr := fmt.Sprintf("127.0.0.1:%d", port)
	srv := &http.Server{Addr: addr, Handler: withCORS(mux)}

	go func() {
		time.Sleep(300 * time.Millisecond)
		openBrowser(fmt.Sprintf("http://%s", addr))
	}()

	log.Printf("🚀 NetPulse running at %s\n", fmt.Sprintf("http://%s", addr))
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}

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
		log.Printf("⚠️  cannot open browser: %v", err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}
