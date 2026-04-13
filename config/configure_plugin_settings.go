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
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/pluginsettings"
)

func NewConfigurePluginSettingsCommand() *cli.Command {
	cmd := &cli.Command{
		Name:                   "plugin-settings",
		DisablePersistentFlags: true,
		Short: i18n.T(
			"manage global plugin system settings",
			"管理插件系统全局设置"),
		Usage: "plugin-settings [command]",
		Long: i18n.T(
			`Configure plugin system settings. When source-base is set, the CLI loads plugins from that base.`,
			`配置插件系统设置。设置 source-base 后，CLI 从该地址获取插件。`),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadPluginSettings()
			if err != nil {
				return err
			}
			return doPluginSettingsShow(ctx, configDir, cfg)
		},
	}

	cmd.AddSubCommand(newPluginSettingsShowCommand())
	cmd.AddSubCommand(newPluginSettingsSetCommand())
	cmd.AddSubCommand(newPluginSettingsClearCommand())
	return cmd
}

func loadPluginSettings() (configDir string, cfg *pluginsettings.PluginSettings, err error) {
	configDir = GetConfigPath()
	cfg, err = pluginsettings.Load(configDir)
	if err != nil {
		return "", nil, fmt.Errorf("load plugin-settings failed: %w", err)
	}
	return configDir, cfg, nil
}

func newPluginSettingsShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show",
		Short: i18n.T("display plugin system settings", "显示插件系统设置"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadPluginSettings()
			if err != nil {
				return err
			}
			return doPluginSettingsShow(ctx, configDir, cfg)
		},
	}
}

func newPluginSettingsSetCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "set",
		Usage: "set --source-base <url>",
		Short: i18n.T("set plugins tree source base URL", "设置插件源根地址"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			flag := ctx.Flags().Get("source-base")
			if flag == nil || !flag.IsAssigned() {
				return fmt.Errorf("missing --source-base <url>")
			}
			v, _ := flag.GetValue()
			v = strings.TrimSpace(v)
			if v == "" {
				return fmt.Errorf("source-base must not be empty (use 'configure plugin-settings clear' to reset)")
			}
			if !strings.HasPrefix(strings.ToLower(v), "http://") && !strings.HasPrefix(strings.ToLower(v), "https://") {
				return fmt.Errorf("source-base must start with http:// or https://")
			}
			configDir, cfg, err := loadPluginSettings()
			if err != nil {
				return err
			}
			cfg.SourceBase = strings.TrimRight(v, "/")
			if err := pluginsettings.Save(configDir, cfg); err != nil {
				return err
			}
			return doPluginSettingsShow(ctx, configDir, cfg)
		},
	}
	cmd.Flags().Add(&cli.Flag{
		Category:     "plugin-settings",
		Name:         "source-base",
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"plugins tree base URL for set (e.g. https://example.com/plugins)",
			"set 命令使用的插件源 URL（例如 https://example.com/plugins）"),
	})
	return cmd
}

func newPluginSettingsClearCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "clear",
		Short: i18n.T("remove custom source base (use built-in default)", "清除自定义插件源根地址（恢复内置默认）"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, _, err := loadPluginSettings()
			if err != nil {
				return err
			}
			cfg := pluginsettings.Default()
			if err := pluginsettings.Save(configDir, cfg); err != nil {
				return err
			}
			return doPluginSettingsShow(ctx, configDir, cfg)
		},
	}
}

func doPluginSettingsShow(ctx *cli.Context, configDir string, cfg *pluginsettings.PluginSettings) error {
	w := ctx.Stdout()
	effective := pluginsettings.EffectiveSourceBase(cfg)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"config_file":           pluginsettings.GetConfigFilePath(configDir),
		"source_base":           strings.TrimSpace(cfg.SourceBase),
		"source_base_effective": effective,
		"env_override":          strings.TrimSpace(os.Getenv(pluginsettings.EnvSourceBase)),
	})
	return nil
}
