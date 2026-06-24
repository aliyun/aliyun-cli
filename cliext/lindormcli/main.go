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
			if ctx.IsHelp() {
				hasHelp := false
				for i, arg := range args {
					if arg == "help" {
						hasHelp = true
						break
					} else if arg == "--help" {
						args[i] = "help"
						hasHelp = true
						break
					}
				}
				if !hasHelp {
					args = append(args, "help")
				}
			}
			c := NewContext(ctx)
			if err := c.Run(args); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					// The subprocess already wrote its own output to
					// the connected stdout/stderr. Propagate the exit
					// code directly instead of returning an error.
					cli.Exit(exitErr.Code)
				}
				return err
			}
			return nil
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}
