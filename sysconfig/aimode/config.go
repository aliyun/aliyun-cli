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

const DefaultUserAgent = "AlibabaCloud-Agent-Skills"

const UserAgentEnabledMarker = "AlibabaCloud-AIMode/enabled"

const AiConfigFileName = "ai-mode.json"

type AiConfig struct {
	Enabled              bool   `json:"enabled"`
	UserAgent            string `json:"user_agent,omitempty"`
	PluginSpecialOSSUTIL any    `json:"ossutil,omitempty"`
}

func DefaultAiConfig() *AiConfig {
	return &AiConfig{Enabled: false, UserAgent: ""}
}

func EffectiveUserAgent(c *AiConfig) string {
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
// UserAgentEnabledMarker + configured or default skills segment.
// Empty when disabled or nil config.
func RequestUserAgentSuffix(c *AiConfig) string {
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

func GetConfigFilePath(configDir string) string {
	return filepath.Join(configDir, AiConfigFileName)
}

func Load(configDir string) (*AiConfig, error) {
	path := GetConfigFilePath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultAiConfig(), nil
		}
		return nil, err
	}
	var c AiConfig
	if err := json.Unmarshal(data, &c); err != nil {
		return DefaultAiConfig(), nil
	}
	return &c, nil
}

func Save(configDir string, c *AiConfig) error {
	if c == nil {
		c = DefaultAiConfig()
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

const EnvAIMode = "ALIBABA_CLOUD_CLI_AI_MODE"
const EnvAIUserAgent = "ALIBABA_CLOUD_CLI_AI_USER_AGENT"

// Keys inside the JSON payload encoded as OSSUTIL_CONFIG_VALUE.
const (
	OssutilConfigKeyAIMode        = "cli_ai_mode"
	OssutilConfigKeyAIUserAgent   = "cli_ai_user_agent"
	OssutilConfigAIModeOssutilKey = "cli_ai_ossutil"
)

func MergeAiModeIntoOssutilPayload(configDir string, envMap map[string]any, forceOn, forceOff bool) {
	if envMap == nil || configDir == "" {
		return
	}
	cfg, err := Load(configDir)
	if err != nil {
		cfg = DefaultAiConfig()
	}
	if cfg != nil && cfg.PluginSpecialOSSUTIL != nil {
		envMap[OssutilConfigAIModeOssutilKey] = cfg.PluginSpecialOSSUTIL
	}
	suf := RequestUserAgentSuffixForCommand(cfg, forceOn, forceOff)
	if suf != "" {
		envMap[OssutilConfigKeyAIMode] = "1"
		envMap[OssutilConfigKeyAIUserAgent] = suf
	}
}

// RequestUserAgentSuffixForCommand applies per-invocation overrides from the root CLI
func RequestUserAgentSuffixForCommand(cfg *AiConfig, forceOn, forceOff bool) string {
	if forceOff {
		return ""
	}
	if forceOn {
		effective := DefaultAiConfig()
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

func MergeUserAgentIntoPluginEnvs(configDir string, envs map[string]string, forceOn, forceOff bool) {
	if envs == nil {
		return
	}
	cfg, err := Load(configDir)
	if err != nil {
		cfg = DefaultAiConfig()
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
