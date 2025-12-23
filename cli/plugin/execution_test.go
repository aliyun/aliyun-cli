package plugin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMatchPluginCommand(t *testing.T) {
	tests := []struct {
		name          string
		pluginCommand string
		userInput     string
		expected      bool
	}{
		// Exact match tests
		{
			name:          "Exact match - lowercase",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "aliyun-cli-fc",
			expected:      true,
		},
		{
			name:          "Exact match - case insensitive uppercase",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "ALIYUN-CLI-FC",
			expected:      true,
		},
		{
			name:          "Exact match - case insensitive mixed",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "Aliyun-Cli-Fc",
			expected:      true,
		},

		// Short name tests
		{
			name:          "Short name - lowercase",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "fc",
			expected:      true,
		},
		{
			name:          "Short name - uppercase",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "FC",
			expected:      true,
		},
		{
			name:          "Short name - mixed case",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "Fc",
			expected:      true,
		},
		{
			name:          "Short name - ecs",
			pluginCommand: "aliyun-cli-ecs",
			userInput:     "ecs",
			expected:      true,
		},
		{
			name:          "Short name - ECS",
			pluginCommand: "aliyun-cli-ecs",
			userInput:     "ECS",
			expected:      true,
		},

		// Negative tests
		{
			name:          "Different plugin",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "ecs",
			expected:      false,
		},
		{
			name:          "Partial match",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "f",
			expected:      false,
		},
		{
			name:          "Wrong prefix",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "alibaba-fc",
			expected:      false,
		},
		{
			name:          "Empty input",
			pluginCommand: "aliyun-cli-fc",
			userInput:     "",
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := matchPluginName(tt.pluginCommand, tt.userInput)
			if result != tt.expected {
				t.Errorf("matchPluginName(%q, %q) = %v, want %v",
					tt.pluginCommand, tt.userInput, result, tt.expected)
			}
		})
	}
}

func TestMatchPluginCommand_RealWorldScenarios(t *testing.T) {
	scenarios := []struct {
		description   string
		pluginCommand string
		userInputs    []string
		shouldMatch   bool
	}{
		{
			description:   "User executes FC plugin with short name",
			pluginCommand: "aliyun-cli-fc",
			userInputs:    []string{"fc", "FC", "Fc"},
			shouldMatch:   true,
		},
		{
			description:   "User executes FC plugin with full name",
			pluginCommand: "aliyun-cli-fc",
			userInputs:    []string{"aliyun-cli-fc", "ALIYUN-CLI-FC", "Aliyun-Cli-Fc"},
			shouldMatch:   true,
		},
		{
			description:   "User tries wrong name",
			pluginCommand: "aliyun-cli-fc",
			userInputs:    []string{"ecs", "oss", "vpc"},
			shouldMatch:   false,
		},
	}

	for _, scenario := range scenarios {
		t.Run(scenario.description, func(t *testing.T) {
			for _, userInput := range scenario.userInputs {
				result := matchPluginName(scenario.pluginCommand, userInput)
				if result != scenario.shouldMatch {
					t.Errorf("Scenario: %s\n  matchPluginName(%q, %q) = %v, want %v",
						scenario.description, scenario.pluginCommand, userInput, result, scenario.shouldMatch)
				}
			}
		})
	}
}

func TestFindLocalPlugin(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{rootDir: tmpDir}

	// Create a test manifest
	manifest := &LocalManifest{
		Plugins: map[string]LocalPlugin{
			"aliyun-cli-fc": {
				Name:    "aliyun-cli-fc",
				Command: "aliyun-cli-fc",
				Path:    "/path/to/fc",
			},
		},
	}
	if err := mgr.saveLocalManifest(manifest); err != nil {
		t.Fatalf("Failed to save manifest: %v", err)
	}

	tests := []struct {
		name      string
		command   string
		wantFound bool
		wantErr   bool
	}{
		{
			name:      "Plugin found - exact match",
			command:   "aliyun-cli-fc",
			wantFound: true,
			wantErr:   false,
		},
		{
			name:      "Plugin found - short name",
			command:   "fc",
			wantFound: true,
			wantErr:   false,
		},
		{
			name:      "Plugin not found",
			command:   "nonexistent",
			wantFound: false,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pluginName, plugin, err := mgr.findLocalPlugin(tt.command)

			if tt.wantErr {
				if err == nil {
					t.Errorf("findLocalPlugin() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("findLocalPlugin() unexpected error: %v", err)
				return
			}

			if tt.wantFound {
				if plugin == nil {
					t.Errorf("findLocalPlugin() expected plugin, got nil")
				} else if plugin.Command != "aliyun-cli-fc" {
					t.Errorf("findLocalPlugin() got plugin.Command = %q, want %q", plugin.Command, "aliyun-cli-fc")
				}
				if pluginName != "aliyun-cli-fc" {
					t.Errorf("findLocalPlugin() got pluginName = %q, want %q", pluginName, "aliyun-cli-fc")
				}
			} else {
				if plugin != nil {
					t.Errorf("findLocalPlugin() expected nil, got plugin: %+v", plugin)
				}
			}
		})
	}
}

func TestResolvePluginBinaryPath(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir := t.TempDir()
	pluginName := "test-plugin"
	binPath := filepath.Join(tmpDir, pluginName)

	// Create a test binary file
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\necho test"), 0755); err != nil {
		t.Fatalf("Failed to create test binary: %v", err)
	}

	tests := []struct {
		name     string
		plugin   *LocalPlugin
		wantErr  bool
		wantPath string
	}{
		{
			name: "Valid plugin path",
			plugin: &LocalPlugin{
				Name: pluginName,
				Path: tmpDir,
			},
			wantErr:  false,
			wantPath: binPath,
		},
		{
			name:     "Nil plugin",
			plugin:   nil,
			wantErr:  true,
			wantPath: "",
		},
		{
			name: "Non-existent binary",
			plugin: &LocalPlugin{
				Name: "nonexistent",
				Path: tmpDir,
			},
			wantErr:  true,
			wantPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := resolvePluginBinaryPath(tt.plugin)

			if tt.wantErr {
				if err == nil {
					t.Errorf("resolvePluginBinaryPath() expected error, got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("resolvePluginBinaryPath() unexpected error: %v", err)
				return
			}

			if path != tt.wantPath {
				t.Errorf("resolvePluginBinaryPath() = %q, want %q", path, tt.wantPath)
			}
		})
	}
}
