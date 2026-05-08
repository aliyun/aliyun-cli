package acrutil

import (
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/acrutil/skill"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewAcrutilCommand() *cli.Command {
	cmd := &cli.Command{
		Name:                   "acrutil",
		Short:                  i18n.T("Alibaba Cloud ACR Enterprise Edition Instance CLI Tool", "阿里云ACR企业版实例CLI工具"),
		Usage:                  "acrutil <command> [args...]",
		Hidden:                 false,
		DisablePersistentFlags: true,
		Run: func(ctx *cli.Context, args []string) error {
			return cli.NewErrorWithTip(
				fmt.Errorf("command missing"),
				"Available commands: skill. Use 'aliyun acrutil --help' for more information.",
			)
		},
	}

	cmd.AddSubCommand(skill.NewSkillCommand())

	return cmd
}
