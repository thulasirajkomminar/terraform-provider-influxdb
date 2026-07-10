package provider

import "testing"

func TestNormalizeLegacyTimestamp(t *testing.T) {
	tests := map[string]struct {
		input string
		want  string
	}{
		"legacy UTC": {
			input: "2024-05-01 10:30:45 +0000 UTC",
			want:  "2024-05-01T10:30:45Z",
		},
		"legacy with nanoseconds": {
			input: "2024-05-01 10:30:45.123456789 +0000 UTC",
			want:  "2024-05-01T10:30:45Z",
		},
		"legacy with offset": {
			input: "2024-05-01 10:30:45 +0200 CEST",
			want:  "2024-05-01T10:30:45+02:00",
		},
		"already RFC3339": {
			input: "2024-05-01T10:30:45Z",
			want:  "2024-05-01T10:30:45Z",
		},
		"unrecognized value passes through": {
			input: "not-a-timestamp",
			want:  "not-a-timestamp",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			if got := normalizeLegacyTimestamp(test.input); got != test.want {
				t.Errorf("normalizeLegacyTimestamp(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}
