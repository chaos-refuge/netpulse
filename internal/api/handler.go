package api

import (
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"time"

	"github.com/vosskstudio/netpulse/internal/detect"
	"github.com/vosskstudio/netpulse/internal/diagnose"
	"github.com/vosskstudio/netpulse/internal/model"
	"github.com/vosskstudio/netpulse/internal/optimize"
)

var (
	errMethodNotAllowed = errors.New("method not allowed; use POST")
	errMissingParam     = errors.New("missing required parameter")
)

func handleStatus(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	type statusData struct {
		Latency     float64 `json:"latency_ms"`
		PacketLoss  float64 `json:"packet_loss_pct"`
		DnsSpeed    float64 `json:"dns_speed_ms"`
		WifiSSID    string  `json:"wifi_ssid"`
		WifiRSSI    int     `json:"wifi_rssi"`
		WifiNoise   int     `json:"wifi_noise"`
		DNSServers  string  `json:"dns_servers"`
		NetworkName string  `json:"network_name"`
	}

	sd := statusData{}

	if rtt, loss, err := detect.QuickPing("www.baidu.com", 3, 2*time.Second); err == nil {
		sd.Latency = rtt
		sd.PacketLoss = loss
	}

	if dnsSpeed, err := detect.DNSResolveSpeed("www.baidu.com"); err == nil {
		sd.DnsSpeed = dnsSpeed
	}

	if info, err := detect.GetWifiInfo(); err == nil {
		sd.WifiSSID = info.SSID
		sd.WifiRSSI = info.RSSI
		sd.WifiNoise = info.Noise
	}

	sd.NetworkName = detect.GetActiveNetworkService()
	sd.DNSServers = detect.GetCurrentDNS()

	writeJSON(w, sd, nil, time.Since(start))
}

func handlePing(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, nil, errMethodNotAllowed, 0)
		return
	}
	var req struct {
		Target string `json:"target"`
		Count  int    `json:"count"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Target == "" {
		req.Target = "114.114.114.114"
	}
	if req.Count <= 0 || req.Count > 20 {
		req.Count = 5
	}

	start := time.Now()
	rtt, loss, err := detect.QuickPing(req.Target, req.Count, 3*time.Second)
	type pingData struct {
		Target     string  `json:"target"`
		AvgRTT     float64 `json:"avg_rtt_ms"`
		PacketLoss float64 `json:"packet_loss_pct"`
		Sent       int     `json:"sent"`
		Received   int     `json:"received"`
	}
	pd := pingData{Target: req.Target, Sent: req.Count}
	if err != nil {
		pd.AvgRTT = -1
		pd.PacketLoss = 100
	} else {
		pd.AvgRTT = rtt
		pd.PacketLoss = loss
		pd.Received = req.Count - int(loss*float64(req.Count)/100)
	}
	writeJSON(w, pd, err, time.Since(start))
}

func handleDnsSpeed(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, nil, errMethodNotAllowed, 0)
		return
	}
	var req struct {
		Domains []string `json:"domains"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if len(req.Domains) == 0 {
		req.Domains = []string{"www.baidu.com", "www.google.com", "www.github.com"}
	}

	start := time.Now()
	type dnsItem struct {
		Domain string  `json:"domain"`
		IP     string  `json:"ip"`
		LatMs  float64 `json:"latency_ms"`
		Error  string  `json:"error,omitempty"`
	}

	var items []dnsItem
	for _, domain := range req.Domains {
		t0 := time.Now()
		ips, err := net.LookupHost(domain)
		elapsed := float64(time.Since(t0).Microseconds()) / 1000.0
		di := dnsItem{Domain: domain, LatMs: elapsed}
		if err != nil {
			di.Error = err.Error()
		} else if len(ips) > 0 {
			di.IP = ips[0]
		}
		items = append(items, di)
	}
	type dnsData struct {
		Results []dnsItem `json:"results"`
		AvgMs   float64   `json:"avg_ms"`
	}
	sum := 0.0
	for _, it := range items {
		sum += it.LatMs
	}
	writeJSON(w, dnsData{Results: items, AvgMs: sum / float64(len(items))}, nil, time.Since(start))
}

func handleTrace(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, nil, errMethodNotAllowed, 0)
		return
	}
	var req struct {
		Target string `json:"target"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Target == "" {
		req.Target = "www.baidu.com"
	}

	start := time.Now()
	hops, err := detect.DoTrace(req.Target, 15, 2*time.Second)
	type traceData struct {
		Target string            `json:"target"`
		Hops   []model.TraceHop  `json:"hops"`
	}
	writeJSON(w, traceData{Target: req.Target, Hops: hops}, err, time.Since(start))
}

func handleWifi(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	info, err := detect.GetWifiInfo()
	if info == nil && err != nil {
		writeJSON(w, nil, err, time.Since(start))
		return
	}
	writeJSON(w, info, err, time.Since(start))
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	type configData struct {
		DNSServers  string `json:"dns_servers"`
		HTTPProxy   string `json:"http_proxy"`
		SOCKSProxy  string `json:"socks_proxy"`
		HTTPSProxy  string `json:"https_proxy"`
		NetworkName string `json:"network_name"`
	}
	cd := configData{
		DNSServers:  detect.GetCurrentDNS(),
		HTTPProxy:   detect.GetProxyState("webproxy"),
		SOCKSProxy:  detect.GetProxyState("socksfirewallproxy"),
		HTTPSProxy:  detect.GetProxyState("securewebproxy"),
		NetworkName: detect.GetActiveNetworkService(),
	}
	writeJSON(w, cd, nil, time.Since(start))
}

func handleDiagnose(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	result := diagnose.Analyze()
	writeJSON(w, result, nil, time.Since(start))
}

func handleSetDNS(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, nil, errMethodNotAllowed, 0)
		return
	}
	var req struct {
		Servers []string `json:"servers"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if len(req.Servers) == 0 {
		writeJSON(w, nil, errMissingParam, 0)
		return
	}

	start := time.Now()
	netName := detect.GetActiveNetworkService()
	err := optimize.SetDNSServers(netName, req.Servers)
	type result struct {
		Network string   `json:"network"`
		Servers []string `json:"servers"`
	}
	writeJSON(w, result{Network: netName, Servers: req.Servers}, err, time.Since(start))
}

func handleFlushDNS(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	err := optimize.FlushDNSCache()
	type result struct {
		Message string `json:"message"`
	}
	msg := "DNS 缓存已刷新"
	if err != nil {
		msg = err.Error()
	}
	writeJSON(w, result{Message: msg}, err, time.Since(start))
}

func handleProxy(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		writeJSON(w, nil, errMethodNotAllowed, 0)
		return
	}
	var req struct {
		Enable bool   `json:"enable"`
		Type   string `json:"type"`
	}
	json.NewDecoder(r.Body).Decode(&req)
	if req.Type == "" {
		req.Type = "http"
	}

	start := time.Now()
	netName := detect.GetActiveNetworkService()
	err := optimize.ToggleProxy(netName, req.Type, req.Enable)
	type result struct {
		Network string `json:"network"`
		Type    string `json:"type"`
		Enabled bool   `json:"enabled"`
	}
	writeJSON(w, result{Network: netName, Type: req.Type, Enabled: req.Enable}, err, time.Since(start))
}

func handlePresets(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	presets := optimize.Presets()
	writeJSON(w, presets, nil, time.Since(start))
}
