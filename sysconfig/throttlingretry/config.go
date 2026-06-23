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

// Package throttlingretry stores global retry settings that only apply to
// OpenAPI throttling errors.
package throttlingretry

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const ConfigFileName = "throttling-retry.json"

const (
	EnvEnabled     = "ALIBABA_CLOUD_THROTTLING_RETRY_ENABLED"
	EnvMaxAttempts = "ALIBABA_CLOUD_THROTTLING_RETRY_MAX_ATTEMPTS"
	EnvMaxDelayMS  = "ALIBABA_CLOUD_THROTTLING_RETRY_MAX_DELAY_MS"
)

type Config struct {
	Enabled     *bool `json:"enabled,omitempty"`
	MaxAttempts int   `json:"max_attempts,omitempty"`
	MaxDelayMS  int64 `json:"max_delay_ms,omitempty"`
}

func Default() *Config {
	return &Config{}
}

func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

func Load(configDir string) (*Config, error) {
	path := GetConfigFilePath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	var c Config
	if err := json.Unmarshal(data, &c); err != nil {
		return Default(), nil
	}
	if c.MaxAttempts < 0 {
		c.MaxAttempts = 0
	}
	if c.MaxDelayMS < 0 {
		c.MaxDelayMS = 0
	}
	return &c, nil
}

func Save(configDir string, c *Config) error {
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

func MergeFromEnv(base *Config) *Config {
	if base == nil {
		base = Default()
	}

	out := *base
	if raw, ok := os.LookupEnv(EnvEnabled); ok {
		if enabled, err := strconv.ParseBool(strings.TrimSpace(raw)); err == nil {
			out.Enabled = &enabled
		}
	}
	if raw, ok := os.LookupEnv(EnvMaxAttempts); ok {
		if maxAttempts, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && maxAttempts > 0 {
			out.MaxAttempts = maxAttempts
		}
	}
	if raw, ok := os.LookupEnv(EnvMaxDelayMS); ok {
		if maxDelayMS, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64); err == nil && maxDelayMS > 0 {
			out.MaxDelayMS = maxDelayMS
		}
	}
	return &out
}

func LoadEffective(configDir string) (*Config, error) {
	c, err := Load(configDir)
	if err != nil {
		return nil, err
	}
	return MergeFromEnv(c), nil
}

func MergeIntoPluginEnvs(configDir string, envs map[string]string) {
	if envs == nil || configDir == "" {
		return
	}
	c, err := LoadEffective(configDir)
	if err != nil {
		return
	}
	if c.Enabled != nil {
		envs[EnvEnabled] = strconv.FormatBool(*c.Enabled)
	}
	if c.MaxAttempts > 0 {
		envs[EnvMaxAttempts] = strconv.Itoa(c.MaxAttempts)
	}
	if c.MaxDelayMS > 0 {
		envs[EnvMaxDelayMS] = strconv.FormatInt(c.MaxDelayMS, 10)
	}
}
