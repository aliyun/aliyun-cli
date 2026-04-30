package acrskill

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewAcrSkillCommand() *cli.Command {
	return &cli.Command{
		Name:   "acr-skill",
		Short:  i18n.T("Alibaba Cloud ACR Skill Management CLI", "阿里云ACR服务Skill管理CLI工具"),
		Usage:  "aliyun acr-skill validate -d ./my-skill",
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
