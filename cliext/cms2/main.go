package cms2

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewCms2Command() *cli.Command {
	return &cli.Command{
		Name: "cms2",
		Short: i18n.T(
			"Alibaba Cloud CloudMonitor (CMS) CLI — manage monitoring integrations, Prometheus, alert rules, and PromQL.",
			"阿里云云监控 CLI — 管理监控集成、Prometheus 实例、告警规则和 PromQL 查询。"),
		Usage:             "aliyun cms2 <command> [args...] [options...]",
		Hidden:            false,
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
		Run: func(ctx *cli.Context, args []string) error {
			if ctx.IsHelp() {
				hasHelp := false
				for i, arg := range args {
					if arg == "help" {
						hasHelp = true
						break
					} else if arg == "--help" {
						args[i] = "help"
						hasHelp = true
						break
					}
				}
				if !hasHelp {
					args = append(args, "help")
				}
			}
			c := NewContext(ctx)
			return c.Run(args)
		},
	}
}
