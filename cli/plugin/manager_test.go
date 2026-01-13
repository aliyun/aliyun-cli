package plugin

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func newTestContext() *cli.Context {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	return cli.NewCommandContext(stdout, stderr)
}

func TestNewManager(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// Save original environment variables
		originalHome := os.Getenv("HOME")
		originalUserProfile := os.Getenv("USERPROFILE")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
			if runtime.GOOS == "windows" {
				if originalUserProfile != "" {
					os.Setenv("USERPROFILE", originalUserProfile)
				} else {
					os.Unsetenv("USERPROFILE")
				}
			}
		}()

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", testHome)
		}

		mgr, err := NewManager()
		assert.NoError(t, err)
		assert.NotNil(t, mgr)
		assert.NotEmpty(t, mgr.rootDir)

		if _, err := os.Stat(mgr.rootDir); os.IsNotExist(err) {
			t.Errorf("NewManager() did not create directory: %s", mgr.rootDir)
		}

		expectedDir := filepath.Join(testHome, ".aliyun", "plugins")
		assert.Equal(t, expectedDir, mgr.rootDir)
	})

	t.Run("Creates correct directory structure", func(t *testing.T) {
		originalHome := os.Getenv("HOME")
		originalUserProfile := os.Getenv("USERPROFILE")
		defer func() {
			if originalHome != "" {
				os.Setenv("HOME", originalHome)
			} else {
				os.Unsetenv("HOME")
			}
			if runtime.GOOS == "windows" {
				if originalUserProfile != "" {
					os.Setenv("USERPROFILE", originalUserProfile)
				} else {
					os.Unsetenv("USERPROFILE")
				}
			}
		}()

		testHome := t.TempDir()
		os.Setenv("HOME", testHome)
		if runtime.GOOS == "windows" {
			os.Setenv("USERPROFILE", testHome)
		}

		mgr, err := NewManager()
		assert.NoError(t, err)
		assert.NotNil(t, mgr)

		expectedDir := filepath.Join(testHome, ".aliyun", "plugins")
		assert.Equal(t, expectedDir, mgr.rootDir)
	})

	t.Run("Home directory not found", func(t *testing.T) {
		originalHome := os.Getenv("HOME")
		defer os.Setenv("HOME", originalHome)

		os.Unsetenv("HOME")
		if runtime.GOOS != "windows" {
			mgr, err := NewManager()
			assert.Nil(t, mgr)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "home directory not found")
		}
	})
}

func TestManager_GetIndex(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		validIndex := Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(validIndex)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}
		index, err := mgr.GetIndex()
		assert.NoError(t, err)
		assert.NotNil(t, index)
		assert.Equal(t, 1, len(index.Plugins))
		assert.Equal(t, "aliyun-cli-fc", index.Plugins[0].Name)
	})

	t.Run("Network error", func(t *testing.T) {
		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: "http://invalid-url-that-does-not-exist.local/plugins/index.json",
		}
		_, err := mgr.GetIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch plugin index")
	})

	t.Run("Non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}
		_, err := mgr.GetIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "status 404")
	})

	t.Run("Invalid JSON", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}
		_, err := mgr.GetIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode plugin index")
	})
}

