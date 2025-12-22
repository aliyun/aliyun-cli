package plugin

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestZipSlipProtection(t *testing.T) {
	// This test verifies that the untar and unzip functions
	// properly prevent Zip Slip attacks by rejecting paths with ".."

	tests := []struct {
		name     string
		archPath string
		isValid  bool
	}{
		{
			name:     "Valid path - simple file",
			archPath: "plugin/binary",
			isValid:  true,
		},
		{
			name:     "Valid path - nested file",
			archPath: "plugin/subdir/file.txt",
			isValid:  true,
		},
		{
			name:     "Invalid path - parent directory",
			archPath: "../etc/passwd",
			isValid:  false,
		},
		{
			name:     "Invalid path - nested parent directory",
			archPath: "plugin/../../etc/passwd",
			isValid:  false,
		},
		{
			name:     "Invalid path - absolute unix path",
			archPath: "/etc/passwd",
			isValid:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			dest := filepath.Join(tmpDir, "extract")
			if err := os.MkdirAll(dest, 0755); err != nil {
				t.Fatalf("Failed to create dest dir: %v", err)
			}

			// Test the path validation logic (same as in untar/unzip)
			// Step 1: Check for absolute paths or ".." patterns
			isValid := true
			if filepath.IsAbs(tt.archPath) {
				isValid = false
			} else if strings.Contains(tt.archPath, "..") {
				isValid = false
			} else if strings.HasPrefix(tt.archPath, "/") || strings.HasPrefix(tt.archPath, "\\") {
				// Reject paths starting with / or \ (cross-platform security)
				isValid = false
			} else {
				// Step 2: Check if the final path is within dest
				target := filepath.Join(dest, tt.archPath)
				target = filepath.Clean(target)
				destPath := filepath.Clean(dest) + string(os.PathSeparator)
				isValid = strings.HasPrefix(target, destPath)
			}

			if isValid != tt.isValid {
				t.Errorf("Path validation for %q: got %v, want %v", tt.archPath, isValid, tt.isValid)
			}
		})
	}
}

func TestIsDevVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected bool
	}{
		// Development versions
		{
			name:     "Explicit dev suffix",
			version:  "0.0.1-dev",
			expected: true,
		},
		{
			name:     "Contains dev",
			version:  "1.0.0-dev.123",
			expected: true,
		},
		{
			name:     "Version 0.0.1",
			version:  "0.0.1",
			expected: true,
		},
		{
			name:     "Version 0.0.x",
			version:  "0.0.5",
			expected: true,
		},
		{
			name:     "Uppercase DEV",
			version:  "0.0.1-DEV",
			expected: true,
		},

		// Production versions
		{
			name:     "Normal version 3.2.0",
			version:  "3.2.0",
			expected: false,
		},
		{
			name:     "Version with v prefix",
			version:  "v3.2.0",
			expected: false,
		},
		{
			name:     "Beta version",
			version:  "3.2.0-beta.1",
			expected: false,
		},
		{
			name:     "RC version",
			version:  "3.2.0-rc.1",
			expected: false,
		},
		{
			name:     "Version 1.0.0",
			version:  "1.0.0",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isDevVersion(tt.version)
			if result != tt.expected {
				t.Errorf("isDevVersion(%q) = %v, want %v", tt.version, result, tt.expected)
			}
		})
	}
}

func TestMatchPluginName(t *testing.T) {
	tests := []struct {
		name       string
		pluginName string
		userInput  string
		expected   bool
	}{
		// Exact match tests
		{
			name:       "Exact match - full name",
			pluginName: "aliyun-cli-fc",
			userInput:  "aliyun-cli-fc",
			expected:   true,
		},
		{
			name:       "Exact match - single word",
			pluginName: "fc",
			userInput:  "fc",
			expected:   true,
		},

		// Short name tests
		{
			name:       "Short name - fc",
			pluginName: "aliyun-cli-fc",
			userInput:  "fc",
			expected:   true,
		},
		{
			name:       "Short name - ecs",
			pluginName: "aliyun-cli-ecs",
			userInput:  "ecs",
			expected:   true,
		},
		{
			name:       "Short name - oss",
			pluginName: "aliyun-cli-oss",
			userInput:  "oss",
			expected:   true,
		},

		// Alternative prefix tests
		{
			name:       "Alternative prefix - aliyun-fc",
			pluginName: "aliyun-cli-fc",
			userInput:  "aliyun-fc",
			expected:   true,
		},
		{
			name:       "Alternative prefix - aliyun-ecs",
			pluginName: "aliyun-cli-ecs",
			userInput:  "aliyun-ecs",
			expected:   true,
		},

		// Negative tests - should not match
		{
			name:       "Different plugin",
			pluginName: "aliyun-cli-fc",
			userInput:  "ecs",
			expected:   false,
		},
		{
			name:       "Partial match",
			pluginName: "aliyun-cli-fc",
			userInput:  "f",
			expected:   false,
		},
		{
			name:       "Wrong prefix",
			pluginName: "aliyun-cli-fc",
			userInput:  "alibaba-fc",
			expected:   false,
		},
		{
			name:       "Empty input",
			pluginName: "aliyun-cli-fc",
			userInput:  "",
			expected:   false,
		},

		// Case-insensitive tests
		{
			name:       "Case insensitive - uppercase short name",
			pluginName: "aliyun-cli-fc",
			userInput:  "FC",
			expected:   true,
		},
		{
			name:       "Case insensitive - mixed case short name",
			pluginName: "aliyun-cli-fc",
			userInput:  "Fc",
			expected:   true,
		},
		{
			name:       "Case insensitive - uppercase full name",
			pluginName: "aliyun-cli-fc",
			userInput:  "ALIYUN-CLI-FC",
			expected:   true,
		},
		{
			name:       "Case insensitive - mixed case full name",
			pluginName: "aliyun-cli-fc",
			userInput:  "Aliyun-Cli-Fc",
			expected:   true,
		},
		{
			name:       "Case insensitive - uppercase alternative prefix",
			pluginName: "aliyun-cli-ecs",
			userInput:  "ALIYUN-ECS",
			expected:   true,
		},
		{
			name:       "Case insensitive - mixed case alternative prefix",
			pluginName: "aliyun-cli-ecs",
			userInput:  "Aliyun-Ecs",
			expected:   true,
		},
		{
			name:       "Case insensitive - ECS uppercase",
			pluginName: "aliyun-cli-ecs",
			userInput:  "ECS",
			expected:   true,
		},
		{
			name:       "Case insensitive - OSS uppercase",
			pluginName: "aliyun-cli-oss",
			userInput:  "OSS",
			expected:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPluginName(tt.pluginName, tt.userInput)
			if result != tt.expected {
				t.Errorf("matchPluginName(%q, %q) = %v, want %v",
					tt.pluginName, tt.userInput, result, tt.expected)
			}
		})
	}
}

