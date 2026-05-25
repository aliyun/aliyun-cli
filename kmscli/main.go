package kmscli

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewKmscliCommand() *cli.Command {
	return &cli.Command{
		Name:   "kmscli",
		Short:  i18n.T("AlibabaCloud KMS CLI", "KMS CLI工具"),
		Usage:  "aliyun kmscli secret getsecret <secretName>\naliyun kmscli openclaw getsecret",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewContext(ctx)
			return options.Run(args)
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}
