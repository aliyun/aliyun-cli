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

	"github.com/aliyun/aliyun-cli/v3/aimode"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewConfigureAiModeCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "ai-mode",
		Short: i18n.T(
			"manage global AI mode and User-Agent for API calls (not profile-scoped)",
			"管理全局 AI 模式及 API 调用的 User-Agent（不按 profile 区分）"),
		Usage: "ai-mode [show|enable|disable|set-user-agent|reset-user-agent]",
		Long: i18n.T(
			`Configure global AI mode. When enabled, all CLI API requests append the configured User-Agent segment (default: AlibabaCloud-Agent-Skills), in addition to the normal Aliyun CLI UA.

Stored in a standalone file (e.g. ~/.aliyun/ai-mode.json), same pattern as safety-policy — not part of profile config.

Commands:
  show               - Display current AI mode config (default)
  enable             - Turn on AI mode (append UA on every API call)
  disable            - Turn off AI mode
  set-user-agent     - Set custom UA segment: --user-agent <value>
  reset-user-agent   - Clear custom UA; use default AlibabaCloud-Agent-Skills when AI mode is on`,
			`配置全局 AI 模式。启用后，所有 CLI API 请求会在常规 Aliyun CLI UA 之外追加配置的 User-Agent 段（默认：AlibabaCloud-Agent-Skills）。

配置保存在独立文件（如 ~/.aliyun/ai-mode.json），与 safety-policy 相同方式 — 不属于 profile。

命令:
  show               - 显示当前 AI 模式配置（默认）
  enable             - 开启 AI 模式（每次 API 调用追加 UA）
  disable            - 关闭 AI 模式
  set-user-agent     - 设置自定义 UA 段: --user-agent <值>
  reset-user-agent   - 清除自定义 UA；启用 AI 模式时使用默认 AlibabaCloud-Agent-Skills`),
		Run: func(ctx *cli.Context, args []string) error {
			return doConfigureAiMode(ctx, args)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Category:     "ai-mode",
		Name:         "user-agent",
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"User-Agent segment for set-user-agent (not the product-call flag under aliyun <product>)",
			"用于 set-user-agent 的 UA 段（与 aliyun <product> 下的 --user-agent 无关）"),
	})
	AddFlags(cmd.Flags())
	return cmd
}

func doConfigureAiMode(ctx *cli.Context, args []string) error {
	configDir := GetConfigDir(ctx)
	cfg, err := aimode.Load(configDir)
	if err != nil {
		return fmt.Errorf("load ai-mode config failed: %w", err)
	}

	subcmd := "show"
	if len(args) > 0 {
		subcmd = args[0]
	}

	switch subcmd {
	case "show":
		return doAiModeShow(ctx, configDir, cfg)
	case "enable":
		return doAiModeEnable(ctx, configDir, cfg)
	case "disable":
		return doAiModeDisable(ctx, configDir, cfg)
	case "set-user-agent":
		return doAiModeSetUserAgent(ctx, configDir, cfg)
	case "reset-user-agent":
		return doAiModeResetUserAgent(ctx, configDir, cfg)
	default:
		return fmt.Errorf("unknown subcommand: %s. Use show, enable, disable, set-user-agent, or reset-user-agent", subcmd)
	}
}

func doAiModeShow(ctx *cli.Context, configDir string, cfg *aimode.Config) error {
	out := struct {
		Enabled                bool   `json:"enabled"`
		UserAgent              string `json:"user_agent,omitempty"`
		EffectiveUserAgent     string `json:"effective_user_agent"`
		RequestUserAgentSuffix string `json:"request_user_agent_suffix,omitempty"`
		ConfigFile             string `json:"config_file"`
	}{
		Enabled:                cfg.Enabled,
		UserAgent:              cfg.UserAgent,
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

func doAiModeEnable(ctx *cli.Context, configDir string, cfg *aimode.Config) error {
	cfg.Enabled = true
	return aimode.Save(configDir, cfg)
}

func doAiModeDisable(ctx *cli.Context, configDir string, cfg *aimode.Config) error {
	cfg.Enabled = false
	return aimode.Save(configDir, cfg)
}

func doAiModeSetUserAgent(ctx *cli.Context, configDir string, cfg *aimode.Config) error {
	v, ok := ctx.Flags().Get("user-agent").GetValue()
	if !ok || v == "" {
		return fmt.Errorf("--user-agent is required for set-user-agent")
	}
	cfg.UserAgent = v
	return aimode.Save(configDir, cfg)
}

func doAiModeResetUserAgent(ctx *cli.Context, configDir string, cfg *aimode.Config) error {
	cfg.UserAgent = ""
	return aimode.Save(configDir, cfg)
}
