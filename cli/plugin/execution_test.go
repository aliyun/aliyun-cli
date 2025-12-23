package plugin

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
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
			_, plugin, err := mgr.findLocalPlugin(tt.command)

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
			} else {
				if plugin != nil {
					t.Errorf("findLocalPlugin() expected nil, got plugin: %+v", plugin)
				}
			}
		})
	}
}

func TestResolvePluginBinaryPath(t *testing.T) {
	tmpDir := t.TempDir()
	pluginName := "test-plugin"
	binPath := filepath.Join(tmpDir, pluginName)

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

func TestAdjustPluginArgs(t *testing.T) {
	t.Run("Adjust plugin-help to --help", func(t *testing.T) {
		args := []string{"plugin-help"}
		adjusted := adjustPluginArgs(args)
		if len(adjusted) != 1 || adjusted[0] != "--help" {
			t.Errorf("adjustPluginArgs(%v) = %v, want [--help]", args, adjusted)
		}
	})

	t.Run("Keep other args unchanged", func(t *testing.T) {
		args := []string{"describe-regions", "--region-id", "cn-hangzhou"}
		adjusted := adjustPluginArgs(args)
		if len(adjusted) != len(args) {
			t.Errorf("adjustPluginArgs(%v) length = %d, want %d", args, len(adjusted), len(args))
		}
		for i := range args {
			if adjusted[i] != args[i] {
				t.Errorf("adjustPluginArgs(%v)[%d] = %q, want %q", args, i, adjusted[i], args[i])
			}
		}
	})

	t.Run("Empty args unchanged", func(t *testing.T) {
		args := []string{}
		adjusted := adjustPluginArgs(args)
		if len(adjusted) != 0 {
			t.Errorf("adjustPluginArgs(%v) = %v, want []", args, adjusted)
		}
	})

	t.Run("plugin-help with additional args", func(t *testing.T) {
		args := []string{"plugin-help", "--verbose"}
		adjusted := adjustPluginArgs(args)
		if len(adjusted) != 1 || adjusted[0] != "--help" {
			t.Errorf("adjustPluginArgs(%v) = %v, want [--help]", args, adjusted)
		}
	})
}

func TestRunPluginCommand(t *testing.T) {
	t.Run("Empty binary path", func(t *testing.T) {
		err := runPluginCommand("", []string{}, os.Stdout, os.Stderr, os.Environ())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "binary path is empty")
	})

	t.Run("Non-existent binary", func(t *testing.T) {
		nonExistentPath := "/non/existent/binary/path"
		err := runPluginCommand(nonExistentPath, []string{}, os.Stdout, os.Stderr, os.Environ())
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin execution failed")
	})

	t.Run("Success with test script", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test-plugin")

		scriptContent := "#!/bin/sh\necho 'test plugin output'\nexit 0\n"
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		err := runPluginCommand(scriptPath, []string{}, os.Stdout, os.Stderr, os.Environ())
		assert.NoError(t, err)
		if err != nil {
			t.Errorf("runPluginCommand() with valid script unexpected error: %v", err)
		}
	})

	t.Run("Success with test script and args", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		tmpDir := t.TempDir()
		scriptPath := filepath.Join(tmpDir, "test-plugin")

		scriptContent := "#!/bin/sh\necho \"args: $@\"\nexit 0\n"
		if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
			t.Fatalf("Failed to create test script: %v", err)
		}

		err := runPluginCommand(scriptPath, []string{"arg1", "arg2"}, os.Stdout, os.Stderr, os.Environ())
		assert.NoError(t, err)
		if err != nil {
			t.Errorf("runPluginCommand() with valid script and args unexpected error: %v", err)
		}
	})
}

