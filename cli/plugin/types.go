package plugin

import (
	"encoding/json"
	"runtime"

	runtimeschema "github.com/aliyun/aliyun-openapi-runtime/schema"
)

// Plugin distribution kinds. A product ships as exactly one form:
// a compiled Go binary run as a separate process ("go"), or a bundle
// of JSON metadata interpreted in-process by aliyun-openapi-runtime ("meta").
// An empty value is treated as "go" for backward compatibility with
// manifests/indexes authored before meta plugins existed.
const (
	PluginTypeGo   = "go"
	PluginTypeMeta = "meta"

	// PluginPlatformAny identifies a platform-independent package artifact.
	// Installers prefer the exact os-arch entry and fall back to this key.
	PluginPlatformAny = "any"
)

// NormalizePluginType maps an empty/unknown type string to the legacy
// default ("go").
func NormalizePluginType(t string) string {
	if t == PluginTypeMeta {
		return PluginTypeMeta
	}
	return PluginTypeGo
}

type Index struct {
	Plugins []PluginInfo `json:"plugins"`
}

type PluginInfo struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	ProductName map[string]string `json:"productName"` // en, zh
	ProductCode string            `json:"productCode"`
	Homepage    string            `json:"homepage"`
	// Command 是插件在 host CLI 里的主二级命令名（例如 "hologram"）。
	// 由索引发布侧从 manifest.command 汇总，此处保留字段以便 client 展示 / 匹配。
	Command string `json:"command,omitempty"`
	// CommandAliases 是产品方声明的额外二级命令入口（例如 hologram 插件的 ["hologres"]）。
	// 与 LocalPlugin.CommandAliases 语义一致，索引发布侧从 manifest.commandAliases 汇总落盘；
	// findPluginInIndex 在 plugin name / short-name 匹配之外再叠加 alias 匹配，支持
	// `aliyun plugin install --name <alias>`。索引侧未落盘时该字段为 nil，行为退化为纯 name 匹配。
	CommandAliases []string               `json:"commandAliases,omitempty"`
	Versions       map[string]VersionInfo `json:"versions"` // version -> VersionInfo
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
	Path        string `json:"path"` // 插件目录路径
	ProductCode string `json:"productCode,omitempty"`
	// Type is the distribution kind: "go" (default) or "meta". Go
	// plugins are executed as a separate process; meta plugins are
	// interpreted in-process by aliyun-openapi-runtime.
	Type string `json:"type,omitempty"`
	// Metadata records the installed metadata encoding/layout contract. The
	// package manifest remains authoritative; this copy supports routing and
	// diagnostics without probing the package directory.
	Metadata *MetadataDescriptor `json:"metadata,omitempty"`
	// Command is the top-level CLI subcommand this plugin serves (e.g. "hologram" for `aliyun hologram ...`).
	// 本期 runtime 不直接把 Command 当作查找 key —— 因为 plugin name 与 Command 都派生自 productCode
	// (plugin name = "aliyun-cli-" + normalize(productCode), Command = normalize(productCode))，
	// matchPluginName 的 short-name 分支已经等价覆盖了主命令。Command 目前只用于安装期冲突校验和展示。
	// 未来若 plugin name 与 Command 解耦，需要同步把 Command 加进 FindInstalledPluginInManifest 的匹配。
	Command string `json:"command"`
	// CommandAliases 是产品方声明的额外二级命令入口（例如 hologram 插件的 ["hologres"]），
	// 是本期真正“新增”的查找 key，由 matchPluginAlias 在 FindInstalledPluginInManifest 里显式匹配。
	// 与主命令一样落到同一份查找路径，因此 execute / help / 管理命令（install/uninstall/update）
	// 都天然识别 alias；管理命令按 alias 定位到插件后仍以 plugin Name 为准做增删。
	CommandAliases   []string `json:"commandAliases,omitempty"`
	CmdNames         []string `json:"cmdNames"`
	ShortDescription string   `json:"shortDescription"`
	Description      string   `json:"description"`
	Inner            bool     `json:"inner,omitempty"` // 内置/产品侧插件（manifest.json inner）
	// ProfileRequired controls whether the host CLI requires a valid configured profile before spawning this plugin.
	// Defaults to true (legacy behavior) when unset.
	// Set to false for plugins whose APIs may not need a profile (e.g. token-backed gateways, hybrid auth).
	// The plugin remains responsible for per-API auth validation.
	// Use a pointer so an absent field in legacy manifests is reliably distinguishable from an explicit `false`.
	ProfileRequired *bool `json:"profileRequired,omitempty"`
}

func (lp *LocalPlugin) IsProfileRequired() bool {
	if lp == nil || lp.ProfileRequired == nil {
		return true
	}
	return *lp.ProfileRequired
}

// IsMeta reports whether this installed plugin is a JSON-metadata
// plugin (interpreted in-process) rather than a Go binary.
func (lp *LocalPlugin) IsMeta() bool {
	return lp != nil && NormalizePluginType(lp.Type) == PluginTypeMeta
}

type PluginAPIVersions struct {
	Default     string                          `json:"default,omitempty"`
	Supported   []string                        `json:"supported,omitempty"`
	VersionInfo map[string]PluginAPIVersionInfo `json:"versionInfo,omitempty"`
}

type PluginAPIVersionInfo struct {
	Deprecated  bool   `json:"deprecated"`
	Description string `json:"description,omitempty"`
	Recommended bool   `json:"recommended"`
}

type PluginManifest struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Type        string `json:"type,omitempty"`
	ProductCode string `json:"productCode,omitempty"`
	// Command is the top-level CLI subcommand this plugin exposes to end users.
	// See LocalPlugin.Command for the runtime semantics.
	Command string `json:"command"`
	// CommandAliases 声明该插件在 host CLI 上的额外二级命令入口，与 Command 一起在安装期做冲突校验并落盘。
	CommandAliases   []string           `json:"commandAliases,omitempty"`
	ShortDescription string             `json:"shortDescription"`
	Description      string             `json:"description"`
	Inner            bool               `json:"inner,omitempty"`
	APIVersions      *PluginAPIVersions `json:"apiVersions,omitempty"`
	Bin              struct {
		Path string `json:"path"` // 二进制文件相对路径
	} `json:"bin"`
	CmdNames        []string            `json:"cmdNames"`
	MinCliVersion   string              `json:"minCliVersion,omitempty"`
	Metadata        *MetadataDescriptor `json:"metadata,omitempty"`
	ProfileRequired *bool               `json:"profileRequired,omitempty"`
}

// MetadataDescriptor is shared with aliyun-openapi-runtime so the installer
// validates exactly the contract consumed by the runtime.
type MetadataDescriptor = runtimeschema.MetadataDescriptor

// Key: kebab-case command name (e.g., "fc create-alias")
// Value: plugin name (e.g., "aliyun-cli-fc")
type CommandIndex map[string]string

func GetCurrentPlatform() string {
	return runtime.GOOS + "-" + runtime.GOARCH
}
