package saectl

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewSaectlCommand() *cli.Command {
	return &cli.Command{
		Name:   "saectl",
		Short:  i18n.T("Alibaba Serverless App Engine CLI", "阿里云 Serverless 应用引擎 CLI工具"),
		Usage:  "aliyun saectl <command> [flags]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true, // DO NOT use default help and version
	}
}
