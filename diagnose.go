package main

import (
	"net/http"
	"strings"
	"time"
)

type finding struct {
	Severity     string `json:"severity"` // good, info, warn, bad
	Category     string `json:"category"` // latency, dns, wifi, packet_loss, proxy, general
	Title        string `json:"title"`
	Description  string `json:"description"`
	Suggestion   string `json:"suggestion"`
	Action       string `json:"action,omitempty"` // one-click fix action: "switch-dns:223.5.5.5,223.6.6.6", "flush-dns"
	ActionLabel  string `json:"action_label,omitempty"`
}

type diagnoseResult struct {
	Findings       []finding `json:"findings"`
	OverallScore   int       `json:"overall_score"`   // 0-100
	OverallVerdict string    `json:"overall_verdict"` // 🟢 一切正常 / 🟡 小问题 / 🟠 需要注意 / 🔴 严重问题
	Summary        string    `json:"summary"`
	CheckedAt      string    `json:"checked_at"`
	Metrics        struct {
		LatencyMs    float64 `json:"latency_ms"`
		PacketLoss   float64 `json:"packet_loss_pct"`
		DnsSpeedMs   float64 `json:"dns_speed_ms"`
		WifiRSSI     int     `json:"wifi_rssi"`
		WifiSSID     string  `json:"wifi_ssid"`
		DNSServers   string  `json:"dns_servers"`
		NetworkName  string  `json:"network_name"`
		HasProxy     bool    `json:"has_proxy"`
	} `json:"metrics"`
}