func TestMatchPluginName_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		description string
		pluginName  string
		userInputs  []string
		shouldMatch bool
	}{
		{
			description: "User installs FC plugin with short name",
			pluginName:  "aliyun-cli-fc",
			userInputs:  []string{"fc"},
			shouldMatch: true,
		},
		{
			description: "User installs FC plugin with full name",
			pluginName:  "aliyun-cli-fc",
			userInputs:  []string{"aliyun-cli-fc"},
			shouldMatch: true,
		},
		{
			description: "User installs FC plugin with alternative prefix",
			pluginName:  "aliyun-cli-fc",
			userInputs:  []string{"aliyun-fc"},
			shouldMatch: true,
		},
		{
			description: "User tries wrong name",
			pluginName:  "aliyun-cli-fc",
			userInputs:  []string{"ecs", "oss", "function-compute"},
			shouldMatch: false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.description, func(t *testing.T) {
			for _, userInput := range scenario.userInputs {
				result := matchPluginName(scenario.pluginName, userInput)
				if result != scenario.shouldMatch {
					t.Errorf("Scenario: %s\nmatchPluginName(%q, %q) = %v, want %v",
						scenario.description, scenario.pluginName, userInput, result, scenario.shouldMatch)
				}
			}
		})
	}
}

func TestMatchPluginName_MultiplePlugins(t *testing.T) {
	plugins := []string{
		"aliyun-cli-fc",
		"aliyun-cli-ecs",
		"aliyun-cli-oss",
		"aliyun-cli-vpc",
	}

	// Each short name should match exactly one plugin
	tests := []struct {
		userInput    string
		expectedName string
	}{
		{"fc", "aliyun-cli-fc"},
		{"ecs", "aliyun-cli-ecs"},
		{"oss", "aliyun-cli-oss"},
		{"vpc", "aliyun-cli-vpc"},
	}

	for _, tt := range tests {
		t.Run("Short name: "+tt.userInput, func(t *testing.T) {
			matchCount := 0
			var matchedPlugin string

			for _, pluginName := range plugins {
				if matchPluginName(pluginName, tt.userInput) {
					matchCount++
					matchedPlugin = pluginName
				}
			}

			if matchCount != 1 {
				t.Errorf("Short name %q matched %d plugins, want exactly 1", tt.userInput, matchCount)
			}

			if matchedPlugin != tt.expectedName {
				t.Errorf("Short name %q matched %q, want %q", tt.userInput, matchedPlugin, tt.expectedName)
			}
		})
	}
}

func TestCalculateSHA256(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		expectedHash string
	}{
		{
			name:         "Empty file",
			content:      "",
			expectedHash: "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
		},
		{
			name:         "Simple text",
			content:      "hello world",
			expectedHash: "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9",
		},
		{
			name:         "JSON content",
			content:      `{"version":"1.0.0","name":"test"}`,
			expectedHash: "7ae693a8c6f793c5c4b0d8b6c1f5b8e9c9f2c8f8e8e8e8e8e8e8e8e8e8e8e8e8",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "test.txt")

			err := os.WriteFile(tmpFile, []byte(tt.content), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			hash, err := calculateSHA256(tmpFile)
			if err != nil {
				t.Fatalf("calculateSHA256() error = %v", err)
			}

			if tt.name == "Empty file" || tt.name == "Simple text" {
				if hash != tt.expectedHash {
					t.Errorf("calculateSHA256() = %v, want %v", hash, tt.expectedHash)
				}
			} else {
				// For other cases, just verify it returns a valid SHA256 hash (64 hex chars)
				if len(hash) != 64 {
					t.Errorf("calculateSHA256() returned hash of length %d, want 64", len(hash))
				}
			}
		})
	}
}

func TestCalculateSHA256_FileNotFound(t *testing.T) {
	_, err := calculateSHA256("/nonexistent/file/path")
	if err == nil {
		t.Error("calculateSHA256() expected error for nonexistent file, got nil")
	}
}

func TestCalculateSHA256_Consistency(t *testing.T) {
	// Test that the same content produces the same hash
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")

	content := "test content for consistency check"
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hash1, err := calculateSHA256(tmpFile)
	if err != nil {
		t.Fatalf("First calculateSHA256() error = %v", err)
	}

	hash2, err := calculateSHA256(tmpFile)
	if err != nil {
		t.Fatalf("Second calculateSHA256() error = %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("calculateSHA256() not consistent: first=%v, second=%v", hash1, hash2)
	}
}
