package main

import (
	"testing"
)

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{
			name:     "bytes only",
			bytes:    100,
			expected: "100 B",
		},
		{
			name:     "kilobytes",
			bytes:    2048,
			expected: "2.0 KB",
		},
		{
			name:     "megabytes",
			bytes:    5242880, // 5 MB
			expected: "5.0 MB",
		},
		{
			name:     "gigabytes",
			bytes:    3221225472, // 3 GB
			expected: "3.0 GB",
		},
		{
			name:     "zero bytes",
			bytes:    0,
			expected: "0 B",
		},
		{
			name:     "1 KB exactly",
			bytes:    1024,
			expected: "1.0 KB",
		},
		{
			name:     "fractional KB",
			bytes:    1536,
			expected: "1.5 KB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %s; expected %s", tt.bytes, result, tt.expected)
			}
		})
	}
}
