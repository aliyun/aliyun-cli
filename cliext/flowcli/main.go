package flowcli

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewFlowcliCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "flow-cli",
		Short:  i18n.T("Alibaba Cloud DevOps Flow CLI for custom pipeline steps", "云效流水线 Flow-CLI，用于自定义开发步骤"),
		Usage:  "aliyun flow-cli <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			// flow-cli uses commander; help is conventionally triggered by --help / -h
			if ctx.IsHelp() {
				hasHelp := false
				for _, a := range args {
					if a == "--help" || a == "-h" {
						hasHelp = true
						break
					}
				}
				if !hasHelp {
					args = append(args, "--help")
				}
			}
			return NewContext(ctx).Run(args)
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
	// register aliyun config flags (e.g. --access-key-id / --region) so
	// they survive into ctx.Flags() for LoadProfileWithContext and so
	// RemoveFlagsForMainCli can strip them from the args we forward.
	config.AddFlags(cmd.Flags())
	return cmd
}