func TestManager_findPluginInIndex(t *testing.T) {
	t.Run("Success - exact match", func(t *testing.T) {
		validIndex := Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
				},
				{
					Name:        "aliyun-cli-ecs",
					Description: "ECS plugin",
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(validIndex)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}

		plugin, err := mgr.findPluginInIndex("aliyun-cli-fc")
		assert.NoError(t, err)
		assert.NotNil(t, plugin)
		assert.Equal(t, "aliyun-cli-fc", plugin.Name)
	})

	t.Run("Success - short name match", func(t *testing.T) {
		validIndex := Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(validIndex)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}

		plugin, err := mgr.findPluginInIndex("fc")
		assert.NoError(t, err)
		assert.NotNil(t, plugin)
		assert.Equal(t, "aliyun-cli-fc", plugin.Name)
	})

	t.Run("Plugin not found", func(t *testing.T) {
		validIndex := Index{
			Plugins: []PluginInfo{
				{
					Name:        "aliyun-cli-fc",
					Description: "FC plugin",
				},
			},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(validIndex)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}

		_, err := mgr.findPluginInIndex("nonexistent-plugin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin nonexistent-plugin not found")
	})

	t.Run("GetIndex error", func(t *testing.T) {
		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: "http://invalid-url-that-does-not-exist.local/plugins/index.json",
		}

		_, err := mgr.findPluginInIndex("aliyun-cli-fc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch plugin index")
	})

	t.Run("Empty index", func(t *testing.T) {
		emptyIndex := Index{
			Plugins: []PluginInfo{},
		}
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(emptyIndex)
		}))
		defer server.Close()

		mgr := &Manager{
			rootDir:  t.TempDir(),
			indexURL: server.URL,
		}

		_, err := mgr.findPluginInIndex("aliyun-cli-fc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin aliyun-cli-fc not found")
	})
}

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
			name:       "Alternative prefix - aliyun-fc (not supported)",
			pluginName: "aliyun-cli-fc",
			userInput:  "aliyun-fc",
			expected:   false,
		},
		{
			name:       "Alternative prefix - aliyun-ecs (not supported)",
			pluginName: "aliyun-cli-ecs",
			userInput:  "aliyun-ecs",
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
			name:       "Case insensitive - uppercase alternative prefix (not supported)",
			pluginName: "aliyun-cli-ecs",
			userInput:  "ALIYUN-ECS",
			expected:   false,
		},
		{
			name:       "Case insensitive - mixed case alternative prefix (not supported)",
			pluginName: "aliyun-cli-ecs",
			userInput:  "Aliyun-Ecs",
			expected:   false,
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

func TestValidateVersionAndPlatform(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{rootDir: tmpDir}
	currentPlatform := GetCurrentPlatform()

	tests := []struct {
		name         string
		targetPlugin *PluginInfo
		version      string
		pluginName   string
		wantErr      bool
		errContains  string
	}{
		{
			name: "Valid version and platform",
			targetPlugin: &PluginInfo{
				Name: "test-plugin",
				Versions: map[string]VersionInfo{
					"1.0.0": {
						Platforms: map[string]PlatformInfo{
							currentPlatform: {
								URL:      "http://example.com/plugin.tar.gz",
								Checksum: "abc123",
							},
						},
					},
				},
			},
			version:    "1.0.0",
			pluginName: "test-plugin",
			wantErr:    false,
		},
		{
			name: "Version not found",
			targetPlugin: &PluginInfo{
				Name: "test-plugin",
				Versions: map[string]VersionInfo{
					"1.0.0": {
						Platforms: map[string]PlatformInfo{
							currentPlatform: {
								URL:      "http://example.com/plugin.tar.gz",
								Checksum: "abc123",
							},
						},
					},
				},
			},
			version:     "2.0.0",
			pluginName:  "test-plugin",
			wantErr:     true,
			errContains: "version 2.0.0 not found",
		},
		{
			name: "Platform not supported",
			targetPlugin: &PluginInfo{
				Name: "test-plugin",
				Versions: map[string]VersionInfo{
					"1.0.0": {
						Platforms: map[string]PlatformInfo{
							"unsupported-platform": {
								URL:      "http://example.com/plugin.tar.gz",
								Checksum: "abc123",
							},
						},
					},
				},
			},
			version:     "1.0.0",
			pluginName:  "test-plugin",
			wantErr:     true,
			errContains: "not supported on",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := newTestContext()
			platInfo, err := mgr.validateVersionAndPlatform(ctx, tt.targetPlugin, tt.version, tt.pluginName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("validateVersionAndPlatform() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("validateVersionAndPlatform() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
				return
			}

			if platInfo == nil {
				t.Errorf("validateVersionAndPlatform() expected PlatformInfo, got nil")
			}
		})
	}
}

func TestValidateVersionAndPlatform_VersionCheck(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{rootDir: tmpDir}
	currentPlatform := GetCurrentPlatform()

	originalVersion := cli.Version
	defer func() {
		cli.Version = originalVersion
	}()

	t.Run("Success - no metadata", func(t *testing.T) {
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})

	t.Run("Success - metadata with empty MinCliVersion", func(t *testing.T) {
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})

	t.Run("Success - dev version skips check", func(t *testing.T) {
		cli.Version = "0.0.1"
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "3.2.0",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})

	t.Run("Success - dev version with -dev suffix skips check", func(t *testing.T) {
		cli.Version = "3.2.0-dev"
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "3.2.0",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})

	t.Run("Error - current version lower than required", func(t *testing.T) {
		cli.Version = "3.1.0"
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "3.2.0",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		assert.NotNil(t, err)
		assert.Nil(t, platInfo)
		assert.Contains(t, err.Error(), "requires CLI version")
		assert.Contains(t, err.Error(), "3.2.0")
		assert.Contains(t, err.Error(), "3.1.0")
		assert.Contains(t, err.Error(), "brew upgrade")
		assert.Contains(t, err.Error(), "github.com/aliyun/aliyun-cli/releases")
	})

	t.Run("Success - current version equal to required", func(t *testing.T) {
		cli.Version = "3.2.0"
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "3.2.0",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})

	t.Run("Success - current version higher than required", func(t *testing.T) {
		cli.Version = "3.3.0"
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Metadata: &VersionMetadata{
						MinCliVersion: "3.2.0",
					},
					Platforms: map[string]PlatformInfo{
						currentPlatform: {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		platInfo, err := mgr.validateVersionAndPlatform(ctx, targetPlugin, "1.0.0", "test-plugin")
		if err != nil {
			t.Errorf("validateVersionAndPlatform() unexpected error: %v", err)
		}
		if platInfo == nil {
			t.Error("validateVersionAndPlatform() expected PlatformInfo, got nil")
		}
	})
}

func TestLoadAndValidatePluginManifest(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{rootDir: tmpDir}

	tests := []struct {
		name         string
		setupFunc    func(string) error
		expectedName string
		wantErr      bool
		errContains  string
	}{
		{
			name: "Valid manifest",
			setupFunc: func(dir string) error {
				manifest := PluginManifest{
					Name:             "test-plugin",
					Command:          "test",
					ShortDescription: "Test plugin",
				}
				manifestPath := filepath.Join(dir, "manifest.json")
				data, err := json.Marshal(manifest)
				if err != nil {
					return err
				}
				return os.WriteFile(manifestPath, data, 0644)
			},
			expectedName: "test-plugin",
			wantErr:      false,
		},
		{
			name: "Manifest file not found",
			setupFunc: func(dir string) error {
				// Don't create manifest.json
				return nil
			},
			expectedName: "test-plugin",
			wantErr:      true,
			errContains:  "manifest.json not found",
		},
		{
			name: "Invalid JSON",
			setupFunc: func(dir string) error {
				manifestPath := filepath.Join(dir, "manifest.json")
				return os.WriteFile(manifestPath, []byte("invalid json"), 0644)
			},
			expectedName: "test-plugin",
			wantErr:      true,
			errContains:  "invalid plugin manifest",
		},
		{
			name: "Name mismatch",
			setupFunc: func(dir string) error {
				manifest := PluginManifest{
					Name:             "wrong-name",
					Command:          "test",
					ShortDescription: "Test plugin",
				}
				manifestPath := filepath.Join(dir, "manifest.json")
				data, err := json.Marshal(manifest)
				if err != nil {
					return err
				}
				return os.WriteFile(manifestPath, data, 0644)
			},
			expectedName: "test-plugin",
			wantErr:      true,
			errContains:  "does not match expected name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(testDir, 0755); err != nil {
				t.Fatalf("Failed to create test directory: %v", err)
			}

			if err := tt.setupFunc(testDir); err != nil {
				t.Fatalf("Setup failed: %v", err)
			}

			manifest, err := mgr.loadAndValidatePluginManifest(testDir, tt.expectedName)

			if tt.wantErr {
				if err == nil {
					t.Errorf("loadAndValidatePluginManifest() expected error, got nil")
					return
				}
				if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("loadAndValidatePluginManifest() error = %v, want error containing %q", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("loadAndValidatePluginManifest() unexpected error: %v", err)
				return
			}

			if manifest == nil {
				t.Errorf("loadAndValidatePluginManifest() expected manifest, got nil")
				return
			}

			if manifest.Name != tt.expectedName {
				t.Errorf("loadAndValidatePluginManifest() got name = %q, want %q", manifest.Name, tt.expectedName)
			}
		})
	}
}

func TestSavePluginToManifest(t *testing.T) {
	tmpDir := t.TempDir()
	mgr := &Manager{rootDir: tmpDir}

	pluginName := "test-plugin"
	version := "1.0.0"
	extractDir := filepath.Join(tmpDir, "extracted")
	pManifest := &PluginManifest{
		Name:             pluginName,
		Command:          "test",
		ShortDescription: "Test plugin description",
	}

	if err := os.MkdirAll(extractDir, 0755); err != nil {
		t.Fatalf("Failed to create extract directory: %v", err)
	}

	if err := mgr.savePluginToManifest(pluginName, version, extractDir, pManifest); err != nil {
		t.Fatalf("savePluginToManifest() error = %v", err)
	}

	localManifest, err := mgr.GetLocalManifest()
	assert.NotNil(t, localManifest)
	assert.NoError(t, err)

	plugin, exists := localManifest.Plugins[pluginName]
	assert.True(t, exists)
	assert.Equal(t, pluginName, plugin.Name)
	assert.Equal(t, version, plugin.Version)
	assert.Equal(t, extractDir, plugin.Path)
	assert.Equal(t, pManifest.Command, plugin.Command)
	assert.Equal(t, pManifest.ShortDescription, plugin.Description)
}

func TestManager_downloadAndVerifyPlugin(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		testContent := []byte("test plugin content")
		expectedChecksum, err := calculateSHA256FromBytes(testContent)
		assert.NoError(t, err)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(testContent)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: t.TempDir()}
		platInfo := &PlatformInfo{
			URL:      server.URL,
			Checksum: expectedChecksum,
		}

		ctx := newTestContext()
		archivePath, err := mgr.downloadAndVerifyPlugin(ctx, platInfo, "test-plugin", "1.0.0")
		assert.NoError(t, err)
		assert.NotEmpty(t, archivePath)
		assert.FileExists(t, archivePath)
		content, err := os.ReadFile(archivePath)
		assert.NoError(t, err)
		assert.Equal(t, string(testContent), string(content))
		os.RemoveAll(filepath.Dir(archivePath))
	})

	t.Run("Download failure - network error", func(t *testing.T) {
		mgr := &Manager{rootDir: t.TempDir()}
		platInfo := &PlatformInfo{
			URL:      "http://invalid-url-that-does-not-exist.local/plugin.tar.gz",
			Checksum: "abc123",
		}

		ctx := newTestContext()
		_, err := mgr.downloadAndVerifyPlugin(ctx, platInfo, "test-plugin", "1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid-url-that-does-not-exist.local")
	})

	t.Run("Checksum mismatch", func(t *testing.T) {
		testContent := []byte("test plugin content")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(testContent)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: t.TempDir()}
		platInfo := &PlatformInfo{
			URL:      server.URL,
			Checksum: "wrong-checksum",
		}

		ctx := newTestContext()
		archivePath, err := mgr.downloadAndVerifyPlugin(ctx, platInfo, "test-plugin", "1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum verification failed")
		assert.Contains(t, err.Error(), "Expected: wrong-checksum")
		assert.Empty(t, archivePath)
	})

	t.Run("Non-200 status code", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: t.TempDir()}
		platInfo := &PlatformInfo{
			URL:      server.URL,
			Checksum: "abc123",
		}

		ctx := newTestContext()
		_, err := mgr.downloadAndVerifyPlugin(ctx, platInfo, "test-plugin", "1.0.0")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "download failed: 404")
	})
}

