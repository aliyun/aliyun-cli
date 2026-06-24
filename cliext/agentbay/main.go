package agentbay

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewAgentBayCommand() *cli.Command {
	return &cli.Command{
		Name: "agentbay",
		Short: i18n.T(
			"The AgentBay command-line interface (CLI) is a tool for the AgentBay service. This topic describes how to use the CLI to create, build, activate, and manage AgentBay custom images.",
			"AgentBay CLI 是 AgentBay 服务的命令行工具，用于管理镜像、API Key、Skill 和认证。"),
		Usage:  "aliyun agentbay [command] [args...] [options...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			options := NewContext(ctx)
			return options.Run(args)
		},
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
	}
}
