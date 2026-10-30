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

package telemetry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	ConfigFileName = "telemetry.json"
	CacheDirName   = "telemetry-cache"
)

type Config struct {
	Enabled bool `json:"enabled"`
}

func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
	}
}

func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

func GetInnerCacheDir(configDir string) string {
	return filepath.Join(configDir, CacheDirName, "inner")
}

func GetPublicCacheDir(configDir string) string {
	return filepath.Join(configDir, CacheDirName, "public")
}

func GetCacheDir(configDir, pipeline string) string {
	if pipeline == PipelineInner {
		return GetInnerCacheDir(configDir)
	}
	return GetPublicCacheDir(configDir)
}

func Load(configDir string) (*Config, error) {
	path := GetConfigFilePath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("invalid telemetry config: %w", err)
	}
	return &c, nil
}

func Save(configDir string, c *Config) error {
	if c == nil {
		c = DefaultConfig()
	}
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "\t")
	if err != nil {
		return err
	}
	// Only explicit preference changes write this single-field file. Atomic
	// replacement keeps concurrent readers safe; the last completed save wins.
	return atomicWriteFile(GetConfigFilePath(configDir), data, 0600)
}
