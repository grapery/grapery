package config

import "testing"

func TestAIConfig_EffectiveHuoshanRequestTimeoutSeconds(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		ai     AIConfig
		expect int
	}{
		{
			name: "override wins",
			ai: AIConfig{
				RequestTimeoutSeconds:        180,
				HuoshanRequestTimeoutSeconds: 777,
			},
			expect: 777,
		},
		{
			name: "global below floor clamps to floor",
			ai: AIConfig{
				RequestTimeoutSeconds: 180,
			},
			expect: EffectiveHuoshanRequestTimeoutFloor,
		},
		{
			name: "global above floor unchanged",
			ai: AIConfig{
				RequestTimeoutSeconds: 900,
			},
			expect: 900,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ai.EffectiveHuoshanRequestTimeoutSeconds(); got != tt.expect {
				t.Fatalf("EffectiveHuoshanRequestTimeoutSeconds() = %d, want %d", got, tt.expect)
			}
		})
	}
}
