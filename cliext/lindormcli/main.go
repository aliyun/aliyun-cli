package lindormcli

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewLindormCliCommand() *cli.Command {
	return &cli.Command{
		Name:   "lindorm",
		Short:  i18n.T("AlibabaCloud Lindorm Open API CLI", "Lindorm Open API CLI工具"),
		Usage:  "aliyun lindorm <command> [options]",
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
