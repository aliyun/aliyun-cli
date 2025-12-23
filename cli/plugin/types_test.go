package plugin

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestVersionInfo_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name          string
		jsonData      string
		wantMetadata  *VersionMetadata
		wantPlatforms map[string]PlatformInfo
		expectError   bool
	}{
		{
			name: "With metadata and platforms",
			jsonData: `{
				"metadata": {
					"minCliVersion": "3.2.0"
				},
				"darwin/amd64": {
					"url": "https://example.com/plugin.tar.gz",
					"checksum": "abc123"
				},
				"linux/amd64": {
					"url": "https://example.com/plugin-linux.tar.gz",
					"checksum": "def456"
				}
			}`,
			wantMetadata: &VersionMetadata{
				MinCliVersion: "3.2.0",
			},
			wantPlatforms: map[string]PlatformInfo{
				"darwin/amd64": {
					URL:      "https://example.com/plugin.tar.gz",
					Checksum: "abc123",
				},
				"linux/amd64": {
					URL:      "https://example.com/plugin-linux.tar.gz",
					Checksum: "def456",
				},
			},
			expectError: false,
		},
		{
			name: "Without metadata",
			jsonData: `{
				"darwin/amd64": {
					"url": "https://example.com/plugin.tar.gz",
					"checksum": "abc123"
				}
			}`,
			wantMetadata: nil,
			wantPlatforms: map[string]PlatformInfo{
				"darwin/amd64": {
					URL:      "https://example.com/plugin.tar.gz",
					Checksum: "abc123",
				},
			},
			expectError: false,
		},
		{
			name: "Only metadata",
			jsonData: `{
				"metadata": {
					"minCliVersion": "3.0.0"
				}
			}`,
			wantMetadata: &VersionMetadata{
				MinCliVersion: "3.0.0",
			},
			wantPlatforms: map[string]PlatformInfo{},
			expectError:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var verInfo VersionInfo
			err := json.Unmarshal([]byte(tt.jsonData), &verInfo)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.wantMetadata == nil {
				if verInfo.Metadata != nil {
					t.Errorf("Expected nil metadata, got %+v", verInfo.Metadata)
				}
			} else {
				if verInfo.Metadata == nil {
					t.Errorf("Expected metadata %+v, got nil", tt.wantMetadata)
				} else if verInfo.Metadata.MinCliVersion != tt.wantMetadata.MinCliVersion {
					t.Errorf("MinCliVersion = %q, want %q", verInfo.Metadata.MinCliVersion, tt.wantMetadata.MinCliVersion)
				}
			}

			if len(verInfo.Platforms) != len(tt.wantPlatforms) {
				t.Errorf("Platform count = %d, want %d", len(verInfo.Platforms), len(tt.wantPlatforms))
			}

			for platform, wantInfo := range tt.wantPlatforms {
				gotInfo, ok := verInfo.Platforms[platform]
				if !ok {
					t.Errorf("Platform %q not found", platform)
					continue
				}

				if gotInfo.URL != wantInfo.URL {
					t.Errorf("Platform %q URL = %q, want %q", platform, gotInfo.URL, wantInfo.URL)
				}

				if gotInfo.Checksum != wantInfo.Checksum {
					t.Errorf("Platform %q Checksum = %q, want %q", platform, gotInfo.Checksum, wantInfo.Checksum)
				}
			}
		})
	}
}

func TestVersionInfo_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		verInfo  VersionInfo
		wantJSON string
	}{
		{
			name: "With metadata and platforms",
			verInfo: VersionInfo{
				Metadata: &VersionMetadata{
					MinCliVersion: "3.2.0",
				},
				Platforms: map[string]PlatformInfo{
					"darwin/amd64": {
						URL:      "https://example.com/plugin.tar.gz",
						Checksum: "abc123",
					},
				},
			},
			wantJSON: `{"darwin/amd64":{"url":"https://example.com/plugin.tar.gz","checksum":"abc123"},"metadata":{"minCliVersion":"3.2.0"}}`,
		},
		{
			name: "Without metadata",
			verInfo: VersionInfo{
				Platforms: map[string]PlatformInfo{
					"linux/amd64": {
						URL:      "https://example.com/plugin-linux.tar.gz",
						Checksum: "def456",
					},
				},
			},
			wantJSON: `{"linux/amd64":{"url":"https://example.com/plugin-linux.tar.gz","checksum":"def456"}}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotJSON, err := json.Marshal(tt.verInfo)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			var got, want map[string]interface{}
			if err := json.Unmarshal(gotJSON, &got); err != nil {
				t.Errorf("Failed to parse got JSON: %v", err)
				return
			}
			if err := json.Unmarshal([]byte(tt.wantJSON), &want); err != nil {
				t.Errorf("Failed to parse want JSON: %v", err)
				return
			}

			// Use reflect.DeepEqual instead of string comparison
			// because json.Marshal doesn't guarantee key order for maps
			if !reflect.DeepEqual(got, want) {
				gotStr, _ := json.Marshal(got)
				wantStr, _ := json.Marshal(want)
				t.Errorf("MarshalJSON() = %s, want %s", gotStr, wantStr)
			}
		})
	}
}

func TestPluginIndex_ParseRealWorld(t *testing.T) {
	jsonData := `{
		"plugins": [
			{
				"name": "aliyun-cli-fc",
				"latestVersion": "0.1.0",
				"description": "FC Plugin",
				"homepage": "https://github.com/aliyun/aliyun-cli",
				"versions": {
					"0.1.0": {
						"metadata": {
							"minCliVersion": "3.2.2"
						},
						"darwin/amd64": {
							"url": "https://example.com/fc-darwin-amd64.tar.gz",
							"checksum": "abc123"
						},
						"linux/amd64": {
							"url": "https://example.com/fc-linux-amd64.tar.gz",
							"checksum": "def456"
						}
					},
					"0.0.9": {
						"darwin/amd64": {
							"url": "https://example.com/fc-0.0.9-darwin-amd64.tar.gz",
							"checksum": "xyz789"
						}
					}
				}
			}
		]
	}`

	var index Index
	err := json.Unmarshal([]byte(jsonData), &index)
	if err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	if len(index.Plugins) != 1 {
		t.Fatalf("Expected 1 plugin, got %d", len(index.Plugins))
	}

	plugin := index.Plugins[0]
	if plugin.Name != "aliyun-cli-fc" {
		t.Errorf("Plugin name = %q, want %q", plugin.Name, "aliyun-cli-fc")
	}

	ver010, ok := plugin.Versions["0.1.0"]
	if !ok {
		t.Fatal("Version 0.1.0 not found")
	}

	if ver010.Metadata == nil {
		t.Error("Version 0.1.0 should have metadata")
	} else if ver010.Metadata.MinCliVersion != "3.2.2" {
		t.Errorf("MinCliVersion = %q, want %q", ver010.Metadata.MinCliVersion, "3.2.2")
	}

	if len(ver010.Platforms) != 2 {
		t.Errorf("Version 0.1.0 should have 2 platforms, got %d", len(ver010.Platforms))
	}

	ver009, ok := plugin.Versions["0.0.9"]
	if !ok {
		t.Fatal("Version 0.0.9 not found")
	}

	if ver009.Metadata != nil {
		t.Errorf("Version 0.0.9 should not have metadata, got %+v", ver009.Metadata)
	}

	if len(ver009.Platforms) != 1 {
		t.Errorf("Version 0.0.9 should have 1 platform, got %d", len(ver009.Platforms))
	}
}
