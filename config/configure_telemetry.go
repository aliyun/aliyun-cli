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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/telemetry"
)

func NewConfigureTelemetryCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "telemetry",
		Short: i18n.T(
			"manage CLI telemetry",
			"管理 CLI 遥测"),
		Usage: "telemetry [command]",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			return doTelemetryShow(ctx, configDir, cfg)
		},
	}
	cmd.AddSubCommand(newConfigureTelemetryShowCommand())
	cmd.AddSubCommand(newConfigureTelemetryEnableCommand())
	cmd.AddSubCommand(newConfigureTelemetryDisableCommand())
	cmd.AddSubCommand(newConfigureTelemetryInnerAutoCommand())
	cmd.AddSubCommand(newConfigureTelemetryPurgeCommand())
	return cmd
}

func newConfigureTelemetryEnableCommand() *cli.Command {
	return &cli.Command{
		Name:  "enable",
		Usage: "enable",
		Short: i18n.T(
			"enable public CLI telemetry",
			"启用公网 CLI 遥测"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			cfg.Enabled = true
			return telemetry.Save(configDir, cfg)
		},
	}
}

func newConfigureTelemetryDisableCommand() *cli.Command {
	return &cli.Command{
		Name:  "disable",
		Usage: "disable",
		Short: i18n.T(
			"disable public CLI telemetry",
			"关闭公网 CLI 遥测"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			cfg.Enabled = false
			return telemetry.Save(configDir, cfg)
		},
	}
}

func loadTelemetryConfig() (string, *telemetry.Config, error) {
	configDir := GetConfigPath()
	cfg, err := telemetry.Load(configDir)
	if err != nil {
		return "", nil, fmt.Errorf("load telemetry config failed: %w", err)
	}
	return configDir, cfg, nil
}

func newConfigureTelemetryShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show",
		Short: i18n.T("display telemetry status", "显示遥测状态"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			return doTelemetryShow(ctx, configDir, cfg)
		},
	}
}

func newConfigureTelemetryInnerAutoCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "inner-auto",
		Short: i18n.T("enable or disable inner plugin auto telemetry", "开启或关闭 inner 插件自动遥测"),
		Usage: "inner-auto enable|disable",
	}
	cmd.AddSubCommand(&cli.Command{
		Name:  "enable",
		Usage: "enable",
		Short: i18n.T("turn on inner plugin auto telemetry", "开启 inner 插件自动遥测"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			on := true
			cfg.InnerAutoOn = &on
			return telemetry.Save(configDir, cfg)
		},
	})
	cmd.AddSubCommand(&cli.Command{
		Name:  "disable",
		Usage: "disable",
		Short: i18n.T("turn off inner plugin auto telemetry", "关闭 inner 插件自动遥测"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			off := false
			cfg.InnerAutoOn = &off
			return telemetry.Save(configDir, cfg)
		},
	})
	return cmd
}

func newConfigureTelemetryPurgeCommand() *cli.Command {
	return &cli.Command{
		Name:  "purge",
		Usage: "purge",
		Short: i18n.T("delete local telemetry cache and reset instance id", "删除本地遥测缓存并重置 instance_id"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, cfg, err := loadTelemetryConfig()
			if err != nil {
				return err
			}
			return telemetry.Purge(configDir, cfg)
		},
	}
}

func doTelemetryShow(ctx *cli.Context, configDir string, cfg *telemetry.Config) error {
	innerStats := telemetry.InnerCacheStats(telemetry.GetInnerCacheDir(configDir))
	publicStats := telemetry.InnerCacheStats(telemetry.GetPublicCacheDir(configDir))
	instancePrefix := cfg.InstanceID
	if len(instancePrefix) > 8 {
		instancePrefix = instancePrefix[:8] + "..."
	}
	out := struct {
		PublicEnabled     bool   `json:"public_enabled"`
		InnerAutoOn       bool   `json:"inner_auto_on"`
		GlobalOptOut      bool   `json:"global_opt_out"`
		InnerOptOut       bool   `json:"inner_opt_out"`
		InstanceIDPrefix  string `json:"instance_id_prefix,omitempty"`
		InnerCacheFiles   int    `json:"inner_cache_files"`
		InnerCacheBytes   int64  `json:"inner_cache_bytes"`
		PublicCacheFiles  int    `json:"public_cache_files"`
		PublicCacheBytes  int64  `json:"public_cache_bytes"`
		LastUploaded      string `json:"last_uploaded_at,omitempty"`
		LastUploadedInner string `json:"last_uploaded_at_inner,omitempty"`
		ConfigFile        string `json:"config_file"`
	}{
		PublicEnabled:     cfg.Enabled,
		InnerAutoOn:       telemetry.InnerAutoOnEnabled(cfg),
		GlobalOptOut:      telemetry.GlobalOptOut(),
		InnerOptOut:       telemetry.InnerOptOut(),
		InstanceIDPrefix:  instancePrefix,
		InnerCacheFiles:   innerStats.Files,
		InnerCacheBytes:   innerStats.TotalBytes,
		PublicCacheFiles:  publicStats.Files,
		PublicCacheBytes:  publicStats.TotalBytes,
		LastUploaded:      cfg.LastUploadedAt,
		LastUploadedInner: cfg.LastUploadedAtInner,
		ConfigFile:        telemetry.GetConfigFilePath(configDir),
	}
	data, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return err
	}
	cli.Println(ctx.Stdout(), string(data))
	return nil
}
