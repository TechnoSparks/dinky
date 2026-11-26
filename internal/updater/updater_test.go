package updater

import "testing"

func TestIsNewer(t *testing.T) {
	cases := []struct {
		name     string
		current  string
		latest   string
		expected bool
	}{
		{"newer release", "1.0.0", "v1.1.0", true},
		{"same version", "v1.1.0", "1.1.0", false},
		{"older current", "0.9.0", "v1.0.0", true},
		{"current with commit suffix", "1.0.0 (abc1234)", "v1.0.1", true},
		{"latest equal with suffix", "v1.2.3 (def5678)", "v1.2.3", false},
		{"dev build", "dev", "v1.0.0", true},
		{"untagged build", "untagged (abc1234)", "v1.0.0", true},
		{"unknown build", "unknown", "v1.0.0", true},
		{"invalid latest", "1.0.0", "not-a-version", false},
		{"prerelease not newer", "1.0.0", "v1.0.0-rc.1", false},
		{"empty current", "", "v1.0.0", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := IsNewer(tc.current, tc.latest)
			if got != tc.expected {
				t.Errorf("IsNewer(%q, %q) = %v, want %v", tc.current, tc.latest, got, tc.expected)
			}
		})
	}
}