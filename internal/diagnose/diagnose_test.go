package diagnose

import (
	"fmt"
	"testing"

	"github.com/vosskstudio/netpulse/internal/model"
)

func TestCalcScore(t *testing.T) {
	tests := []struct {
		name     string
		findings []model.Finding
		want     int
	}{
		{"empty findings", []model.Finding{}, 100},
		{"all good", []model.Finding{
			{Severity: "good"}, {Severity: "good"}, {Severity: "good"},
		}, 100},
		{"one bad", []model.Finding{{Severity: "bad"}}, 80},
		{"one bad one warn", []model.Finding{{Severity: "bad"}, {Severity: "warn"}}, 70},
		{"multiple bads floor at zero", []model.Finding{
			{Severity: "bad"}, {Severity: "bad"}, {Severity: "bad"},
			{Severity: "bad"}, {Severity: "bad"}, {Severity: "warn"},
		}, 0},
		{"mixed severities", []model.Finding{
			{Severity: "good"}, {Severity: "warn"}, {Severity: "bad"}, {Severity: "info"},
		}, 68},
		{"info only", []model.Finding{
			{Severity: "info"}, {Severity: "info"}, {Severity: "info"},
		}, 94},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := CalcScore(tt.findings); got != tt.want {
				t.Errorf("CalcScore() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestWifiRSSIToString(t *testing.T) {
	tests := []struct {
		rssi int
		want string
	}{
		{-30, "优秀"}, {-40, "优秀"}, {-50, "优秀"},
		{-55, "良好"}, {-60, "良好"},
		{-65, "一般"}, {-70, "一般"},
		{-75, "较弱"}, {-80, "较弱"},
		{-85, "极弱"}, {-90, "极弱"}, {-100, "极弱"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("rssi_%d", tt.rssi), func(t *testing.T) {
			if got := WifiRSSIToString(tt.rssi); got != tt.want {
				t.Errorf("WifiRSSIToString(%d) = %q, want %q", tt.rssi, got, tt.want)
			}
		})
	}
}

func TestVerdictAndSummary(t *testing.T) {
	tests := []struct {
		score       int
		wantVerdict string
	}{
		{100, "🟢 一切正常"},
		{90, "🟢 一切正常"},
		{89, "🟡 有轻微问题"},
		{70, "🟡 有轻微问题"},
		{69, "🟠 需要注意"},
		{50, "🟠 需要注意"},
		{49, "🔴 严重问题"},
		{0, "🔴 严重问题"},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("score_%d", tt.score), func(t *testing.T) {
			verdict, _ := verdictAndSummary(tt.score)
			if verdict != tt.wantVerdict {
				t.Errorf("verdictAndSummary(%d) verdict = %q, want %q", tt.score, verdict, tt.wantVerdict)
			}
		})
	}
}
