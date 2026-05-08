package appmanagerutil

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewAppManagerCommand() *cli.Command {
	return &cli.Command{
		Name:   "appmanager",
		Short:  i18n.T("Alibaba Cloud AppManager CLI", "阿里云应用管理CLI工具"),
		Usage:  "aliyun appmanager <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			// appmanager-cli 使用 click 框架，help 通过 --help 触发
			if ctx.IsHelp() {
				hasHelp := false
				for _, arg := range args {
					if arg == "--help" || arg == "-h" {
						hasHelp = true
						break
					}
				}
				if !hasHelp {
					args = append(args, "--help")
				}
			}
			options := NewContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true, // DO NOT use default help and version
	}
}