func handleDiagnose(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	result := diagnoseResult{}
	result.CheckedAt = time.Now().Format("15:04:05")

	// --- collect metrics ---
	var findings []finding

	// 1. Ping
	latency := -1.0
	loss := -1.0
	if rtt, pct, err := quickPing("www.baidu.com", 5, 3*time.Second); err == nil {
		latency = rtt
		loss = pct
	}
	result.Metrics.LatencyMs = latency
	result.Metrics.PacketLoss = loss

	// 2. DNS speed
	dnsSpeed := -1.0
	if ds, err := dnsResolveSpeed("www.baidu.com"); err == nil {
		dnsSpeed = ds
	}
	result.Metrics.DnsSpeedMs = dnsSpeed

	// 3. WiFi
	if info, err := getWifiInfo(); err == nil {
		result.Metrics.WifiSSID = info.SSID
		result.Metrics.WifiRSSI = info.RSSI
	}

	// 4. DNS servers
	result.Metrics.DNSServers = getCurrentDNS()

	// 5. Network
	result.Metrics.NetworkName = getActiveNetworkService()

	// 6. Proxy
	proxyState := getProxyState("webproxy")
	result.Metrics.HasProxy = strings.Contains(proxyState, "✅")

	// --- diagnose ---

	// === Latency ===
	if latency < 0 {
		findings = append(findings, finding{
			Severity: "bad", Category: "latency",
			Title:       "网络不通",
			Description: "Ping 测试完全无响应，可能网络已断开或目标不可达。",
			Suggestion:  "检查网线/WiFi 是否连接、路由器是否正常工作、是否开启了 VPN 导致路由异常。",
		})
	} else if latency > 300 {
		findings = append(findings, finding{
			Severity: "bad", Category: "latency",
			Title:       "延迟极高",
			Description: "Ping 延迟超过 300ms，网络体验会很差，网页加载缓慢、视频卡顿。",
			Suggestion:  "可能是 VPN/代理引起的额外延迟；也可能是 WiFi 信号太弱。试试关闭代理、靠近路由器、或切换 DNS 服务器。",
			Action:      "flush-dns",
			ActionLabel: "🚀 刷新 DNS 缓存",
		})
	} else if latency > 100 {
		findings = append(findings, finding{
			Severity: "warn", Category: "latency",
			Title:       "延迟偏高",
			Description: "Ping 延迟在 100-300ms 之间，浏览网页可能感觉稍有延迟。",
			Suggestion:  "如果开启了 VPN/代理，这是正常范围。否则可以尝试切换 DNS 或检查 WiFi 信号。",
		})
	} else {
		findings = append(findings, finding{
			Severity: "good", Category: "latency",
			Title:       "延迟正常",
			Description: "Ping 延迟在合理范围内，网络响应迅速。",
			Suggestion:  "",
		})
	}

	// === Packet Loss ===
	if loss > 20 {
		findings = append(findings, finding{
			Severity: "bad", Category: "packet_loss",
			Title:       "严重丢包",
			Description: "丢包率超过 20%，网络极不稳定，可能导致连接中断、视频会议卡顿。",
			Suggestion:  "检查 WiFi 信号强度（当前 " + wifiRSSIToString(result.Metrics.WifiRSSI) + "），尝试靠近路由器或切换到 5GHz 频段。如果用有线网络，检查网线。",
		})
	} else if loss > 5 {
		findings = append(findings, finding{
			Severity: "warn", Category: "packet_loss",
			Title:       "轻微丢包",
			Description: "丢包率在 5%-20% 之间，可能偶尔感觉到网络不稳定。",
			Suggestion:  "可能是 WiFi 干扰导致，尝试切换 WiFi 信道或靠近路由器。也可能是 VPN 导致的。",
		})
	} else if loss >= 0 {
		findings = append(findings, finding{
			Severity: "good", Category: "packet_loss",
			Title:       "无丢包",
			Description: "网络连接稳定，未检测到丢包。",
			Suggestion:  "",
		})
	}

	// === DNS Speed ===
	if dnsSpeed > 200 {
		findings = append(findings, finding{
			Severity: "bad", Category: "dns",
			Title:       "DNS 解析极慢",
			Description: "DNS 解析超过 200ms，每次打开网页都会多等 0.2 秒以上。",
			Suggestion:  "当前 DNS 服务器响应太慢，建议切换到更快的 DNS。114DNS 和阿里 DNS 在国内通常表现较好。",
			Action:      "switch-dns:223.5.5.5,223.6.6.6",
			ActionLabel: "⚡ 切换到阿里 DNS",
		})
	} else if dnsSpeed > 80 {
		findings = append(findings, finding{
			Severity: "warn", Category: "dns",
			Title:       "DNS 解析偏慢",
			Description: "DNS 解析在 80-200ms 之间，还有优化空间。",
			Suggestion:  "尝试切换到 114DNS (114.114.114.114) 或阿里 DNS (223.5.5.5)，通常能提升解析速度。",
		})
	} else if dnsSpeed >= 0 {
		findings = append(findings, finding{
			Severity: "good", Category: "dns",
			Title:       "DNS 解析迅速",
			Description: "DNS 解析速度正常，域名查询响应快。",
			Suggestion:  "",
		})
	}

	// === WiFi Signal ===
	if result.Metrics.WifiSSID != "" {
		rssi := result.Metrics.WifiRSSI
		if rssi < -80 {
			findings = append(findings, finding{
				Severity: "bad", Category: "wifi",
				Title:       "WiFi 信号极弱",
				Description: "信号强度低于 -80 dBm，连接可能经常断开。",
				Suggestion:  "尽量靠近路由器，减少障碍物（墙、金属家具）。如果可能，切换到 5GHz 频段（干扰少、速度快但穿墙弱）。",
			})
		} else if rssi < -70 {
			findings = append(findings, finding{
				Severity: "warn", Category: "wifi",
				Title:       "WiFi 信号偏弱",
				Description: "信号强度在 -70 到 -80 dBm 之间，高速下载或视频通话可能受影响。",
				Suggestion:  "靠近路由器 1-2 米试试，或调整路由器天线方向。2.4GHz 穿墙好但速度慢，5GHz 穿墙差但速度快——选适合的频段。",
			})
		} else {
			findings = append(findings, finding{
				Severity: "good", Category: "wifi",
				Title:       "WiFi 信号良好",
				Description: "信号强度正常（" + wifiRSSIToString(rssi) + "），连接稳定。",
				Suggestion:  "",
			})
		}
	}

	// === Proxy ===
	if result.Metrics.HasProxy {
		if latency > 150 {
			findings = append(findings, finding{
				Severity: "info", Category: "proxy",
				Title:       "代理已开启，延迟较高",
				Description: "检测到 HTTP 代理已启用。当前延迟偏高，可能是代理服务器本身速度慢。",
				Suggestion:  "如果不需要代理，可以关闭它来提升网速。如果必须使用，可以尝试更换代理节点。",
			})
		} else {
			findings = append(findings, finding{
				Severity: "info", Category: "proxy",
				Title:       "代理已启用",
				Description: "HTTP/SOCKS 代理正在运行，所有流量经过代理服务器。",
				Suggestion:  "代理开启状态下部分检测（如路由追踪）可能不准确。一切正常。",
			})
		}
	}

	// === DNS server check ===
	dnsServers := result.Metrics.DNSServers
	if strings.Contains(dnsServers, "114.114.114.114") && dnsSpeed > 80 {
		findings = append(findings, finding{
			Severity: "info", Category: "dns",
			Title:       "114DNS 当前响应偏慢",
			Description: "你正在使用 114DNS，但当前解析速度不太理想。",
			Suggestion:  "换个 DNS 试试？阿里 DNS (223.5.5.5) 或 DNSPod (119.29.29.29) 在某些网络环境下更快。",
			Action:      "switch-dns:223.5.5.5,223.6.6.6",
			ActionLabel: "⚡ 试试阿里 DNS",
		})
	}

	// Calculate score
	result.Findings = findings
	result.OverallScore = calcScore(findings)

	switch {
	case result.OverallScore >= 90:
		result.OverallVerdict = "🟢 一切正常"
		result.Summary = "网络状态良好，各项指标都在正常范围内。"
	case result.OverallScore >= 70:
		result.OverallVerdict = "🟡 有轻微问题"
		result.Summary = "网络基本正常，但有一些小问题值得优化。"
	case result.OverallScore >= 50:
		result.OverallVerdict = "🟠 需要注意"
		result.Summary = "网络存在较明显的问题，建议尽快处理。"
	default:
		result.OverallVerdict = "🔴 严重问题"
		result.Summary = "网络状况很差，需要立即排查。"
	}

	writeJSON(w, result, nil, time.Since(start))
}

func wifiRSSIToString(rssi int) string {
	if rssi >= -50 {
		return "优秀"
	} else if rssi >= -60 {
		return "良好"
	} else if rssi >= -70 {
		return "一般"
	} else if rssi >= -80 {
		return "较弱"
	}
	return "极弱"
}

func calcScore(findings []finding) int {
	score := 100
	for _, f := range findings {
		switch f.Severity {
		case "bad":
			score -= 20
		case "warn":
			score -= 10
		case "info":
			score -= 2
		}
	}
	if score < 0 {
		score = 0
	}
	return score
}
