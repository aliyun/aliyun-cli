package config

import (
	"encoding/json"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/updatenotifier"
	"path/filepath"
)

func NewConfigureUpdateNotifierCommand() *cli.Command {
	show := func(ctx *cli.Context, args []string) error {
		if len(args) > 0 {
			return cli.NewInvalidCommandError(args[0], ctx)
		}
		dir := GetConfigPath()
		cfg, err := updatenotifier.Load(dir)
		if err != nil {
			return err
		}
		return json.NewEncoder(ctx.Stdout()).Encode(map[string]any{
			"enabled": cfg.Enabled, "environment_disabled": updatenotifier.EnvironmentDisabled(),
			"config_file": filepath.Join(dir, updatenotifier.ConfigFileName),
		})
	}
	c := &cli.Command{Name: "update-notifier", DisablePersistentFlags: true, Usage: "update-notifier [show|enable|disable]", Short: i18n.T("manage automatic update checks", "管理自动更新检查"), Run: show}
	c.AddSubCommand(&cli.Command{Name: "show", Short: i18n.T("show update check settings", "显示更新检查设置"), Run: show})
	for _, enabled := range []bool{true, false} {
		name := "disable"
		if enabled {
			name = "enable"
		}
		c.AddSubCommand(&cli.Command{Name: name, Short: i18n.T(name+" automatic update checks", name+" 自动更新检查"), Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			return updatenotifier.Save(GetConfigPath(), enabled)
		}})
	}
	return c
}
