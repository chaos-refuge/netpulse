package main

// --- DNS presets (cross-platform) ---

type dnsPreset struct {
	Name    string
	Servers []string
}

var dnsPresets = []dnsPreset{
	{"114DNS", []string{"114.114.114.114", "114.114.115.115"}},
	{"AliDNS", []string{"223.5.5.5", "223.6.6.6"}},
	{"DNSPod", []string{"119.29.29.29", "119.28.28.28"}},
	{"Cloudflare", []string{"1.1.1.1", "1.0.0.1"}},
	{"Google", []string{"8.8.8.8", "8.8.4.4"}},
}

func getDNSPresets() []dnsPreset {
	return dnsPresets
}
