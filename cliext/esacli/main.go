package esacli

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewEsacliCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "esa-cli",
		Short:  i18n.T("Alibaba Cloud ESA CLI for Edge Routine development", "阿里云 ESA 边缘 Routine 开发工具"),
		Usage:  "aliyun esa-cli <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			// esa-cli uses yargs; help is conventionally triggered by --help
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
