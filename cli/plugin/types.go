package plugin

import (
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
	Versions      map[string]VersionInfo `json:"versions"` // version -> platform -> info
}

type VersionInfo map[string]PlatformInfo

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
	Command     string `json:"command"` // 触发命令，如 "fc"
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

func GetCurrentPlatform() string {
	return runtime.GOOS + "/" + runtime.GOARCH
}
