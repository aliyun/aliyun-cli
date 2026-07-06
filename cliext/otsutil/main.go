package otsutil

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewOtsutilCommand() *cli.Command {
	return &cli.Command{
		Name:   "otsutil",
		Short:  i18n.T("Alibaba Cloud Tablestore Utility", "阿里云表格存储工具"),
		Usage:  "aliyun otsutil <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			if ctx.IsHelp() {
				hasHelp := false
				for i, arg := range args {
					if arg == "help" {
						hasHelp = true
						break
					} else if arg == "--help" {
						// 将 --help 替换为 help
						args[i] = "help"
						hasHelp = true
						break
					}
				}
				// 如果没有找到 help 相关参数，说明是被过滤掉的 "help" 参数
				if !hasHelp {
					args = append(args, "help")
				}
			}
			// fmt.Println("otsutil args", args)
			options := NewContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true, // DO NOT use default help and version
	}
}
