package logging

import (
	"testing"
)

func TestShortenString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"abstratium.dev/tickets/framework/fwctx", "a.t.f.fwctx"},
		{"example.com/path/to/resource", "e.p.t.resource"},
		{"single", "single"}, // No slashes
		{"", ""},             // Empty string
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			actual := shortenString(tc.input)
			if actual != tc.expected {
				t.Errorf("Expected: %q, Got: %q", tc.expected, actual)
			}
		})
	}
}
