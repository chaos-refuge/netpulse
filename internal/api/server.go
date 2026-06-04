package api

import (
	"context"
	"embed"
	"io/fs"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaos-refuge/netpulse/internal/config"
)

// NewServer creates a configured HTTP server with all routes.
func NewServer(cfg *config.Config, frontend embed.FS) *http.Server {
	mux := http.NewServeMux()

	// Frontend
	frontendFS, err := fs.Sub(frontend, "frontend")
	if err != nil {
		slog.Warn("frontend subdir not found, serving from root", "error", err)
		frontendFS = frontend
	}
	mux.Handle("/css/", http.FileServer(http.FS(frontendFS)))
	mux.Handle("/js/", http.FileServer(http.FS(frontendFS)))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}
		data, err := frontend.ReadFile("frontend/index.html")
		if err != nil {
			http.Error(w, "frontend not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write(data)
	})

	// API routes
	mux.HandleFunc("/api/status", handleStatus)
	mux.HandleFunc("/api/detect/ping", handlePing)
	mux.HandleFunc("/api/detect/dns", handleDnsSpeed)
	mux.HandleFunc("/api/detect/trace", handleTrace)
	mux.HandleFunc("/api/detect/wifi", handleWifi)
	mux.HandleFunc("/api/detect/config", handleConfig)
	mux.HandleFunc("/api/diagnose", handleDiagnose)
	mux.HandleFunc("/api/optimize/dns", handleSetDNS)
	mux.HandleFunc("/api/optimize/flush", handleFlushDNS)
	mux.HandleFunc("/api/optimize/proxy", handleProxy)
	mux.HandleFunc("/api/optimize/presets", handlePresets)

	handler := chainMiddleware(mux,
		recoveryMiddleware,
		loggingMiddleware,
		corsMiddleware,
	)

	port := findPort(cfg.StartPort)
	addr := net.JoinHostPort(cfg.Host, itoa(port))

	return &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
	}
}

// Run starts the server with graceful shutdown handling.
func Run(srv *http.Server, launchBrowser bool) error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go func() {
		<-ctx.Done()
		slog.Info("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown error", "error", err)
		}
	}()

	if launchBrowser {
		go func() {
			time.Sleep(300 * time.Millisecond)
			openBrowser("http://" + srv.Addr)
		}()
	}

	slog.Info("NetPulse running", "addr", "http://"+srv.Addr)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}
