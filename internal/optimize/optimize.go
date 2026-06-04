// Package optimize provides network optimization actions: DNS switching, cache flushing, proxy toggling.
package optimize

import "github.com/vosskstudio/netpulse/internal/model"

// Presets returns the built-in DNS server presets.
func Presets() []model.DNSPreset {
	return model.DefaultDNSPresets()
}
