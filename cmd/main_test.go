package main

import (
	"testing"

	"github.com/tomohiro-owada/devrag/internal/config"
)

func TestShouldCheckForUpdates(t *testing.T) {
	disabled := config.DefaultConfig()
	updateCheckDisabled := false
	disabled.UpdateCheck = &updateCheckDisabled

	enabled := config.DefaultConfig()
	updateCheckEnabled := true
	enabled.UpdateCheck = &updateCheckEnabled

	tests := []struct {
		name  string
		isMCP bool
		cfg   *config.Config
		want  bool
	}{
		{
			name:  "cli mode with default config",
			isMCP: false,
			cfg:   config.DefaultConfig(),
			want:  true,
		},
		{
			name:  "cli mode with update check disabled",
			isMCP: false,
			cfg:   disabled,
			want:  false,
		},
		{
			name:  "cli mode with update check enabled",
			isMCP: false,
			cfg:   enabled,
			want:  true,
		},
		{
			name:  "mcp mode never checks",
			isMCP: true,
			cfg:   enabled,
			want:  false,
		},
		{
			name:  "nil config does not check",
			isMCP: false,
			cfg:   nil,
			want:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldCheckForUpdates(tt.isMCP, tt.cfg); got != tt.want {
				t.Fatalf("shouldCheckForUpdates(%v, %v) = %v, want %v", tt.isMCP, tt.cfg, got, tt.want)
			}
		})
	}
}
