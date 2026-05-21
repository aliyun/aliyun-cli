package maxc

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// NewMaxcCommand wires the `aliyun maxc` cliext entrypoint. It mirrors
// cliext/cms2/main.go intentionally: aliyun root must NOT parse subcommand
// flags (EnableUnknownFlag) and must keep raw args (KeepArgs) so the maxc
// child process receives its own command line verbatim. SkipDefaultHelp
// hands `--help` to the child rather than emitting the parent's auto help.
func NewMaxcCommand() *cli.Command {
	return &cli.Command{
		Name: "maxc",
		Short: i18n.T(
			"MaxCompute CLI for AI agents — structured envelope output, query/job/meta/data tools.",
			"MaxCompute CLI 工具层（供 AI agent 调用） — 结构化输出，覆盖 SQL/作业/元数据/数据采样。"),
		Usage:             "aliyun maxc <command> [args...] [options...]",
		Hidden:            false,
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
		Run: func(ctx *cli.Context, args []string) error {
			// cms2 pattern: when parent --help is detected, rewrite the
			// last help token so the child sees `help` (its native form).
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
			return c.Run(args)
		},
	}
}
