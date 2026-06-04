// Package config provides NetPulse server configuration with functional options.
package config

import "time"

// Config holds all server and detection parameters.
type Config struct {
	Host          string
	Port          int
	StartPort     int
	BrowserOpen   bool
	ReadTimeout   time.Duration
	WriteTimeout  time.Duration
	PingTimeout   time.Duration
	DNSTimeout    time.Duration
	TraceTimeout  time.Duration
	PingCount     int
	MaxPingCount  int
	MaxTraceHops  int
	ShutdownWait  time.Duration
}

// Option is a functional option for configuring the server.
type Option func(*Config)

// WithPort sets the preferred server port.
func WithPort(port int) Option {
	return func(c *Config) { c.Port = port }
}

// WithStartPort sets the starting port for auto-discovery.
func WithStartPort(port int) Option {
	return func(c *Config) { c.StartPort = port }
}

// WithoutBrowser disables automatic browser opening.
func WithoutBrowser() Option {
	return func(c *Config) { c.BrowserOpen = false }
}

// WithPingDefaults sets ping-related defaults.
func WithPingDefaults(count, maxCount int, timeout time.Duration) Option {
	return func(c *Config) {
		c.PingCount = count
		c.MaxPingCount = maxCount
		c.PingTimeout = timeout
	}
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return &Config{
		Host:          "127.0.0.1",
		StartPort:     8080,
		BrowserOpen:   true,
		ReadTimeout:   30 * time.Second,
		WriteTimeout:  30 * time.Second,
		PingTimeout:   3 * time.Second,
		DNSTimeout:    2 * time.Second,
		TraceTimeout:  2 * time.Second,
		PingCount:     5,
		MaxPingCount:  20,
		MaxTraceHops:  15,
		ShutdownWait:  5 * time.Second,
	}
}
