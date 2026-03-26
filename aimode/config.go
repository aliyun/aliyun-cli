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

package aimode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/util"
)

// DefaultUserAgent is the protocol default when user_agent is unset in config.
const DefaultUserAgent = "AlibabaCloud-Agent-Skills"

// UserAgentEnabledMarker is always included in User-Agent when AI mode is enabled (before the skills segment).
const UserAgentEnabledMarker = "AlibabaCloud-AIMode/enabled"

// ConfigFileName is the standalone global AI mode file under the config directory (not per-profile).
const ConfigFileName = "ai-mode.json"

// Config holds global AI mode and User-Agent settings (shared file, like safety-policy.json).
type Config struct {
	Enabled   bool   `json:"enabled"`
	UserAgent string `json:"user_agent,omitempty"`
}

// DefaultConfig returns AI mode off and empty user_agent (effective UA = DefaultUserAgent when enabled).
func DefaultConfig() *Config {
	return &Config{Enabled: false, UserAgent: ""}
}

// EffectiveUserAgent returns the User-Agent segment to append when AI mode is on.
func EffectiveUserAgent(c *Config) string {
	if c == nil {
		return util.SanitizeUserAgent(DefaultUserAgent)
	}
	s := strings.TrimSpace(c.UserAgent)
	if s == "" {
		return util.SanitizeUserAgent(DefaultUserAgent)
	}
	return util.SanitizeUserAgent(s)
}

// RequestUserAgentSuffix is the full string appended to HTTP User-Agent when AI mode is enabled:
// UserAgentEnabledMarker + configured or default skills segment. Empty when disabled or nil config.
func RequestUserAgentSuffix(c *Config) string {
	if c == nil || !c.Enabled {
		return ""
	}
	seg := EffectiveUserAgent(c)
	m := util.SanitizeUserAgent(UserAgentEnabledMarker)
	if seg == "" {
		return m
	}
	return strings.TrimSpace(m + " " + seg)
}

// GetConfigFilePath returns ~/.aliyun/ai-mode.json (or under --config-path directory).
func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, ConfigFileName)
}

// Load reads ai-mode.json. On missing file or invalid JSON, returns DefaultConfig() and nil error.
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
	return &c, nil
}

// Save writes ai-mode.json.
func Save(configDir string, c *Config) error {
	if c == nil {
		c = DefaultConfig()
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

// EnvAIMode is set to "1" for plugin subprocesses when global AI mode is enabled (plugins read this instead of ai-mode.json).
const EnvAIMode = "ALIBABA_CLOUD_CLI_AI_MODE"

// EnvAIUserAgent carries the sanitized AI User-Agent segment for plugins (default from config / AlibabaCloud-Agent-Skills).
const EnvAIUserAgent = "ALIBABA_CLOUD_CLI_AI_USER_AGENT"

// Keys inside the JSON payload encoded as OSSUTIL_CONFIG_VALUE (same channel as region_id).
const (
	OssutilConfigKeyAIMode      = "cli_ai_mode"
	OssutilConfigKeyAIUserAgent = "cli_ai_user_agent"
)

// RequestUserAgentSuffixForCommand applies per-invocation overrides from the root CLI:
// forceOff disables the suffix even when ai-mode.json has enabled: true;
// forceOn enables the suffix for this command only, still using user_agent / skills from ai-mode.json.
// If both forceOn and forceOff are true, forceOff wins.
func RequestUserAgentSuffixForCommand(cfg *Config, forceOn, forceOff bool) string {
	if forceOff {
		return ""
	}
	if forceOn {
		effective := DefaultConfig()
		if cfg != nil {
			effective.UserAgent = cfg.UserAgent
		}
		effective.Enabled = true
		return RequestUserAgentSuffix(effective)
	}
	if cfg == nil {
		return ""
	}
	return RequestUserAgentSuffix(cfg)
}

// MergeIntoOssutilConfigPayload adds AI mode fields to the map that becomes OSSUTIL_CONFIG_VALUE
// (alongside region_id, credentials, etc.). Respects forceOn/forceOff like RequestUserAgentSuffixForCommand.
func MergeIntoOssutilConfigPayload(configDir string, envMap map[string]any, forceOn, forceOff bool) {
	if envMap == nil || configDir == "" {
		return
	}
	cfg, err := Load(configDir)
	if err != nil {
		cfg = DefaultConfig()
	}
	suf := RequestUserAgentSuffixForCommand(cfg, forceOn, forceOff)
	if suf == "" {
		return
	}
	envMap[OssutilConfigKeyAIMode] = "1"
	envMap[OssutilConfigKeyAIUserAgent] = suf
}

// MergeUserAgentIntoPluginEnvs injects AI mode env vars for plugin subprocesses when AI mode is enabled
// (from ai-mode.json and/or forceOn). ALIBABA_CLOUD_USER_AGENT is left unchanged here.
// When forceOff is true, clears EnvAIMode and EnvAIUserAgent in envs so a shell-inherited ALIBABA_CLOUD_CLI_AI_* does not leak into the child.
func MergeUserAgentIntoPluginEnvs(configDir string, envs map[string]string, forceOn, forceOff bool) {
	if envs == nil {
		return
	}
	cfg, err := Load(configDir)
	if err != nil {
		cfg = DefaultConfig()
	}
	suf := RequestUserAgentSuffixForCommand(cfg, forceOn, forceOff)
	if suf == "" {
		if forceOff {
			envs[EnvAIMode] = ""
			envs[EnvAIUserAgent] = ""
		}
		return
	}
	envs[EnvAIMode] = "1"
	envs[EnvAIUserAgent] = suf
}
