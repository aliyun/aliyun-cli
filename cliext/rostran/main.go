package rostran

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewRostranCommand() *cli.Command {
	return &cli.Command{
		Name:   "rostran",
		Short:  i18n.T("ROS Transform Tool", "ROS 模板转换工具"),
		Usage:  "aliyun rostran <command> [flags]\n  aliyun rostran upgrade",
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