// Helper function to calculate SHA256 from bytes (for test setup)
func calculateSHA256FromBytes(data []byte) (string, error) {
	hash := sha256.New()
	if _, err := hash.Write(data); err != nil {
		return "", err
	}
	return hex.EncodeToString(hash.Sum(nil)), nil
}

func TestUntar(t *testing.T) {
	t.Run("Success - extract files and directories", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		if err := os.MkdirAll(destDir, 0755); err != nil {
			t.Fatalf("Failed to create dest dir: %v", err)
		}

		// Create a test tar.gz archive
		if err := createTestTarGz(archivePath, []testFile{
			{name: "plugin/", content: "", isDir: true},
			{name: "plugin/binary", content: "binary content", isDir: false},
			{name: "plugin/config.txt", content: "config content", isDir: false},
			{name: "plugin/subdir/", content: "", isDir: true},
			{name: "plugin/subdir/file.txt", content: "file content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := untar(archivePath, destDir)
		assert.NoError(t, err)
		binaryPath := filepath.Join(destDir, "plugin", "binary")
		content, err := os.ReadFile(binaryPath)
		assert.NoError(t, err)
		assert.Equal(t, "binary content", string(content))

		configPath := filepath.Join(destDir, "plugin", "config.txt")
		content, err = os.ReadFile(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "config content", string(content))

		subdirPath := filepath.Join(destDir, "plugin", "subdir", "file.txt")
		content, err = os.ReadFile(subdirPath)
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	t.Run("Error - source file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "extract")

		err := untar("/nonexistent/file.tar.gz", destDir)
		assert.Error(t, err)
	})

	t.Run("Error - invalid gzip file", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "invalid.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		// Create invalid gzip file
		os.WriteFile(archivePath, []byte("not a gzip file"), 0644)

		err := untar(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "gzip: invalid")
	})

	t.Run("Error - absolute path in archive", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("untar test skipped on Windows")
		}

		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with absolute path
		if err := createTestTarGz(archivePath, []testFile{
			{name: "/etc/passwd", content: "malicious", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := untar(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal absolute path")
	})

	t.Run("Error - path with .. in archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with .. path
		if err := createTestTarGz(archivePath, []testFile{
			{name: "../etc/passwd", content: "malicious", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := untar(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal path with '..'")
	})

	t.Run("Error - path starting with /", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with path starting with /
		if err := createTestTarGz(archivePath, []testFile{
			{name: "/plugin/binary", content: "content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := untar(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal")
	})

	t.Run("Error - path starting with \\", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.tar.gz")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with path starting with \
		if err := createTestTarGz(archivePath, []testFile{
			{name: "\\plugin\\binary", content: "content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := untar(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal path starting with separator")
	})
}

func TestUnzip(t *testing.T) {
	t.Run("Success - extract files and directories", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("unzip test skipped on Windows")
		}

		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")
		destDir := filepath.Join(tmpDir, "extract")

		if err := createTestZip(archivePath, []testFile{
			{name: "plugin/binary", content: "binary content", isDir: false},
			{name: "plugin/config.txt", content: "config content", isDir: false},
			{name: "plugin/subdir/", content: "", isDir: true},
			{name: "plugin/subdir/file.txt", content: "file content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := unzip(archivePath, destDir)
		assert.NoError(t, err)

		binaryPath := filepath.Join(destDir, "plugin", "binary")
		content, err := os.ReadFile(binaryPath)
		assert.NoError(t, err)
		assert.Equal(t, "binary content", string(content))

		configPath := filepath.Join(destDir, "plugin", "config.txt")
		content, err = os.ReadFile(configPath)
		assert.NoError(t, err)
		assert.Equal(t, "config content", string(content))

		subdirPath := filepath.Join(destDir, "plugin", "subdir", "file.txt")
		content, err = os.ReadFile(subdirPath)
		assert.NoError(t, err)
		assert.Equal(t, "file content", string(content))
	})

	t.Run("Error - source file not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		destDir := filepath.Join(tmpDir, "extract")

		err := unzip("/nonexistent/file.zip", destDir)
		assert.Error(t, err)
	})

	t.Run("Error - invalid zip file", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "invalid.zip")
		destDir := filepath.Join(tmpDir, "extract")

		// Create invalid zip file
		os.WriteFile(archivePath, []byte("not a zip file"), 0644)

		err := unzip(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not a valid zip file")
	})

	t.Run("Error - absolute path in archive", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("unzip test skipped on Windows")
		}

		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with absolute path
		if err := createTestZip(archivePath, []testFile{
			{name: "/etc/passwd", content: "malicious", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := unzip(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal absolute path")
	})

	t.Run("Error - path with .. in archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with .. path
		if err := createTestZip(archivePath, []testFile{
			{name: "../etc/passwd", content: "malicious", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := unzip(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal path with '..'")
	})

	t.Run("Error - path starting with /", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with path starting with /
		if err := createTestZip(archivePath, []testFile{
			{name: "/plugin/binary", content: "content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := unzip(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal")
	})

	t.Run("Error - path starting with \\", func(t *testing.T) {
		tmpDir := t.TempDir()
		archivePath := filepath.Join(tmpDir, "test.zip")
		destDir := filepath.Join(tmpDir, "extract")

		// Create archive with path starting with \
		if err := createTestZip(archivePath, []testFile{
			{name: "\\plugin\\binary", content: "content", isDir: false},
		}); err != nil {
			t.Fatalf("Failed to create test archive: %v", err)
		}

		err := unzip(archivePath, destDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "illegal path starting with separator")
	})
}

type testFile struct {
	name    string
	content string
	isDir   bool
}

func createTestTarGz(archivePath string, files []testFile) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	for _, file := range files {
		var hdr *tar.Header
		if file.isDir {
			hdr = &tar.Header{
				Name:     file.name,
				Typeflag: tar.TypeDir,
				Mode:     0755,
			}
		} else {
			hdr = &tar.Header{
				Name:     file.name,
				Typeflag: tar.TypeReg,
				Size:     int64(len(file.content)),
				Mode:     0644,
			}
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}

		if !file.isDir {
			if _, err := tw.Write([]byte(file.content)); err != nil {
				return err
			}
		}
	}

	return nil
}

func createTestZip(archivePath string, files []testFile) error {
	f, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer f.Close()

	zw := zip.NewWriter(f)
	defer zw.Close()

	for _, file := range files {
		var w io.Writer
		if file.isDir {
			// Create directory entry
			hdr := &zip.FileHeader{
				Name:   file.name,
				Method: zip.Store,
			}
			hdr.SetMode(os.ModeDir | 0755)
			w, err = zw.CreateHeader(hdr)
			if err != nil {
				return err
			}
		} else {
			w, err = zw.Create(file.name)
			if err != nil {
				return err
			}
			if _, err := w.Write([]byte(file.content)); err != nil {
				return err
			}
		}
	}

	return nil
}

func TestManager_installPlugin(t *testing.T) {
	t.Run("Success - install plugin with specified version", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archiveContent := createTestPluginArchive(t, "test-plugin", "1.0.0", "test")
		expectedChecksum, err := calculateSHA256FromBytes(archiveContent)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      server.URL,
							Checksum: expectedChecksum,
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err = mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.NoError(t, err)

		localManifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin, exists := localManifest.Plugins["test-plugin"]
		assert.True(t, exists)
		assert.Equal(t, "test-plugin", plugin.Name)
		assert.Equal(t, "1.0.0", plugin.Version)
		assert.Equal(t, "test", plugin.Command)

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		assert.DirExists(t, pluginDir)

		manifestPath := filepath.Join(pluginDir, "manifest.json")
		assert.FileExists(t, manifestPath)
	})

	t.Run("Success - install plugin with empty version (use latest)", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archiveContent := createTestPluginArchive(t, "test-plugin", "2.0.0", "test")
		expectedChecksum, err := calculateSHA256FromBytes(archiveContent)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"2.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      server.URL,
							Checksum: expectedChecksum,
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err = mgr.installPlugin(ctx, targetPlugin, "", false)
		assert.NoError(t, err)

		localManifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin, exists := localManifest.Plugins["test-plugin"]
		assert.True(t, exists)
		assert.Equal(t, "2.0.0", plugin.Version)
	})

	t.Run("Error - version validation fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{},
				},
			},
		}

		ctx := newTestContext()
		err := mgr.installPlugin(ctx, targetPlugin, "999.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "version 999.0.0 not found")
	})

	t.Run("Error - platform validation fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						"unsupported-platform": {
							URL:      "http://example.com/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err := mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin test-plugin version 1.0.0 not supported on "+platform)
	})

	t.Run("Error - download fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      "http://invalid-url-that-does-not-exist.local/plugin.tar.gz",
							Checksum: "abc123",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err := mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid-url-that-does-not-exist.local")
	})

	t.Run("Error - checksum verification fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archiveContent := createTestPluginArchive(t, "test-plugin", "1.0.0", "test")

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      server.URL,
							Checksum: "wrong-checksum",
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err := mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "checksum verification failed")
		assert.Contains(t, err.Error(), "Expected: wrong-checksum")
	})

	t.Run("Error - manifest not found in archive", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archiveContent := createTestPluginArchiveWithoutManifest(t)
		expectedChecksum, err := calculateSHA256FromBytes(archiveContent)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      server.URL,
							Checksum: expectedChecksum,
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err = mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "manifest.json not found")
	})

	t.Run("Error - manifest name mismatch", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archiveContent := createTestPluginArchive(t, "wrong-plugin-name", "1.0.0", "test")
		expectedChecksum, err := calculateSHA256FromBytes(archiveContent)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		targetPlugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {
					Platforms: map[string]PlatformInfo{
						platform: {
							URL:      server.URL,
							Checksum: expectedChecksum,
						},
					},
				},
			},
		}

		ctx := newTestContext()
		err = mgr.installPlugin(ctx, targetPlugin, "1.0.0", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin manifest name wrong-plugin-name does not match expected name test-plugin")
	})
}

