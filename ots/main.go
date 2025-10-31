package ots

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewOtsCommand() *cli.Command {
	return &cli.Command{
		Name:   "ots",
		Short:  i18n.T("Alibaba Cloud Tablestore CLI", "阿里云表格存储CLI工具"),
		Usage:  "aliyun ots <command> --region cn-hangzhou",
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

