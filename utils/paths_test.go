package utils

import "testing"

func TestNumberedAccount(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want bool
	}{
		{"plain claude has empty suffix", ".claude", false},
		{"numeric suffix", ".claude2", true},
		{"multi-digit suffix", ".claude10", true},
		{"named suffix", ".claude-alice", false},
		{"no separator word", ".claudesync", false},
		{"claudex", ".claudex", false},
		{"signed suffix", ".claude-2", false},
		{"unrelated dir", ".config", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := numberedAccount(tt.in); got != tt.want {
				t.Fatalf("numberedAccount(%q) = %v, want %v", tt.in, got, tt.want)
			}
		})
	}
}
