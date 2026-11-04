package bl

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewBlCommand() *cli.Command {
	cmd := &cli.Command{
		Name:              "bl",
		Short:             i18n.T("Alibaba Cloud Model Studio CLI", "阿里云百炼 CLI"),
		Usage:             "aliyun bl <command> [args...]",
		RawArgs:           true,
		SkipDefaultHelp:   true,
		EnableUnknownFlag: true,
		Run: func(ctx *cli.Context, args []string) error {
			if err := NewContext(ctx).Run(args); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					cli.Exit(exitErr.Code)
					return nil
				}
				return err
			}
			return nil
		},
	}
	config.AddFlags(cmd.Flags())
	return cmd
}
