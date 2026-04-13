// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package pluginsettings holds global (non-profile) plugin system settings,
// stored next to config.json (same directory as ai-mode.json).
package pluginsettings

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/util"
)

const ConfigFileName = "plugin-settings.json"

const EnvSourceBase = "ALIBABA_CLOUD_CLI_PLUGIN_SOURCE_BASE"

type PluginSettings struct {
	// SourceBase is the URL prefix for the plugins tree, e.g. https://example.com/plugins
	// Index: {SourceBase}/plugin_pkg_index.json, {SourceBase}/plugin_search_index.json
	// Packages: {SourceBase}/pkgs/{name}/{version}/{filename}
	SourceBase string `json:"source_base,omitempty"`
}

func Default() *PluginSettings {
	return &PluginSettings{}
}

func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

func Load(configDir string) (*PluginSettings, error) {
	path := GetConfigFilePath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	var c PluginSettings
	if err := json.Unmarshal(data, &c); err != nil {
		return Default(), nil
	}
	c.SourceBase = strings.TrimSpace(c.SourceBase)
	return &c, nil
}

func Save(configDir string, c *PluginSettings) error {
	if c == nil {
		c = Default()
	}
	path := GetConfigFilePath(configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}

func EffectiveSourceBase(c *PluginSettings) string {
	if v := strings.TrimSpace(util.GetFromEnv(EnvSourceBase)); v != "" {
		return strings.TrimRight(v, "/")
	}
	if c == nil {
		return ""
	}
	return strings.TrimRight(strings.TrimSpace(c.SourceBase), "/")
}
