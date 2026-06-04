// Package diagnose provides the network health diagnosis engine.
package diagnose

import (
	"strings"
	"time"

	"github.com/vosskstudio/netpulse/internal/detect"
	"github.com/vosskstudio/netpulse/internal/model"
)

// collectMetrics gathers raw network metrics using the detect package.
func collectMetrics() model.DiagnoseMetrics {
	m := model.DiagnoseMetrics{}

	if rtt, pct, err := detect.QuickPing("www.baidu.com", 5, 3*time.Second); err == nil {
		m.LatencyMs = rtt
		m.PacketLoss = pct
	} else {
		m.LatencyMs = -1
		m.PacketLoss = -1
	}

	if ds, err := detect.DNSResolveSpeed("www.baidu.com"); err == nil {
		m.DnsSpeedMs = ds
	} else {
		m.DnsSpeedMs = -1
	}

	if info, err := detect.GetWifiInfo(); err == nil {
		m.WifiSSID = info.SSID
		m.WifiRSSI = info.RSSI
	}

	m.DNSServers = detect.GetCurrentDNS()
	m.NetworkName = detect.GetActiveNetworkService()

	proxyState := detect.GetProxyState("webproxy")
	m.HasProxy = strings.Contains(proxyState, "✅")

	return m
}

// Analyze runs the full diagnosis pipeline: collect metrics, analyze findings, score.
func Analyze() model.DiagnoseResult {
	result := model.DiagnoseResult{
		CheckedAt: time.Now().Format("15:04:05"),
	}

	result.Metrics = collectMetrics()
	result.Findings = analyzeFindings(result.Metrics)
	result.OverallScore = CalcScore(result.Findings)
	result.OverallVerdict, result.Summary = verdictAndSummary(result.OverallScore)

	return result
}

