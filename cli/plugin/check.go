package plugin

import (
	"context"
	"fmt"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/trust"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
	"golang.org/x/mod/semver"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type UpdateScope struct {
	Root       string `json:"root"`
	SourceBase string `json:"source_base"`
	Site       string `json:"site"`
}

func ResolveUpdateScope(configDir string) (UpdateScope, error) {
	settings, err := pluginsettings.Load(configDir)
	if err != nil {
		return UpdateScope{}, err
	}
	root := os.Getenv(EnvPluginsDir)
	if root == "" {
		root = filepath.Join(getHomePath(), ".aliyun", "plugins")
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return UpdateScope{}, err
	}
	source := pluginsettings.EffectiveSourceBase(settings)
	if source == "" {
		source = "https://aliyuncli.alicdn.com/plugins-v2"
	}
	return UpdateScope{Root: root, SourceBase: source, Site: pluginsettings.EffectiveSite(settings)}, nil
}

func ReadInstalled(root string) (*LocalManifest, error) {
	return (&Manager{rootDir: root}).GetLocalManifest()
}

type UpdateResult struct {
	Name                   string `json:"name"`
	Type                   string `json:"type"`
	Current                string `json:"current"`
	Latest                 string `json:"latest,omitempty"`
	MinCliVersion          string `json:"min_cli_version,omitempty"`
	InstalledMinCliVersion string `json:"installed_min_cli_version,omitempty"`
	UpdateAvailable        bool   `json:"update_available"`
	RequiresCLIUpgrade     bool   `json:"requires_cli_upgrade,omitempty"`
	Status                 string `json:"status"`
	Command                string `json:"command,omitempty"`
}

type UpdateReport struct {
	Status    string           `json:"status"`
	Fresh     bool             `json:"fresh"`
	CheckedAt time.Time        `json:"checked_at"`
	Plugins   []UpdateResult   `json:"plugins"`
	Proof     trust.CacheProof `json:"-"`
}

// CheckUpdates reads one verified index for all installed Go and metadata plugins.
// No archive is downloaded and no unsigned fallback is enabled.
func CheckUpdates(ctx context.Context, configDir string, scope UpdateScope) (UpdateReport, error) {
	r := UpdateReport{Status: "failed", CheckedAt: time.Now().UTC(), Plugins: []UpdateResult{}}
	local, err := ReadInstalled(scope.Root)
	if err != nil {
		return r, fmt.Errorf("installed plugin manifest is unreadable")
	}
	if len(local.Plugins) == 0 {
		r.Status, r.Fresh = "complete", true
		return r, nil
	}
	c := trust.New(trust.Config{CacheDir: filepath.Join(configDir, "trust"), SourceBase: scope.SourceBase, DiscoveryURL: trust.DiscoveryURLForSite(scope.Site)})
	artifact, err := c.FetchArtifactForUpdateCheck(ctx, trust.PurposePluginIndex, "plugins")
	if err != nil {
		return r, fmt.Errorf("plugin index check failed")
	}
	var index Index
	if err := trust.DecodeStrictJSON(artifact.Bytes, &index); err != nil {
		return r, fmt.Errorf("verified plugin index is invalid")
	}
	r = CompareUpdates(local, &index, cli.GetVersion())
	r.CheckedAt, r.Fresh, r.Proof = time.Now().UTC(), artifact.Fresh, artifact.Proof
	if !artifact.Fresh {
		r.Status = "cached"
		return r, fmt.Errorf("plugin index refresh failed; using valid verified cache")
	}
	return r, nil
}

func validVersion(v string) bool { return semver.IsValid("v" + strings.TrimPrefix(v, "v")) }

func CompareUpdates(local *LocalManifest, index *Index, cliVersion string) UpdateReport {
	r := UpdateReport{Status: "complete", Plugins: []UpdateResult{}}
	remote := make(map[string]*PluginInfo)
	for i := range index.Plugins {
		remote[index.Plugins[i].Name] = &index.Plugins[i]
	}
	for name, installed := range local.Plugins {
		item := UpdateResult{Name: name, Type: NormalizePluginType(installed.Type), Current: installed.Version, InstalledMinCliVersion: strings.TrimSpace(installed.MinCliVersion), Status: "not_in_source"}
		p := remote[name]
		if p != nil {
			latest, err := getLatestVersion(p, false)
			if err != nil {
				item.Status = "no_stable_release"
			} else if !validVersion(latest) || !validVersion(installed.Version) {
				item.Status = "unknown_version"
			} else {
				item.Latest = latest
				v := p.Versions[latest]
				_, anyPlatform := v.Platforms[PluginPlatformAny]
				_, native := v.Platforms[GetCurrentPlatform()]
				item.Status = "up_to_date"
				if v.Metadata != nil {
					item.MinCliVersion = strings.TrimSpace(v.Metadata.MinCliVersion)
				}
				if !anyPlatform && !native {
					item.Status = "unsupported_platform"
				} else {
					// Use a shell-safe canonical name; never display an index-supplied command.
					item.Command = "aliyun plugin update --name " + shellName(name)
					if compareVersion(installed.Version, latest) < 0 {
						item.Status, item.UpdateAvailable = "update_available", true
					}
				}
			}
		}
		required := item.MinCliVersion
		if validVersion(item.InstalledMinCliVersion) && (!validVersion(required) || compareVersion(required, item.InstalledMinCliVersion) < 0) {
			required = item.InstalledMinCliVersion
		}
		if validVersion(required) && validVersion(cliVersion) && !isDevVersion(cliVersion) {
			item.RequiresCLIUpgrade = compareVersion(cliVersion, required) < 0
		}
		r.Plugins = append(r.Plugins, item)
	}
	sort.Slice(r.Plugins, func(i, j int) bool { return r.Plugins[i].Name < r.Plugins[j].Name })
	return r
}

func shellName(name string) string {
	for _, ch := range name {
		if !(ch >= 'a' && ch <= 'z' || ch >= 'A' && ch <= 'Z' || ch >= '0' && ch <= '9' || ch == '-' || ch == '_' || ch == '.') {
			return "'" + strings.ReplaceAll(name, "'", "'\"'\"'") + "'"
		}
	}
	return name
}