func TestExecutePlugin(t *testing.T) {
	originalHome := os.Getenv("HOME")
	defer os.Setenv("HOME", originalHome)

	t.Run("Plugin not found", func(t *testing.T) {
		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		os.WriteFile(manifestPath, []byte(`{"plugins":{}}`), 0644)

		ok, err := ExecutePlugin("nonexistent-plugin", []string{}, nil)
		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("Plugin found and executed successfully", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		// Create plugin directory and binary
		pluginDir := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-test")
		os.MkdirAll(pluginDir, 0755)
		binPath := filepath.Join(pluginDir, "aliyun-cli-test")
		scriptContent := "#!/bin/sh\necho 'test plugin output'\nexit 0\n"
		os.WriteFile(binPath, []byte(scriptContent), 0755)

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		manifestJSON := `{"plugins":{"aliyun-cli-test":{"name":"aliyun-cli-test","version":"1.0.0","description":"Test plugin","path":"` + pluginDir + `","command":"test"}}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		ok, err := ExecutePlugin("test", []string{}, ctx)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Contains(t, stdout.String(), "test plugin output")
	})

	t.Run("Plugin binary path resolution fails", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-test")
		os.MkdirAll(pluginDir, 0755)

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		manifestJSON := `{"plugins":{"aliyun-cli-test":{"name":"aliyun-cli-test","version":"1.0.0","description":"Test plugin","path":"` + pluginDir + `","command":"test"}}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		ok, err := ExecutePlugin("test", []string{}, ctx)
		assert.Error(t, err)
		assert.True(t, ok) // Plugin was found, but binary path resolution failed
		assert.Contains(t, err.Error(), "failed to resolve plugin binary path")
	})

	t.Run("Plugin execution with args", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-test")
		os.MkdirAll(pluginDir, 0755)
		binPath := filepath.Join(pluginDir, "aliyun-cli-test")
		scriptContent := "#!/bin/sh\necho \"args: $@\"\nexit 0\n"
		os.WriteFile(binPath, []byte(scriptContent), 0755)

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		manifestJSON := `{"plugins":{"aliyun-cli-test":{"name":"aliyun-cli-test","version":"1.0.0","description":"Test plugin","path":"` + pluginDir + `","command":"test"}}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		stdout := new(bytes.Buffer)
		stderr := new(bytes.Buffer)
		ctx := cli.NewCommandContext(stdout, stderr)

		ok, err := ExecutePlugin("test", []string{"arg1", "arg2"}, ctx)
		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Contains(t, stdout.String(), "args: arg1 arg2")
	})

	t.Run("Plugin execution with plugin-help subcommand", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("shell script test skipped on Windows")
		}

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)

		pluginDir := filepath.Join(testHome, ".aliyun", "plugins", "aliyun-cli-test")
		os.MkdirAll(pluginDir, 0755)
		binPath := filepath.Join(pluginDir, "aliyun-cli-test")
		scriptContent := "#!/bin/sh\nif [ \"$1\" = \"--help\" ]; then echo 'plugin help'; exit 0; fi\nexit 1\n"
		os.WriteFile(binPath, []byte(scriptContent), 0755)

		manifestPath := filepath.Join(testHome, ".aliyun", "plugins", "manifest.json")
		os.MkdirAll(filepath.Dir(manifestPath), 0755)
		manifestJSON := `{"plugins":{"aliyun-cli-test":{"name":"aliyun-cli-test","version":"1.0.0","description":"Test plugin","path":"` + pluginDir + `","command":"test"}}}`
		os.WriteFile(manifestPath, []byte(manifestJSON), 0644)

		// Test that plugin-help is converted to --help
		ok, err := ExecutePlugin("test", []string{"plugin-help"}, nil)
		assert.NoError(t, err)
		assert.True(t, ok)
	})

	t.Run("Manager creation failure", func(t *testing.T) {
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		// Set HOME to a non-existent path - NewManager should still work
		os.Setenv("HOME", "")

		ok, err := ExecutePlugin("test", []string{}, nil)
		assert.NoError(t, err)
		assert.False(t, ok)
		// When manager creation fails, ExecutePlugin returns (false, nil)
		if ok {
			t.Error("ExecutePlugin() with invalid HOME expected ok=false, got ok=true")
		}
	})
}