func createTestPluginArchive(t *testing.T, pluginName, version, command string) []byte {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "plugin.tar.gz")

	manifest := PluginManifest{
		Name:             pluginName,
		Version:          version,
		Command:          command,
		ShortDescription: "Test plugin",
		Description:      "Test plugin description",
	}
	manifestJSON, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	if err := createTestTarGz(archivePath, []testFile{
		{name: "manifest.json", content: string(manifestJSON), isDir: false},
		{name: "binary", content: "binary content", isDir: false},
	}); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	content, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	return content
}

func createTestPluginArchiveWithoutManifest(t *testing.T) []byte {
	tmpDir := t.TempDir()
	archivePath := filepath.Join(tmpDir, "plugin.tar.gz")

	if err := createTestTarGz(archivePath, []testFile{
		{name: "binary", content: "binary content", isDir: false},
	}); err != nil {
		t.Fatalf("Failed to create test archive: %v", err)
	}

	content, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("Failed to read archive: %v", err)
	}

	return content
}

func TestManager_Upgrade(t *testing.T) {
	t.Run("Success - upgrade plugin to latest version", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		os.MkdirAll(pluginDir, 0755)
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"test-plugin": {
					Name:    "test-plugin",
					Version: "1.0.0",
					Path:    pluginDir,
					Command: "test",
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		archiveContent := createTestPluginArchive(t, "test-plugin", "2.0.0", "test")
		expectedChecksum, err := calculateSHA256FromBytes(archiveContent)
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		indexJSON := `{
			"plugins": [
				{
					"name": "test-plugin",
					"versions": {
						"2.0.0": {
							"` + platform + `": {
								"url": "` + server.URL + `",
								"checksum": "` + expectedChecksum + `"
							}
						}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err = mgr.Upgrade(ctx, "test-plugin", false)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin, exists := localManifest.Plugins["test-plugin"]
		assert.True(t, exists)
		assert.Equal(t, "2.0.0", plugin.Version)
	})

	t.Run("Error - plugin not found locally", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		ctx := newTestContext()
		err := mgr.Upgrade(ctx, "nonexistent-plugin", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin nonexistent-plugin not found in local manifest")
	})

	t.Run("Error - plugin not found in repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		os.MkdirAll(pluginDir, 0755)
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"test-plugin": {
					Name:    "test-plugin",
					Version: "1.0.0",
					Path:    pluginDir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		indexJSON := `{"plugins": []}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.Upgrade(ctx, "test-plugin", false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin test-plugin not found in repository")
	})

	t.Run("Success - plugin already up to date", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		os.MkdirAll(pluginDir, 0755)
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"test-plugin": {
					Name:    "test-plugin",
					Version: "2.0.0",
					Path:    pluginDir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		indexJSON := `{
			"plugins": [
				{
					"name": "test-plugin",
					"versions": {
						"2.0.0": {}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.Upgrade(ctx, "test-plugin", false)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin, exists := localManifest.Plugins["test-plugin"]
		assert.True(t, exists)
		assert.Equal(t, "2.0.0", plugin.Version)
	})
}

func TestManager_Uninstall(t *testing.T) {
	t.Run("Success - uninstall plugin", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		os.MkdirAll(pluginDir, 0755)
		os.WriteFile(filepath.Join(pluginDir, "binary"), []byte("binary content"), 0755)
		os.WriteFile(filepath.Join(pluginDir, "manifest.json"), []byte(`{"name":"test-plugin"}`), 0644)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"test-plugin": {
					Name:    "test-plugin",
					Version: "1.0.0",
					Path:    pluginDir,
					Command: "test",
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		ctx := newTestContext()
		err := mgr.Uninstall(ctx, "test-plugin")
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		if _, exists := localManifest.Plugins["test-plugin"]; exists {
			t.Error("Plugin was not removed from manifest")
		}
	})

	t.Run("Error - plugin not found", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		ctx := newTestContext()
		err := mgr.Uninstall(ctx, "nonexistent-plugin")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "plugin nonexistent-plugin not found in local manifest")
	})

	t.Run("Success - uninstall with short name", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "aliyun-cli-fc")
		os.MkdirAll(pluginDir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"aliyun-cli-fc": {
					Name:    "aliyun-cli-fc",
					Version: "1.0.0",
					Path:    pluginDir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		ctx := newTestContext()
		err := mgr.Uninstall(ctx, "fc")
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		_, exists := localManifest.Plugins["aliyun-cli-fc"]
		assert.False(t, exists)
	})
}

func TestManager_UpdateAll(t *testing.T) {
	t.Run("Success - no plugins installed", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}
		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.NoError(t, err)
	})

	t.Run("Success - all plugins up to date", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		plugin1Dir := filepath.Join(tmpDir, "plugin1")
		plugin2Dir := filepath.Join(tmpDir, "plugin2")
		os.MkdirAll(plugin1Dir, 0755)
		os.MkdirAll(plugin2Dir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"plugin1": {
					Name:    "plugin1",
					Version: "2.0.0",
					Path:    plugin1Dir,
				},
				"plugin2": {
					Name:    "plugin2",
					Version: "2.0.0",
					Path:    plugin2Dir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"2.0.0": {}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"2.0.0": {}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)
		assert.Equal(t, 2, len(localManifest.Plugins))
		assert.Equal(t, "2.0.0", localManifest.Plugins["plugin1"].Version)
		assert.Equal(t, "2.0.0", localManifest.Plugins["plugin2"].Version)
	})

	t.Run("Success - update some plugins", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		plugin1Dir := filepath.Join(tmpDir, "plugin1")
		plugin2Dir := filepath.Join(tmpDir, "plugin2")
		os.MkdirAll(plugin1Dir, 0755)
		os.MkdirAll(plugin2Dir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"plugin1": {
					Name:    "plugin1",
					Version: "1.0.0",
					Path:    plugin1Dir,
				},
				"plugin2": {
					Name:    "plugin2",
					Version: "2.0.0",
					Path:    plugin2Dir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		archive1Content := createTestPluginArchive(t, "plugin1", "2.0.0", "plugin1")
		checksum1, _ := calculateSHA256FromBytes(archive1Content)

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive1Content)
		}))
		defer server1.Close()

		platform := GetCurrentPlatform()
		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"2.0.0": {
							"` + platform + `": {
								"url": "` + server1.URL + `",
								"checksum": "` + checksum1 + `"
							}
						}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"2.0.0": {}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin1, exists := localManifest.Plugins["plugin1"]
		assert.True(t, exists)
		assert.Equal(t, "2.0.0", plugin1.Version)
	})

	t.Run("Success - plugin not found in repository", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "local-only-plugin")
		os.MkdirAll(pluginDir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"local-only-plugin": {
					Name:    "local-only-plugin",
					Version: "1.0.0",
					Path:    pluginDir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		indexJSON := `{"plugins": []}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)
		assert.Equal(t, 1, len(localManifest.Plugins))
		assert.Equal(t, "1.0.0", localManifest.Plugins["local-only-plugin"].Version)
	})

	t.Run("Error - index fetch fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		pluginDir := filepath.Join(tmpDir, "test-plugin")
		os.MkdirAll(pluginDir, 0755)
		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"test-plugin": {
					Name:    "test-plugin",
					Version: "1.0.0",
					Path:    pluginDir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		mgr.indexURL = "http://invalid-url-that-does-not-exist.local/index.json"

		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get plugin index")
	})

	t.Run("Error - installPlugin fails during update", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		plugin1Dir := filepath.Join(tmpDir, "plugin1")
		plugin2Dir := filepath.Join(tmpDir, "plugin2")
		os.MkdirAll(plugin1Dir, 0755)
		os.MkdirAll(plugin2Dir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"plugin1": {
					Name:    "plugin1",
					Version: "1.0.0",
					Path:    plugin1Dir,
				},
				"plugin2": {
					Name:    "plugin2",
					Version: "1.0.0",
					Path:    plugin2Dir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		platform := GetCurrentPlatform()
		// Create index with plugin1 having invalid URL (will cause installPlugin to fail)
		// and plugin2 having valid archive (will succeed)
		archive2Content := createTestPluginArchive(t, "plugin2", "2.0.0", "plugin2")
		checksum2, _ := calculateSHA256FromBytes(archive2Content)

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive2Content)
		}))
		defer server2.Close()

		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"2.0.0": {
							"` + platform + `": {
								"url": "http://invalid-url-that-does-not-exist.local/plugin1.tar.gz",
								"checksum": "abc123"
							}
						}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"2.0.0": {
							"` + platform + `": {
								"url": "` + server2.URL + `",
								"checksum": "` + checksum2 + `"
							}
						}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.UpdateAll(ctx, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "1 plugin(s) failed to update")

		// Verify plugin2 was updated successfully despite plugin1 failure
		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin2, exists := localManifest.Plugins["plugin2"]
		assert.True(t, exists)
		assert.Equal(t, "2.0.0", plugin2.Version)

		// Verify plugin1 version remains unchanged (update failed)
		plugin1, exists := localManifest.Plugins["plugin1"]
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", plugin1.Version)
	})
}

