package iact3

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewIact3Command() *cli.Command {
	return &cli.Command{
		Name:   "iact3",
		Short:  i18n.T("IaC Templates Validation Test Tool", "IaC 模板验证测试工具"),
		Usage:  "aliyun iact3 test --template /path/to/template\n  aliyun iact3 upgrade",
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
