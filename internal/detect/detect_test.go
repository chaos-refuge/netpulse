package detect

import (
	"errors"
	"testing"
)

func TestParseTraceOutput(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		cmdErr  error
		wantLen int
		wantHop int
		wantIP  string
	}{
		{
			name: "normal macOS traceroute",
			output: `traceroute to www.baidu.com (103.235.46.39), 30 hops max, 60 byte packets
 1  192.168.1.1  3.456 ms
 2  10.0.0.1  5.678 ms
 3  103.235.46.39  12.345 ms`,
			cmdErr:  nil,
			wantLen: 3,
			wantHop: 1,
			wantIP:  "192.168.1.1",
		},
		{
			name: "timeout hops with asterisks",
			output: `traceroute to example.com
 1  * * *
 2  * * *
 3  192.168.1.1  5.678 ms`,
			cmdErr:  nil,
			wantLen: 3,
			wantHop: 1,
			wantIP:  "*",
		},
		{
			name:    "empty output with error",
			output:  "",
			cmdErr:  errors.New("timeout"),
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hops, err := ParseTraceOutput(tt.output, tt.cmdErr)
			if len(hops) != tt.wantLen {
				t.Errorf("ParseTraceOutput() len = %d, want %d", len(hops), tt.wantLen)
			}
			if tt.cmdErr != nil && err == nil {
				t.Error("ParseTraceOutput() should return error when cmdErr is set and output is empty")
			}
			if len(hops) > 0 {
				if hops[0].Hop != tt.wantHop {
					t.Errorf("ParseTraceOutput()[0].Hop = %d, want %d", hops[0].Hop, tt.wantHop)
				}
				if hops[0].IP != tt.wantIP {
					t.Errorf("ParseTraceOutput()[0].IP = %q, want %q", hops[0].IP, tt.wantIP)
				}
			}
		})
	}
}

func TestExtractField(t *testing.T) {
	output := "SSID: MyWiFi\nBSSID: aa:bb:cc:dd:ee:ff\nChannel: 6"
	tests := []struct {
		name    string
		pattern string
		want    string
	}{
		{"SSID", `SSID:\s*(.+)`, "MyWiFi"},
		{"BSSID", `BSSID:\s*(.+)`, "aa:bb:cc:dd:ee:ff"},
		{"no match", `NotFound:\s*(.+)`, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ExtractField(output, tt.pattern); got != tt.want {
				t.Errorf("ExtractField() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractInt(t *testing.T) {
	output := "Signal: -44 dBm\nNoise: -92 dBm"
	if val, err := ExtractInt(output, `Signal:\s*(-?\d+)`); err != nil || val != -44 {
		t.Errorf("ExtractInt(Signal) = %d, %v; want -44, nil", val, err)
	}
	if _, err := ExtractInt(output, `NotFound:\s*(\d+)`); err == nil {
		t.Error("ExtractInt(NotFound) should error")
	}
}
