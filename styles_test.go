package main

import (
	"strings"
	"testing"
)

func TestStatusIcon(t *testing.T) {
	tests := []struct {
		status       string
		expectedIcon string
	}{
		{"waiting", iconWaiting},
		{"done", iconDone},
		{"permission", iconPermission},
		{"working", iconWorking},
		{"error", iconError},
		{"unknown", iconUnknown},
		{"", iconUnknown},
		{"random", iconUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			result := statusIcon(tt.status)
			if !strings.Contains(result, tt.expectedIcon) {
				t.Errorf("statusIcon(%q) = %q, expected to contain %q", tt.status, result, tt.expectedIcon)
			}
		})
	}
}
