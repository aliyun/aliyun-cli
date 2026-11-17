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
	"net/url"
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
			`Configure plugin system settings. Site selects the official trust endpoint; source-base selects a custom plugin source.`,
			`配置插件系统设置。site 选择官方信任站点，source-base 选择自定义插件源。`),
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
		Usage: "set [--source-base <url>] [--site <cn|intl>]",
		Short: i18n.T("set plugin source or official trust site", "设置插件源或官方信任站点"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			sourceFlag := ctx.Flags().Get("source-base")
			siteFlag := ctx.Flags().Get("site")
			sourceAssigned := sourceFlag != nil && sourceFlag.IsAssigned()
			siteAssigned := siteFlag != nil && siteFlag.IsAssigned()
			if !sourceAssigned && !siteAssigned {
				return fmt.Errorf("at least one of --source-base or --site is required")
			}
			configDir, cfg, err := loadPluginSettings()
			if err != nil {
				return err
			}
			if sourceAssigned {
				v, _ := sourceFlag.GetValue()
				v = strings.TrimSpace(v)
				if v == "" {
					return fmt.Errorf("source-base must not be empty (use 'configure plugin-settings clear' to reset)")
				}
				u, parseErr := url.Parse(v)
				if parseErr != nil || u == nil || u.Host == "" || !strings.EqualFold(u.Scheme, "https") {
					return fmt.Errorf("source-base must use https://")
				}
				cfg.SourceBase = strings.TrimRight(v, "/")
			}
			if siteAssigned {
				v, _ := siteFlag.GetValue()
				if !pluginsettings.IsValidSite(v) {
					return fmt.Errorf("site must be cn or intl")
				}
				cfg.Site = pluginsettings.NormalizeSite(v)
			}
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
	cmd.Flags().Add(&cli.Flag{
		Category:     "plugin-settings",
		Name:         "site",
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"official trust site: cn or intl",
			"官方信任站点：cn 或 intl"),
	})
	return cmd
}

func newPluginSettingsClearCommand() *cli.Command {
	return &cli.Command{
		Name:  "clear",
		Usage: "clear",
		Short: i18n.T("reset plugin settings (official cn site)", "重置插件设置（恢复国内官方站点）"),
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
	effectiveSite := pluginsettings.EffectiveSite(cfg)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(map[string]any{
		"config_file":           pluginsettings.GetConfigFilePath(configDir),
		"source_base":           strings.TrimSpace(cfg.SourceBase),
		"source_base_effective": effective,
		"site":                  pluginsettings.NormalizeSite(cfg.Site),
		"site_effective":        effectiveSite,
		"env_override":          strings.TrimSpace(os.Getenv(pluginsettings.EnvSourceBase)),
		"source_base_env":       strings.TrimSpace(os.Getenv(pluginsettings.EnvSourceBase)),
		"site_env":              strings.TrimSpace(os.Getenv(pluginsettings.EnvSite)),
	})
	return nil
}
