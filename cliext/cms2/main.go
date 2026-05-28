package cms2

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewCms2Command() *cli.Command {
	return &cli.Command{
		Name: "cms2",
		Short: i18n.T(
			"Alibaba Cloud CloudMonitor (CMS) CLI — manage the full lifecycle of monitoring integrations, including APM, RUM, Prometheus Service, Synthetics, Alert Center, Event Center, and more.",
			"阿里云云监控 CLI — 管理云监控的接入/集成的全生命周期，包括应用监控（APM）、前端监控（RUM）、Prometheus 服务、云拔测（Synthetics）、告警中心、事件中心等。"),
		Usage:             "aliyun cms2 <command> [args...] [options...]",
		Hidden:            false,
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
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
					// The subprocess already wrote its own output (JSON
					// success/error envelope) to the connected stdout/stderr.
					// Propagate the exit code directly instead of returning
					// an error — returning an error causes the CLI framework
					// to print an ANSI-colored "ERROR: ..." line on stdout,
					// which corrupts the subprocess's JSON stream.
					cli.Exit(exitErr.Code)
				}
				return err
			}
			return nil
		},
	}
}
