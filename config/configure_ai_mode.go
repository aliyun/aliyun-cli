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

package config

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

func NewConfigureAiModeCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "ai-mode",
		Short: i18n.T(
			"manage global AI mode and User-Agent for API calls (not profile-scoped)",
			"管理全局 AI 模式及 API 调用的 User-Agent（不按 profile 区分）"),
		Usage: "ai-mode [command] [--config-path <configPath>]",
		Long: i18n.T(
			`Configure global AI mode. When enabled, all CLI API requests append the configured User-Agent segment (default: AlibabaCloud-Agent-Skills), in addition to the normal Aliyun CLI UA.`,
			`配置全局 AI 模式。启用后，所有 CLI API 请求会在常规 Aliyun CLI UA 之外追加配置的 User-Agent 段（默认：AlibabaCloud-Agent-Skills）。`),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeShow(ctx, configDir, cfg)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Category:     "ai-mode",
		Name:         "user-agent",
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"User-Agent segment for set-user-agent (not the product-call flag under aliyun <product>)",
			"用于 set-user-agent 的 UA 段（与 aliyun <product> 下的 --user-agent 无关）"),
	})
	cmd.Flags().Add(&cli.Flag{
		Category:     "ai-mode",
		Name:         "ossutil",
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"JSON text for set-ossutil (decoded to object/array/value; stored as ossutil in ai-mode.json)",
			"用于 set-ossutil 的 JSON 文本（解析为对象/数组/值；写入 ai-mode.json 的 ossutil）"),
	})
	AddFlags(cmd.Flags())

	cmd.AddSubCommand(newConfigureAiModeShowCommand())
	cmd.AddSubCommand(newConfigureAiModeEnableCommand())
	cmd.AddSubCommand(newConfigureAiModeDisableCommand())
	cmd.AddSubCommand(newConfigureAiModeSetUserAgentCommand())
	cmd.AddSubCommand(newConfigureAiModeResetUserAgentCommand())
	cmd.AddSubCommand(newConfigureAiModeSetOssutilCommand())
	cmd.AddSubCommand(newConfigureAiModeResetOssutilCommand())
	return cmd
}

func loadAiModeConfig(ctx *cli.Context) (configDir string, cfg *aimode.AiConfig, err error) {
	configDir = GetConfigDir(ctx)
	cfg, err = aimode.Load(configDir)
	if err != nil {
		return "", nil, fmt.Errorf("load ai-mode config failed: %w", err)
	}
	return configDir, cfg, nil
}

func newConfigureAiModeShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show [--config-path <configPath>]",
		Short: i18n.T("display current AI mode config", "显示当前 AI 模式配置"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeShow(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeEnableCommand() *cli.Command {
	return &cli.Command{
		Name:  "enable",
		Usage: "enable [--config-path <configPath>]",
		Short: i18n.T("turn on AI mode", "开启 AI 模式"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeEnable(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeDisableCommand() *cli.Command {
	return &cli.Command{
		Name:  "disable",
		Usage: "disable [--config-path <configPath>]",
		Short: i18n.T("turn off AI mode", "关闭 AI 模式"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeDisable(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeSetUserAgentCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-user-agent",
		Usage: "set-user-agent --user-agent <value> [--config-path <configPath>]",
		Short: i18n.T("set custom User-Agent segment for AI mode", "设置 AI 模式的自定义 User-Agent 段"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeSetUserAgent(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeResetUserAgentCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset-user-agent",
		Usage: "reset-user-agent [--config-path <configPath>]",
		Short: i18n.T("clear custom User-Agent segment (use default when AI mode is on)", "清除自定义 UA 段（启用 AI 模式时使用默认值）"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeResetUserAgent(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeSetOssutilCommand() *cli.Command {
	return &cli.Command{
		Name:  "set-ossutil",
		Usage: "set-ossutil --ossutil '<json>' [--config-path <configPath>]",
		Short: i18n.T("set ossutil JSON for cli_ai_ossutil (OSSUTIL_CONFIG_VALUE)", "设置 cli_ai_ossutil 的 ossutil JSON（写入 OSSUTIL_CONFIG_VALUE）"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeSetOssutil(ctx, configDir, cfg)
		},
	}
}

func newConfigureAiModeResetOssutilCommand() *cli.Command {
	return &cli.Command{
		Name:  "reset-ossutil",
		Usage: "reset-ossutil [--config-path <configPath>]",
		Short: i18n.T("clear ossutil / cli_ai_ossutil blob", "清除 ossutil / cli_ai_ossutil 段"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadAiModeConfig(ctx)
			if err != nil {
				return err
			}
			return doAiModeResetOssutil(ctx, configDir, cfg)
		},
	}
}

func doAiModeShow(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	out := struct {
		Enabled                bool   `json:"enabled"`
		UserAgent              string `json:"user_agent,omitempty"`
		Ossutil                any    `json:"ossutil,omitempty"`
		EffectiveUserAgent     string `json:"effective_user_agent"`
		RequestUserAgentSuffix string `json:"request_user_agent_suffix,omitempty"`
		ConfigFile             string `json:"config_file"`
	}{
		Enabled:                cfg.Enabled,
		UserAgent:              cfg.UserAgent,
		Ossutil:                cfg.PluginSpecialOSSUTIL,
		EffectiveUserAgent:     aimode.EffectiveUserAgent(cfg),
		RequestUserAgentSuffix: aimode.RequestUserAgentSuffix(cfg),
		ConfigFile:             aimode.GetConfigFilePath(configDir),
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	cli.Println(ctx.Stdout(), string(data))
	return nil
}

func doAiModeEnable(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	cfg.Enabled = true
	return aimode.Save(configDir, cfg)
}

func doAiModeDisable(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	cfg.Enabled = false
	return aimode.Save(configDir, cfg)
}

func doAiModeSetUserAgent(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	v, ok := ctx.Flags().Get("user-agent").GetValue()
	if !ok || v == "" {
		return fmt.Errorf("--user-agent is required for set-user-agent")
	}
	cfg.UserAgent = v
	return aimode.Save(configDir, cfg)
}

func doAiModeResetUserAgent(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	cfg.UserAgent = ""
	return aimode.Save(configDir, cfg)
}

func doAiModeSetOssutil(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	v, ok := ctx.Flags().Get("ossutil").GetValue()
	if !ok || strings.TrimSpace(v) == "" {
		return fmt.Errorf("--ossutil is required for set-ossutil (JSON text, e.g. '{\"k\":1}')")
	}
	var parsed any
	if err := json.Unmarshal([]byte(strings.TrimSpace(v)), &parsed); err != nil {
		return fmt.Errorf("invalid JSON for --ossutil: %w", err)
	}
	cfg.PluginSpecialOSSUTIL = parsed
	return aimode.Save(configDir, cfg)
}

func doAiModeResetOssutil(ctx *cli.Context, configDir string, cfg *aimode.AiConfig) error {
	cfg.PluginSpecialOSSUTIL = nil
	return aimode.Save(configDir, cfg)
}
