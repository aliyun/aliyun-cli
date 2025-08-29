package go_migrate

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewGoMigrateCommand() *cli.Command {
	return &cli.Command{
		Name:   "go-migrate",
		Short:  i18n.T("Alibaba Cloud Golang SDK V1 to V2 Migration Tool", "阿里云 Golang SDK V1 到 V2 迁移工具"),
		Usage:  "aliyun go-migrate <codePath> --yes",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewGoMigrateOptionsContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true, // DO NOT use default help and version
	}
}