func analyzeFindings(m model.DiagnoseMetrics) []model.Finding {
	var findings []model.Finding

	// Latency
	switch {
	case m.LatencyMs < 0:
		findings = append(findings, model.Finding{
			Severity: "bad", Category: "latency",
			Title:       "网络不通",
			Description: "Ping 测试完全无响应，可能网络已断开或目标不可达。",
			Suggestion:  "检查网线/WiFi 是否连接、路由器是否正常工作、是否开启了 VPN 导致路由异常。",
		})
	case m.LatencyMs > 300:
		findings = append(findings, model.Finding{
			Severity: "bad", Category: "latency",
			Title:       "延迟极高",
			Description: "Ping 延迟超过 300ms，网络体验会很差。",
			Suggestion:  "可能是 VPN/代理引起的额外延迟；也可能是 WiFi 信号太弱。试试关闭代理、靠近路由器、或切换 DNS 服务器。",
			Action:      "flush-dns", ActionLabel: "🚀 刷新 DNS 缓存",
		})
	case m.LatencyMs > 100:
		findings = append(findings, model.Finding{
			Severity: "warn", Category: "latency",
			Title:       "延迟偏高",
			Description: "Ping 延迟在 100-300ms 之间，浏览网页可能感觉稍有延迟。",
			Suggestion:  "如果开启了 VPN/代理，这是正常范围。否则可以尝试切换 DNS 或检查 WiFi 信号。",
		})
	default:
		findings = append(findings, model.Finding{
			Severity: "good", Category: "latency",
			Title: "延迟正常", Description: "Ping 延迟在合理范围内，网络响应迅速。",
		})
	}

	// Packet Loss
	switch {
	case m.PacketLoss > 20:
		findings = append(findings, model.Finding{
			Severity: "bad", Category: "packet_loss",
			Title:       "严重丢包",
			Description: "丢包率超过 20%，网络极不稳定。",
			Suggestion:  "检查 WiFi 信号强度（当前 " + WifiRSSIToString(m.WifiRSSI) + "），尝试靠近路由器或切换到 5GHz 频段。",
		})
	case m.PacketLoss > 5:
		findings = append(findings, model.Finding{
			Severity: "warn", Category: "packet_loss",
			Title:       "轻微丢包",
			Description: "丢包率在 5%-20% 之间，可能偶尔感觉到网络不稳定。",
			Suggestion:  "可能是 WiFi 干扰导致，尝试切换 WiFi 信道或靠近路由器。",
		})
	case m.PacketLoss >= 0:
		findings = append(findings, model.Finding{
			Severity: "good", Category: "packet_loss",
			Title: "无丢包", Description: "网络连接稳定，未检测到丢包。",
		})
	}

	// DNS Speed
	switch {
	case m.DnsSpeedMs > 200:
		findings = append(findings, model.Finding{
			Severity: "bad", Category: "dns",
			Title:       "DNS 解析极慢",
			Description: "DNS 解析超过 200ms，每次打开网页都会多等 0.2 秒以上。",
			Suggestion:  "当前 DNS 服务器响应太慢，建议切换到更快的 DNS。",
			Action:      "switch-dns:223.5.5.5,223.6.6.6", ActionLabel: "⚡ 切换到阿里 DNS",
		})
	case m.DnsSpeedMs > 80:
		findings = append(findings, model.Finding{
			Severity: "warn", Category: "dns",
			Title:       "DNS 解析偏慢",
			Description: "DNS 解析在 80-200ms 之间，还有优化空间。",
			Suggestion:  "尝试切换到 114DNS (114.114.114.114) 或阿里 DNS (223.5.5.5)。",
		})
	case m.DnsSpeedMs >= 0:
		findings = append(findings, model.Finding{
			Severity: "good", Category: "dns",
			Title: "DNS 解析迅速", Description: "DNS 解析速度正常，域名查询响应快。",
		})
	}

	// WiFi Signal
	if m.WifiSSID != "" {
		rssi := m.WifiRSSI
		switch {
		case rssi < -80:
			findings = append(findings, model.Finding{
				Severity: "bad", Category: "wifi",
				Title:       "WiFi 信号极弱",
				Description: "信号强度低于 -80 dBm，连接可能经常断开。",
				Suggestion:  "尽量靠近路由器，减少障碍物。如果可能，切换到 5GHz 频段。",
			})
		case rssi < -70:
			findings = append(findings, model.Finding{
				Severity: "warn", Category: "wifi",
				Title:       "WiFi 信号偏弱",
				Description: "信号强度在 -70 到 -80 dBm 之间。",
				Suggestion:  "靠近路由器 1-2 米试试，或调整路由器天线方向。",
			})
		default:
			findings = append(findings, model.Finding{
				Severity: "good", Category: "wifi",
				Title: "WiFi 信号良好", Description: "信号强度正常（" + WifiRSSIToString(rssi) + "），连接稳定。",
			})
		}
	}

	// Proxy
	if m.HasProxy {
		if m.LatencyMs > 150 {
			findings = append(findings, model.Finding{
				Severity: "info", Category: "proxy",
				Title:       "代理已开启，延迟较高",
				Description: "检测到 HTTP 代理已启用。当前延迟偏高，可能是代理服务器本身速度慢。",
				Suggestion:  "如果不需要代理，可以关闭它来提升网速。",
			})
		} else {
			findings = append(findings, model.Finding{
				Severity: "info", Category: "proxy",
				Title:       "代理已启用",
				Description: "HTTP/SOCKS 代理正在运行，所有流量经过代理服务器。",
				Suggestion:  "代理开启状态下部分检测（如路由追踪）可能不准确。一切正常。",
			})
		}
	}

	// DNS server check
	if strings.Contains(m.DNSServers, "114.114.114.114") && m.DnsSpeedMs > 80 {
		findings = append(findings, model.Finding{
			Severity: "info", Category: "dns",
			Title:       "114DNS 当前响应偏慢",
			Description: "你正在使用 114DNS，但当前解析速度不太理想。",
			Suggestion:  "换个 DNS 试试？阿里 DNS (223.5.5.5) 或 DNSPod (119.29.29.29) 在某些网络环境下更快。",
			Action:      "switch-dns:223.5.5.5,223.6.6.6", ActionLabel: "⚡ 试试阿里 DNS",
		})
	}

	return findings
}

// CalcScore computes an overall health score from findings (0-100).
func CalcScore(findings []model.Finding) int {
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

// WifiRSSIToString returns a human-readable signal quality label.
func WifiRSSIToString(rssi int) string {
	switch {
	case rssi >= -50:
		return "优秀"
	case rssi >= -60:
		return "良好"
	case rssi >= -70:
		return "一般"
	case rssi >= -80:
		return "较弱"
	default:
		return "极弱"
	}
}

func verdictAndSummary(score int) (string, string) {
	switch {
	case score >= 90:
		return "🟢 一切正常", "网络状态良好，各项指标都在正常范围内。"
	case score >= 70:
		return "🟡 有轻微问题", "网络基本正常，但有一些小问题值得优化。"
	case score >= 50:
		return "🟠 需要注意", "网络存在较明显的问题，建议尽快处理。"
	default:
		return "🔴 严重问题", "网络状况很差，需要立即排查。"
	}
}
