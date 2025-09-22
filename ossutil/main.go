package ossutil

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewOssutilCommand() *cli.Command {
	return &cli.Command{
		Name:   "ossutil",
		Short:  i18n.T("Alibaba OSS Service CLI", "阿里云OSS服务CLI工具"),
		Usage:  "aliyun ossutil ls --region cn-hangzhou",
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
