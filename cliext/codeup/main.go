package codeup

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewCodeupCliCommand() *cli.Command {
	return &cli.Command{
		Name:                   "codeup-cli",
		Short:                  i18n.T("Migrate third-party code repositories to Alibaba Cloud Codeup", "三方代码平台迁移到阿里云Codeup平台的命令行工具"),
		Usage:                  "aliyun codeup-cli <command> [args...]",
		Hidden:                 false,
		DisablePersistentFlags: true,
		EnableUnknownFlag:      true,
		KeepArgs:               true,
		SkipDefaultHelp:        true,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewContext(ctx)
			return options.Run(args)
		},
	}
}
