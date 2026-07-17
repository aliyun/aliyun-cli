package acrutil

import (
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil/diagnosis"
	"github.com/aliyun/aliyun-cli/v3/cliext/acrutil/skill"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewAcrutilCommand() *cli.Command {
	cmd := &cli.Command{
		Name:                   "acrutil",
		Short:                  i18n.T("Alibaba Cloud ACR Enterprise Edition Instance CLI Tool", "阿里云ACR企业版实例CLI工具"),
		Usage:                  "aliyun acrutil <command> [args...]",
		Hidden:                 false,
		DisablePersistentFlags: true,
		EnableUnknownFlag:      true,
		KeepArgs:               true,
		SkipDefaultHelp:        true,
	}

	skillCmd := skill.NewSkillCommand()
	diagCmd := diagnosis.NewDiagnosisCommand()

	cmd.Run = func(ctx *cli.Context, args []string) error {
		// Handle help flag: convert --help to help subcommand
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

		// If no arguments or just "help", show available commands
		if len(args) == 0 || (len(args) == 1 && args[0] == "help") {
			cli.Printf(ctx.Stdout(), "%s\n\n", cmd.Short.Text())
			cli.Printf(ctx.Stdout(), "Usage:\n")
			cli.Printf(ctx.Stdout(), "  %s\n\n", cmd.Usage)

			cli.Printf(ctx.Stdout(), "Available Commands:\n")
			cli.Printf(ctx.Stdout(), "  %-20s %s\n", skillCmd.Name, skillCmd.Short.Text())
			cli.Printf(ctx.Stdout(), "  %-20s %s\n", diagCmd.Name, diagCmd.Short.Text())
			cli.Printf(ctx.Stdout(), "\n")

			cli.Printf(ctx.Stdout(), "Use `aliyun acrutil <command> --help` for more information.\n")
			return nil
		}

		// For unknown commands, show error
		return cli.NewErrorWithTip(
			fmt.Errorf("unknown command: %v", args),
			"Use 'aliyun acrutil --help' for more information.",
		)
	}

	cmd.AddSubCommand(skillCmd)
	cmd.AddSubCommand(diagCmd)

	return cmd
}
