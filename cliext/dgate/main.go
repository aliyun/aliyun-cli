package dgate

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

// NewDgateCommand 注册 `aliyun dgate`。
// dgate 使用自己的 OAuth / access token（类型 C），这里只负责安装、透传参数和注入 COMPAT_MODE。
func NewDgateCommand() *cli.Command {
	return &cli.Command{
		Name: "dgate",
		Short: i18n.T(
			"Alibaba Cloud DMS data gateway CLI",
			"阿里云 DMS 数据网关命令行",
		),
		Usage:             "aliyun dgate [command] [args...]",
		EnableUnknownFlag: true,
		KeepArgs:          true,
		SkipDefaultHelp:   true,
		Run: func(ctx *cli.Context, args []string) error {
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
					cli.Exit(exitErr.Code)
					return nil
				}
				return err
			}
			return nil
		},
	}
}
