// Package model defines shared types used across NetPulse packages.
package model

// TraceHop represents a single hop in a traceroute path.
type TraceHop struct {
	Hop int     `json:"hop"`
	IP  string  `json:"ip"`
	RTT float64 `json:"rtt_ms"`
}

// WifiInfo holds WiFi interface details.
type WifiInfo struct {
	SSID    string `json:"ssid"`
	BSSID   string `json:"bssid"`
	RSSI    int    `json:"rssi"`
	Noise   int    `json:"noise"`
	Channel int    `json:"channel"`
	TxRate  int    `json:"tx_rate"`
	PhyMode string `json:"phy_mode"`
	Country string `json:"country_code"`
}

// Finding is a single diagnosis item with severity, explanation, and optional fix.
type Finding struct {
	Severity    string `json:"severity"` // good, info, warn, bad
	Category    string `json:"category"` // latency, dns, wifi, packet_loss, proxy, general
	Title       string `json:"title"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Action      string `json:"action,omitempty"`
	ActionLabel string `json:"action_label,omitempty"`
}

// DiagnoseResult holds the full diagnosis output.
type DiagnoseResult struct {
	Findings       []Finding      `json:"findings"`
	OverallScore   int            `json:"overall_score"`
	OverallVerdict string         `json:"overall_verdict"`
	Summary        string         `json:"summary"`
	CheckedAt      string         `json:"checked_at"`
	Metrics        DiagnoseMetrics `json:"metrics"`
}

// DiagnoseMetrics holds raw network metrics collected during diagnosis.
type DiagnoseMetrics struct {
	LatencyMs   float64 `json:"latency_ms"`
	PacketLoss  float64 `json:"packet_loss_pct"`
	DnsSpeedMs  float64 `json:"dns_speed_ms"`
	WifiRSSI    int     `json:"wifi_rssi"`
	WifiSSID    string  `json:"wifi_ssid"`
	DNSServers  string  `json:"dns_servers"`
	NetworkName string  `json:"network_name"`
	HasProxy    bool    `json:"has_proxy"`
}

// APIResponse is the standard JSON envelope for all API responses.
type APIResponse struct {
	OK        bool        `json:"ok"`
	Data      interface{} `json:"data"`
	Error     string      `json:"error"`
	ElapsedMs int64       `json:"elapsed_ms"`
}

// DNSPreset describes a predefined DNS server configuration.
type DNSPreset struct {
	Name    string   `json:"name"`
	Servers []string `json:"servers"`
}

// DefaultDNSPresets returns the built-in DNS preset configurations.
func DefaultDNSPresets() []DNSPreset {
	return []DNSPreset{
		{"114DNS", []string{"114.114.114.114", "114.114.115.115"}},
		{"AliDNS", []string{"223.5.5.5", "223.6.6.6"}},
		{"DNSPod", []string{"119.29.29.29", "119.28.28.28"}},
		{"Cloudflare", []string{"1.1.1.1", "1.0.0.1"}},
		{"Google", []string{"8.8.8.8", "8.8.4.4"}},
	}
}
