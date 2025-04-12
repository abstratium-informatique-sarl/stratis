package logging

import (
	"testing"
)

func TestShortenString(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{"abstratium.dev/someapp/package/subpackage", "a.s.p.subpackage"},
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
