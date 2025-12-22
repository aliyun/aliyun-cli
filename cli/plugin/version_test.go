package plugin

import (
	"testing"
)

func TestCompareVersion(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int // 1: v1 > v2, -1: v1 < v2, 0: v1 == v2
	}{
		// Basic comparisons
		{
			name:     "Equal versions",
			v1:       "3.2.0",
			v2:       "3.2.0",
			expected: 0,
		},
		{
			name:     "v1 greater than v2 (major)",
			v1:       "4.0.0",
			v2:       "3.2.0",
			expected: 1,
		},
		{
			name:     "v1 less than v2 (major)",
			v1:       "2.0.0",
			v2:       "3.2.0",
			expected: -1,
		},
		{
			name:     "v1 greater than v2 (minor)",
			v1:       "3.3.0",
			v2:       "3.2.0",
			expected: 1,
		},
		{
			name:     "v1 less than v2 (minor)",
			v1:       "3.1.0",
			v2:       "3.2.0",
			expected: -1,
		},
		{
			name:     "v1 greater than v2 (patch)",
			v1:       "3.2.1",
			v2:       "3.2.0",
			expected: 1,
		},
		{
			name:     "v1 less than v2 (patch)",
			v1:       "3.2.0",
			v2:       "3.2.1",
			expected: -1,
		},

		// With 'v' prefix
		{
			name:     "With v prefix - equal",
			v1:       "v3.2.0",
			v2:       "v3.2.0",
			expected: 0,
		},
		{
			name:     "With v prefix - v1 greater",
			v1:       "v3.3.0",
			v2:       "v3.2.0",
			expected: 1,
		},
		{
			name:     "Mixed prefix",
			v1:       "v3.2.0",
			v2:       "3.2.0",
			expected: 0,
		},

		// Pre-release versions (semver: pre-release < release)
		{
			name:     "Beta version vs stable",
			v1:       "3.2.0-beta.1",
			v2:       "3.2.0",
			expected: -1, // In semver, pre-release versions are LESS than the release version
		},
		{
			name:     "Different beta versions",
			v1:       "3.2.1-beta.1",
			v2:       "3.2.0",
			expected: 1,
		},

		// Real-world scenarios
		{
			name:     "Current CLI 3.2.2 vs Min 3.2.0",
			v1:       "3.2.2",
			v2:       "3.2.0",
			expected: 1,
		},
		{
			name:     "Current CLI 3.1.9 vs Min 3.2.0",
			v1:       "3.1.9",
			v2:       "3.2.0",
			expected: -1,
		},
		{
			name:     "Current CLI 4.0.0 vs Min 3.2.0",
			v1:       "4.0.0",
			v2:       "3.2.0",
			expected: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersion(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersion(%q, %q) = %d, want %d",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}

func TestCompareVersion_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{
			name:     "Empty strings",
			v1:       "",
			v2:       "",
			expected: 0,
		},
		{
			name:     "v1 empty",
			v1:       "",
			v2:       "3.2.0",
			expected: -1,
		},
		{
			name:     "v2 empty",
			v1:       "3.2.0",
			v2:       "",
			expected: 1,
		},
		{
			name:     "Single digit versions",
			v1:       "3",
			v2:       "2",
			expected: 1,
		},
		{
			name:     "Two digit versions",
			v1:       "3.2",
			v2:       "3.3",
			expected: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := compareVersion(tt.v1, tt.v2)
			if result != tt.expected {
				t.Errorf("compareVersion(%q, %q) = %d, want %d",
					tt.v1, tt.v2, result, tt.expected)
			}
		})
	}
}
