package server

import "testing"

func TestInstallationSourceForPath(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		goBin  string
		goPath string
		home   string
		want   string
	}{
		{"homebrew cellar", "/opt/homebrew/Cellar/crit-plus/0.20.2/bin/crit-plus", "", "", "/Users/me", installationSourceHomebrew},
		{"homebrew opt", "/opt/homebrew/opt/crit-plus/bin/crit-plus", "", "", "/Users/me", installationSourceHomebrew},
		{"homebrew intel opt", "/usr/local/opt/crit-plus/bin/crit-plus", "", "", "/Users/me", installationSourceHomebrew},
		{"linuxbrew opt", "/home/linuxbrew/.linuxbrew/opt/crit-plus/bin/crit-plus", "", "", "/home/me", installationSourceHomebrew},
		{"nix store", "/nix/store/hash-crit-0.20.2/bin/crit-plus", "", "", "/home/me", installationSourceNix},
		{"nix path containing opt/crit-plus", "/nix/store/hash/opt/crit-plus/bin/crit-plus", "", "", "/home/me", installationSourceNix},
		{"manual FHS opt", "/opt/crit-plus/bin/crit-plus", "", "", "/Users/me", installationSourceUnknown},
		{"default Go bin", "/Users/me/go/bin/crit-plus", "", "", "/Users/me", installationSourceGo},
		{"configured Go bin", "/tools/bin/crit-plus", "/tools/bin", "", "/Users/me", installationSourceGo},
		{"GOPATH bin", "/workspace/bin/crit-plus", "", "/workspace", "/Users/me", installationSourceGo},
		{"downloaded binary", "/usr/local/bin/crit-plus", "", "", "/Users/me", installationSourceUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := installationSourceForPath(tt.path, tt.goBin, tt.goPath, tt.home); got != tt.want {
				t.Errorf("installationSourceForPath() = %q, want %q", got, tt.want)
			}
		})
	}
}
