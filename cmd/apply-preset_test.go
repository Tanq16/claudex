package cmd

import "testing"

func TestPresetParts(t *testing.T) {
	tests := []struct {
		name       string
		skillsFlag bool
		agentsFlag bool
		skills     bool
		agents     bool
	}{
		{"neither flag applies both halves", false, false, true, true},
		{"--skills leaves AGENTS.md alone", true, false, true, false},
		{"--agents links no skills", false, true, false, true},
		{"both flags are the same as neither", true, true, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			skills, agents := presetParts(tt.skillsFlag, tt.agentsFlag)
			if skills != tt.skills || agents != tt.agents {
				t.Fatalf("presetParts(%v, %v) = (%v, %v), want (%v, %v)", tt.skillsFlag, tt.agentsFlag, skills, agents, tt.skills, tt.agents)
			}
		})
	}
}
