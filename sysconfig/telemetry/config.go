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
	"os"
	"path/filepath"
	"time"
)

const ConfigFileName = "telemetry.json"

type Config struct {
	Enabled             bool   `json:"enabled,omitempty"`
	InnerAutoOn         *bool  `json:"inner_auto_on,omitempty"`
	InstanceID          string `json:"instance_id,omitempty"`
	LastUploadedAtInner string `json:"last_uploaded_at_inner,omitempty"`
	ConsentVersion      string `json:"consent_version,omitempty"`
	ConsentAt           string `json:"consent_at,omitempty"`
	LastUploadedAt      string `json:"last_uploaded_at,omitempty"`
}

func DefaultConfig() *Config {
	on := true
	return &Config{
		Enabled:     false,
		InnerAutoOn: &on,
	}
}

func InnerAutoOnEnabled(c *Config) bool {
	if c == nil || c.InnerAutoOn == nil {
		return true
	}
	return *c.InnerAutoOn
}

func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

func GetInnerCacheDir(configDir string) string {
	return filepath.Join(configDir, "telemetry", "inner")
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
		return DefaultConfig(), nil
	}
	if c.InnerAutoOn == nil {
		on := true
		c.InnerAutoOn = &on
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
	return os.WriteFile(GetConfigFilePath(configDir), data, 0600)
}

func EnsureInstanceID(configDir string, c *Config) (string, error) {
	if c == nil {
		c = DefaultConfig()
	}
	if c.InstanceID != "" {
		return c.InstanceID, nil
	}
	id, err := newUUID()
	if err != nil {
		return "", err
	}
	c.InstanceID = id
	if err := Save(configDir, c); err != nil {
		return "", err
	}
	return id, nil
}

func MarkInnerUploaded(configDir string, c *Config) error {
	if c == nil {
		var err error
		c, err = Load(configDir)
		if err != nil {
			return err
		}
	}
	c.LastUploadedAtInner = time.Now().Format(time.RFC3339)
	return Save(configDir, c)
}

func Purge(configDir string, c *Config) error {
	if c == nil {
		var err error
		c, err = Load(configDir)
		if err != nil {
			return err
		}
	}
	_ = os.RemoveAll(filepath.Join(configDir, "telemetry"))
	c.InstanceID = ""
	if err := Save(configDir, c); err != nil {
		return err
	}
	return nil
}
