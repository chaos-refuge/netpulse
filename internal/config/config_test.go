package config

import (
	"testing"
	"time"
)

func TestDefault(t *testing.T) {
	cfg := Default()
	if cfg.Host != "127.0.0.1" {
		t.Errorf("Default().Host = %q, want 127.0.0.1", cfg.Host)
	}
	if cfg.StartPort != 8080 {
		t.Errorf("Default().StartPort = %d, want 8080", cfg.StartPort)
	}
	if !cfg.BrowserOpen {
		t.Error("Default().BrowserOpen should be true")
	}
	if cfg.PingCount != 5 {
		t.Errorf("Default().PingCount = %d, want 5", cfg.PingCount)
	}
	if cfg.MaxPingCount != 20 {
		t.Errorf("Default().MaxPingCount = %d, want 20", cfg.MaxPingCount)
	}
}

func TestWithPort(t *testing.T) {
	cfg := Default()
	WithPort(3000)(cfg)
	if cfg.Port != 3000 {
		t.Errorf("WithPort(3000): Port = %d, want 3000", cfg.Port)
	}
}

func TestWithoutBrowser(t *testing.T) {
	cfg := Default()
	WithoutBrowser()(cfg)
	if cfg.BrowserOpen {
		t.Error("WithoutBrowser: BrowserOpen should be false")
	}
}

func TestWithPingDefaults(t *testing.T) {
	cfg := Default()
	WithPingDefaults(10, 50, 5*time.Second)(cfg)
	if cfg.PingCount != 10 {
		t.Errorf("PingCount = %d, want 10", cfg.PingCount)
	}
	if cfg.MaxPingCount != 50 {
		t.Errorf("MaxPingCount = %d, want 50", cfg.MaxPingCount)
	}
	if cfg.PingTimeout != 5*time.Second {
		t.Errorf("PingTimeout = %v, want 5s", cfg.PingTimeout)
	}
}

func TestOptionComposition(t *testing.T) {
	cfg := Default()
	WithPort(9090)(cfg)
	WithoutBrowser()(cfg)
	WithPingDefaults(3, 10, 1*time.Second)(cfg)

	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.BrowserOpen {
		t.Error("BrowserOpen should be false")
	}
	if cfg.PingCount != 3 {
		t.Errorf("PingCount = %d, want 3", cfg.PingCount)
	}
}
