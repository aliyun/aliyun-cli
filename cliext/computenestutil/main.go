package computenestutil

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewComputenestCommand() *cli.Command {
	cmd := &cli.Command{
		Name:   "computenest-cli",
		Short:  i18n.T("Alibaba Cloud ComputeNest CLI", "阿里云计算巢CLI工具"),
		Usage:  "aliyun computenest-cli <command> [args...]",
		Hidden: false,
		Run: func(ctx *cli.Context, args []string) error {
			// computenest-cli 使用 click 框架，help 通过 --help 触发
			if ctx.IsHelp() {
				hasHelp := false
				for _, arg := range args {
					if arg == "--help" || arg == "-h" {
						hasHelp = true
						break
					}
				}
				if !hasHelp {
					args = append(args, "--help")
				}
			}
			options := NewContext(ctx)
			return options.Run(args)
		},
		// allow unknown args
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true, // DO NOT use default help and version
	}
	// 注册 aliyun 主程序 config 类 flag（如 --access-key-id / --region 等），
	// 使 ctx.Flags() 在子命令上下文中可见这些 flag，
	// 从而 LoadProfileWithContext 能正确合并命令行覆盖值，
	// 并且 RemoveFlagsForMainCli 能将它们从透传给 computenest-cli 的 args 中剔除。
	config.AddFlags(cmd.Flags())
	return cmd
}