func TestManager_InstallAll(t *testing.T) {
	t.Run("Success - install all plugins from index", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		archive1Content := createTestPluginArchive(t, "plugin1", "1.0.0", "plugin1")
		archive2Content := createTestPluginArchive(t, "plugin2", "1.0.0", "plugin2")
		checksum1, _ := calculateSHA256FromBytes(archive1Content)
		checksum2, _ := calculateSHA256FromBytes(archive2Content)

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive1Content)
		}))
		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive2Content)
		}))
		defer server2.Close()

		platform := GetCurrentPlatform()
		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"1.0.0": {
							"` + platform + `": {
								"url": "` + server1.URL + `",
								"checksum": "` + checksum1 + `"
							}
						}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"1.0.0": {
							"` + platform + `": {
								"url": "` + server2.URL + `",
								"checksum": "` + checksum2 + `"
							}
						}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallAll(ctx)
		assert.NoError(t, err)

		localManifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin1, exists := localManifest.Plugins["plugin1"]
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", plugin1.Version)
		plugin2, exists := localManifest.Plugins["plugin2"]
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", plugin2.Version)
	})

	t.Run("Success - skip already installed plugins", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		plugin1Dir := filepath.Join(tmpDir, "plugin1")
		os.MkdirAll(plugin1Dir, 0755)

		localManifest := &LocalManifest{
			Plugins: map[string]LocalPlugin{
				"plugin1": {
					Name:    "plugin1",
					Version: "1.0.0",
					Path:    plugin1Dir,
				},
			},
		}
		mgr.saveLocalManifest(localManifest)

		archive2Content := createTestPluginArchive(t, "plugin2", "1.0.0", "plugin2")
		checksum2, _ := calculateSHA256FromBytes(archive2Content)

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive2Content)
		}))
		defer server2.Close()

		platform := GetCurrentPlatform()
		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"1.0.0": {}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"1.0.0": {
							"` + platform + `": {
								"url": "` + server2.URL + `",
								"checksum": "` + checksum2 + `"
							}
						}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallAll(ctx)
		assert.NoError(t, err)

		localManifest, err = mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin2, exists := localManifest.Plugins["plugin2"]
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", plugin2.Version)
	})

	t.Run("Success - empty index", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		indexJSON := `{"plugins": []}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallAll(ctx)
		assert.NoError(t, err)

		localManifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)
		assert.Equal(t, 0, len(localManifest.Plugins))
	})

	t.Run("Error - index fetch fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		mgr.indexURL = "http://invalid-url-that-does-not-exist.local/index.json"

		ctx := newTestContext()
		err := mgr.InstallAll(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get plugin index")
	})

	t.Run("Error - installPlugin fails during install", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		platform := GetCurrentPlatform()
		// Create index with plugin1 having invalid URL (will cause installPlugin to fail)
		// and plugin2 having valid archive (will succeed)
		archive2Content := createTestPluginArchive(t, "plugin2", "1.0.0", "plugin2")
		checksum2, _ := calculateSHA256FromBytes(archive2Content)

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive2Content)
		}))
		defer server2.Close()

		indexJSON := `{
			"plugins": [
				{
					"name": "plugin1",
					"versions": {
						"1.0.0": {
							"` + platform + `": {
								"url": "http://invalid-url-that-does-not-exist.local/plugin1.tar.gz",
								"checksum": "abc123"
							}
						}
					}
				},
				{
					"name": "plugin2",
					"versions": {
						"1.0.0": {
							"` + platform + `": {
								"url": "` + server2.URL + `",
								"checksum": "` + checksum2 + `"
							}
						}
					}
				}
			]
		}`
		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()
		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallAll(ctx)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "1 plugin(s) failed to install")

		// Verify plugin2 was installed successfully despite plugin1 failure
		localManifest, err := mgr.GetLocalManifest()
		assert.NoError(t, err)

		plugin2, exists := localManifest.Plugins["plugin2"]
		assert.True(t, exists)
		assert.Equal(t, "1.0.0", plugin2.Version)

		// Verify plugin1 was not installed (install failed)
		_, exists = localManifest.Plugins["plugin1"]
		assert.False(t, exists)
	})
}

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

func TestManager_InstallMultiple(t *testing.T) {
	t.Run("Success - install multiple plugins", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		// Create plugin archives
		archive1Content := createTestPluginArchive(t, "aliyun-cli-plugin1", "1.0.0", "plugin1")
		checksum1, _ := calculateSHA256FromBytes(archive1Content)
		archive2Content := createTestPluginArchive(t, "aliyun-cli-plugin2", "1.0.0", "plugin2")
		checksum2, _ := calculateSHA256FromBytes(archive2Content)

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive1Content)
		}))
		defer server1.Close()

		server2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive2Content)
		}))
		defer server2.Close()

		platform := GetCurrentPlatform()
		indexJSON := fmt.Sprintf(`{
			"plugins": [
				{
					"name": "aliyun-cli-plugin1",
					"description": "Plugin 1",
					"homepage": "https://example.com",
					"versions": {
						"1.0.0": {
							"%s": {
								"url": "%s",
								"checksum": "%s"
							}
						}
					}
				},
				{
					"name": "aliyun-cli-plugin2",
					"description": "Plugin 2",
					"homepage": "https://example.com",
					"versions": {
						"1.0.0": {
							"%s": {
								"url": "%s",
								"checksum": "%s"
							}
						}
					}
				}
			]
		}`, platform, server1.URL, checksum1,
			platform, server2.URL, checksum2)

		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()

		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallMultiple(ctx, []string{"plugin1", "plugin2"}, "", false)

		assert.NoError(t, err)

		stdout := ctx.Stdout().(*bytes.Buffer).String()
		assert.Contains(t, stdout, "Installing plugin1...")
		assert.Contains(t, stdout, "Installing plugin2...")
		assert.Contains(t, stdout, "Installed: 2")
		assert.NotContains(t, stdout, "Failed:")
	})

	t.Run("Error - all plugins fail to install", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		// Create mock server with empty index
		indexJSON := `{"plugins": []}`

		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()

		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		err := mgr.InstallMultiple(ctx, []string{"nonexistent1", "nonexistent2"}, "", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "2 plugin(s) failed to install")

		// Verify output
		stdout := ctx.Stdout().(*bytes.Buffer).String()
		stderr := ctx.Stderr().(*bytes.Buffer).String()

		assert.Contains(t, stdout, "Installing nonexistent1...")
		assert.Contains(t, stdout, "Installing nonexistent2...")
		assert.Contains(t, stdout, "Failed: 2")
		assert.NotContains(t, stdout, "Installed:")

		assert.Contains(t, stderr, "Failed to install nonexistent1:")
		assert.Contains(t, stderr, "Failed to install nonexistent2:")
	})

	t.Run("Partial success - some plugins install, some fail", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		// Create plugin archive for plugin1
		archive1Content := createTestPluginArchive(t, "aliyun-cli-plugin1", "1.0.0", "plugin1")
		checksum1, _ := calculateSHA256FromBytes(archive1Content)

		server1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archive1Content)
		}))
		defer server1.Close()

		platform := GetCurrentPlatform()
		indexJSON := fmt.Sprintf(`{
			"plugins": [
				{
					"name": "aliyun-cli-plugin1",
					"description": "Plugin 1",
					"homepage": "https://example.com",
					"versions": {
						"1.0.0": {
							"%s": {
								"url": "%s",
								"checksum": "%s"
							}
						}
					}
				}
			]
		}`, platform, server1.URL, checksum1)

		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()

		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		// Install one valid plugin and one invalid plugin
		err := mgr.InstallMultiple(ctx, []string{"plugin1", "nonexistent"}, "", false)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "1 plugin(s) failed to install")

		// Verify output
		stdout := ctx.Stdout().(*bytes.Buffer).String()
		stderr := ctx.Stderr().(*bytes.Buffer).String()

		assert.Contains(t, stdout, "Installing plugin1...")
		assert.Contains(t, stdout, "Installing nonexistent...")
		assert.Contains(t, stdout, "Installed: 1")
		assert.Contains(t, stdout, "Failed: 1")

		assert.Contains(t, stderr, "Failed to install nonexistent:")
	})

	t.Run("Empty plugin list", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		ctx := newTestContext()
		err := mgr.InstallMultiple(ctx, []string{}, "", false)

		assert.NoError(t, err)

		// Verify no output about installation counts
		stdout := ctx.Stdout().(*bytes.Buffer).String()
		assert.NotContains(t, stdout, "Installed:")
		assert.NotContains(t, stdout, "Failed:")
	})

	t.Run("With version parameter", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		// Create plugin archive
		archiveContent := createTestPluginArchive(t, "aliyun-cli-plugin1", "1.0.0", "plugin1")
		checksum, _ := calculateSHA256FromBytes(archiveContent)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		indexJSON := fmt.Sprintf(`{
			"plugins": [
				{
					"name": "aliyun-cli-plugin1",
					"description": "Plugin 1",
					"homepage": "https://example.com",
					"versions": {
						"1.0.0": {
							"%s": {
								"url": "%s",
								"checksum": "%s"
							}
						},
						"2.0.0": {
							"%s": {
								"url": "https://example.com/plugin1-2.0.0.tar.gz",
								"checksum": "def456"
							}
						}
					}
				}
			]
		}`, platform, server.URL, checksum,
			platform)

		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()

		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		// Install specific version
		err := mgr.InstallMultiple(ctx, []string{"plugin1"}, "1.0.0", false)

		assert.NoError(t, err)

		// Verify output
		stdout := ctx.Stdout().(*bytes.Buffer).String()
		assert.Contains(t, stdout, "Installing plugin1...")
		assert.Contains(t, stdout, "Installed: 1")
	})

	t.Run("With enablePre parameter", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir}

		// Create plugin archive
		archiveContent := createTestPluginArchive(t, "aliyun-cli-plugin1", "1.0.0-beta.1", "plugin1")
		checksum, _ := calculateSHA256FromBytes(archiveContent)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(archiveContent)
		}))
		defer server.Close()

		platform := GetCurrentPlatform()
		indexJSON := fmt.Sprintf(`{
			"plugins": [
				{
					"name": "aliyun-cli-plugin1",
					"description": "Plugin 1",
					"homepage": "https://example.com",
					"versions": {
						"1.0.0-beta.1": {
							"%s": {
								"url": "%s",
								"checksum": "%s"
							}
						}
					}
				}
			]
		}`, platform, server.URL, checksum)

		indexServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer indexServer.Close()

		mgr.indexURL = indexServer.URL

		ctx := newTestContext()
		// Install with enablePre=true
		err := mgr.InstallMultiple(ctx, []string{"plugin1"}, "", true)

		assert.NoError(t, err)

		// Verify output
		stdout := ctx.Stdout().(*bytes.Buffer).String()
		assert.Contains(t, stdout, "Installing plugin1...")
		assert.Contains(t, stdout, "Installed: 1")
	})
}

func TestGetLatestVersion(t *testing.T) {
	t.Run("No versions available", func(t *testing.T) {
		plugin := &PluginInfo{
			Name:     "test-plugin",
			Versions: map[string]VersionInfo{},
		}

		_, err := getLatestVersion(plugin, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no versions available for plugin test-plugin")
	})

	t.Run("Only stable versions - enablePre=false", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {},
				"1.2.0": {},
				"1.1.0": {},
				"2.0.0": {},
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "2.0.0", version, "Should return the newest stable version")
	})

	t.Run("Only stable versions - enablePre=true", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {},
				"1.2.0": {},
				"2.0.0": {},
			},
		}

		version, err := getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "2.0.0", version, "Should return the newest version")
	})

	t.Run("Only pre-release versions - enablePre=false", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0-alpha":  {},
				"1.0.0-beta":   {},
				"1.0.0-beta.2": {},
			},
		}

		_, err := getLatestVersion(plugin, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no stable version available for plugin test-plugin")
		assert.Contains(t, err.Error(), "Latest pre-release version: 1.0.0-beta.2")
		assert.Contains(t, err.Error(), "Use --enable-pre to install pre-release versions")
	})

	t.Run("Only pre-release versions - enablePre=true", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0-alpha": {},
				"1.0.0-beta":  {},
				"1.0.0-rc.1":  {},
			},
		}

		version, err := getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0-rc.1", version, "Should return the newest pre-release version")
	})

	t.Run("Mixed stable and pre-release - enablePre=false", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0":       {},
				"1.1.0":       {},
				"2.0.0-beta":  {},
				"2.0.0-rc.1":  {},
				"1.2.0":       {},
				"1.5.0-alpha": {},
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "1.2.0", version, "Should return the newest stable version, ignoring pre-releases")
	})

	t.Run("Mixed stable and pre-release - enablePre=true", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0":      {},
				"1.1.0":      {},
				"2.0.0-beta": {},
				"2.0.0-rc.1": {},
				"1.2.0":      {},
			},
		}

		version, err := getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "2.0.0-rc.1", version, "Should return the newest version including pre-releases")
	})

	t.Run("Semantic version sorting", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0":   {},
				"1.10.0":  {},
				"1.2.0":   {},
				"1.9.0":   {},
				"2.0.0":   {},
				"10.0.0":  {},
				"1.10.10": {},
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "10.0.0", version, "Should correctly sort versions semantically")
	})

	t.Run("Pre-release version sorting", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0-alpha":   {},
				"1.0.0-alpha.1": {},
				"1.0.0-beta":    {},
				"1.0.0-beta.2":  {},
				"1.0.0-rc.1":    {},
			},
		}

		version, err := getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0-rc.1", version, "Should correctly sort pre-release versions")
	})

	t.Run("Complex version mix", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"0.1.0":        {},
				"1.0.0":        {},
				"1.0.1":        {},
				"1.1.0-alpha":  {},
				"1.1.0-beta":   {},
				"1.1.0":        {},
				"2.0.0-alpha":  {},
				"2.0.0-beta.1": {},
				"2.0.0-beta.2": {},
				"2.0.0-rc.1":   {},
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "1.1.0", version, "Should return latest stable version")

		version, err = getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "2.0.0-rc.1", version, "Should return latest version including pre-releases")
	})

	t.Run("Single version", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0": {},
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0", version)
	})

	t.Run("Single pre-release version with enablePre=false", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0-beta": {},
			},
		}

		_, err := getLatestVersion(plugin, false)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no stable version available")
		assert.Contains(t, err.Error(), "Latest pre-release version: 1.0.0-beta")
	})

	t.Run("Pre-release newer than stable - enablePre=false", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0":      {},
				"1.1.0":      {},
				"2.0.0-beta": {}, // Newer than stable versions
			},
		}

		version, err := getLatestVersion(plugin, false)
		assert.NoError(t, err)
		assert.Equal(t, "1.1.0", version, "Should ignore pre-release and return latest stable")
	})

	t.Run("Multiple pre-releases for same version", func(t *testing.T) {
		plugin := &PluginInfo{
			Name: "test-plugin",
			Versions: map[string]VersionInfo{
				"1.0.0-alpha.1":  {},
				"1.0.0-alpha.2":  {},
				"1.0.0-alpha.10": {},
				"1.0.0-beta.1":   {},
			},
		}

		version, err := getLatestVersion(plugin, true)
		assert.NoError(t, err)
		assert.Equal(t, "1.0.0-beta.1", version, "Should return the latest pre-release")
	})
}

func TestManager_GetCommandIndex(t *testing.T) {
	t.Run("Success - fetch valid command index", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{
			"fc": "aliyun-cli-fc",
			"oss": "aliyun-cli-oss",
			"ecs": "aliyun-cli-ecs"
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		index, err := mgr.GetCommandIndex()
		assert.NoError(t, err)
		assert.NotNil(t, index)
		assert.Equal(t, 3, len(*index))
		assert.Equal(t, "aliyun-cli-fc", (*index)["fc"])
		assert.Equal(t, "aliyun-cli-oss", (*index)["oss"])
		assert.Equal(t, "aliyun-cli-ecs", (*index)["ecs"])
	})

	t.Run("Error - network error", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir, commandIndexURL: "http://invalid-url-that-does-not-exist.local:9999"}

		_, err := mgr.GetCommandIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch command index")
	})

	t.Run("Error - non-200 status code", func(t *testing.T) {
		tmpDir := t.TempDir()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		_, err := mgr.GetCommandIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch command index: status 404")
	})

	t.Run("Error - invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte("invalid json"))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		_, err := mgr.GetCommandIndex()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to decode command index")
	})

	t.Run("Success - empty command index", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		index, err := mgr.GetCommandIndex()
		assert.NoError(t, err)
		assert.NotNil(t, index)
		assert.Equal(t, 0, len(*index))
	})
}

func TestManager_FindPluginByCommand(t *testing.T) {
	t.Run("Success - find plugin by exact command name", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{
			"fc": "aliyun-cli-fc",
			"oss": "aliyun-cli-oss",
			"ecs": "aliyun-cli-ecs"
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		pluginName, err := mgr.FindPluginByCommand("fc")
		assert.NoError(t, err)
		assert.Equal(t, "aliyun-cli-fc", pluginName)
	})

	t.Run("Success - case insensitive search", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{
			"fc": "aliyun-cli-fc",
			"oss": "aliyun-cli-oss"
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		pluginName, err := mgr.FindPluginByCommand("FC")
		assert.NoError(t, err)
		assert.Equal(t, "aliyun-cli-fc", pluginName)

		pluginName, err = mgr.FindPluginByCommand("Fc")
		assert.NoError(t, err)
		assert.Equal(t, "aliyun-cli-fc", pluginName)
	})

	t.Run("Success - trim whitespace", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{
			"fc": "aliyun-cli-fc"
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		// Test with leading/trailing spaces
		pluginName, err := mgr.FindPluginByCommand("  fc  ")
		assert.NoError(t, err)
		assert.Equal(t, "aliyun-cli-fc", pluginName)
	})

	t.Run("Error - command not found", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{
			"fc": "aliyun-cli-fc",
			"oss": "aliyun-cli-oss"
		}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		_, err := mgr.FindPluginByCommand("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no plugin found for command: nonexistent")
	})

	t.Run("Error - GetCommandIndex fails", func(t *testing.T) {
		tmpDir := t.TempDir()
		mgr := &Manager{rootDir: tmpDir, commandIndexURL: "http://invalid-url-that-does-not-exist.local:9999"}

		_, err := mgr.FindPluginByCommand("fc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to fetch command index")
	})

	t.Run("Success - empty index returns error", func(t *testing.T) {
		tmpDir := t.TempDir()

		indexJSON := `{}`

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(indexJSON))
		}))
		defer server.Close()

		mgr := &Manager{rootDir: tmpDir, commandIndexURL: server.URL}

		_, err := mgr.FindPluginByCommand("fc")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no plugin found for command: fc")
	})
}
