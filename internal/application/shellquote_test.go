package application

import "testing"

func TestShellQuote(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"/tmp/dinky", "'/tmp/dinky'"},
		{"/path/with space", "'/path/with space'"},
		{"it's", "'it'\\''s'"},
		{"", "''"},
	}

	for _, tc := range cases {
		got := shellQuote(tc.input)
		if got != tc.expected {
			t.Errorf("shellQuote(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}