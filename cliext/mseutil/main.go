package mseutil

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewMseutilCommand() *cli.Command {
	return &cli.Command{
		Name:   "mseutil",
		Short:  i18n.T("Alibaba Cloud MSE utility for diagnosing Nacos/ZooKeeper instances", "阿里云 MSE 诊断工具（Nacos/ZooKeeper）"),
		Usage:  "aliyun mseutil <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			// mseutil is cobra-based; help is --help / -h
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
			if err := NewContext(ctx).Run(args); err != nil {
				if exitErr, ok := err.(*ExitError); ok {
					// Subprocess already wrote its own output to stdout/stderr.
					// Propagate the exit code directly instead of returning an
					// error (which would print an ANSI "ERROR: ..." line).
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
}
