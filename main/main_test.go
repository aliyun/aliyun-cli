package main

import (
	"testing"
)

func TestMain(m *testing.M) {
	Main([]string{})
}

func TestParseInSecure(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected bool
	}{
		{
			name:     "Insecure flag present",
			args:     []string{"--insecure"},
			expected: true,
		},
		{
			name:     "Insecure flag with value",
			args:     []string{"--insecure", "true"},
			expected: true,
		},
		{
			name:     "Insecure flag absent",
			args:     []string{"--secure"},
			expected: false,
		},
		{
			name:     "Empty args",
			args:     []string{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := ParseInSecure(tt.args)
			if result != tt.expected {
				t.Errorf("ParseInSecure(%v) = %v; want %v", tt.args, result, tt.expected)
			}
		})
	}
}
