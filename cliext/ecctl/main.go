package ecctl

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewEcctlCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "ecctl",
		Short:  i18n.T("Alibaba Cloud Elastic Compute Control CLI", "弹性计算控制 CLI (ecctl)"),
		Usage:  "aliyun ecctl <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			args = AppendHelpIfNeeded(ctx.IsHelp(), args)
			if err := NewContext(ctx).Run(args); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					cli.Exit(exitErr.Code)
					return nil
				}
				return err
			}
			return nil
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
	config.AddFlags(cmd.Flags())
	return cmd
}
