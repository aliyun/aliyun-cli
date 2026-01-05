package plugin

import (
	"encoding/json"
	"runtime"
)

type Index struct {
	Plugins []PluginInfo `json:"plugins"`
}

type PluginInfo struct {
	Name          string                 `json:"name"`
	LatestVersion string                 `json:"latestVersion"`
	Description   string                 `json:"description"`
	Homepage      string                 `json:"homepage"`
	Versions      map[string]VersionInfo `json:"versions"` // version -> VersionInfo
}

type VersionInfo struct {
	Metadata  *VersionMetadata        `json:"metadata,omitempty"` // Version metadata (minCliVersion, etc.)
	Platforms map[string]PlatformInfo `json:"-"`                  // For internal use during unmarshaling
}

func (v *VersionInfo) UnmarshalJSON(data []byte) error {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	v.Platforms = make(map[string]PlatformInfo)

	for key, value := range raw {
		if key == "metadata" {
			var meta VersionMetadata
			if err := json.Unmarshal(value, &meta); err != nil {
				return err
			}
			v.Metadata = &meta
		} else {
			// It's a platform entry
			var platform PlatformInfo
			if err := json.Unmarshal(value, &platform); err != nil {
				return err
			}
			v.Platforms[key] = platform
		}
	}

	return nil
}

func (v VersionInfo) MarshalJSON() ([]byte, error) {
	result := make(map[string]interface{})

	if v.Metadata != nil {
		result["metadata"] = v.Metadata
	}

	for key, value := range v.Platforms {
		result[key] = value
	}

	return json.Marshal(result)
}

type VersionMetadata struct {
	MinCliVersion string `json:"minCliVersion,omitempty"`
}

type PlatformInfo struct {
	URL      string `json:"url"`
	Checksum string `json:"checksum"`
}

type LocalManifest struct {
	Plugins map[string]LocalPlugin `json:"plugins"`
}

type LocalPlugin struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Path        string `json:"path"`    // 插件目录路径
	Command     string `json:"command"` // 触发命令，not used for now
	Description string `json:"description"`
}

type PluginManifest struct {
	Name             string `json:"name"`
	Version          string `json:"version"`
	Command          string `json:"command"`
	ShortDescription string `json:"shortDescription"`
	Description      string `json:"description"`
	Bin              struct {
		Path string `json:"path"` // 二进制文件相对路径
	} `json:"bin"`
}

// CommandIndex is an inverted index mapping command names to plugin names
// Key: kebab-case command name (e.g., "fc create-alias")
// Value: plugin name (e.g., "aliyun-cli-fc")
type CommandIndex map[string]string

func GetCurrentPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
