/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(NewSecureFlag())
	fs.Add(NewForceFlag())
	fs.Add(NewEndpointFlag())
	fs.Add(NewVersionFlag())
	fs.Add(NewHeaderFlag())
	fs.Add(NewBodyFlag())
	fs.Add(NewBodyFileFlag())
	fs.Add(PagerFlag)
	fs.Add(NewAcceptFlag())
	fs.Add(NewOutputFlag())
	fs.Add(WaiterFlag)
	fs.Add(NewDryRunFlag())
	fs.Add(NewQuietFlag())
	fs.Add(NewRoaFlag())
}

const (
	SecureFlagName   = "secure"
	ForceFlagName    = "force"
	EndpointFlagName = "endpoint"
	VersionFlagName  = "version"
	HeaderFlagName   = "header"
	BodyFlagName     = "body"
	BodyFileFlagName = "body-file"
	AcceptFlagName   = "accept"
	RoaFlagName      = "roa"
	DryRunFlagName   = "dryrun"
	QuietFlagName    = "quiet"
	OutputFlagName   = "output"
)

func OutputFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(OutputFlagName)
}

func SecureFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(SecureFlagName)
}

func ForceFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ForceFlagName)
}

func EndpointFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(EndpointFlagName)
}

func VersionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(VersionFlagName)
}

func HeaderFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(HeaderFlagName)
}

func BodyFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(BodyFlagName)
}

func BodyFileFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(BodyFileFlagName)
}

func AcceptFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(AcceptFlagName)
}

func RoaFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RoaFlagName)
}

func DryRunFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(DryRunFlagName)
}

func QuietFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(QuietFlagName)
}

// TODO next version
//VerboseFlag = &cli.Flag{Category: "caller",
//	Name: "verbose",
//	Shorthand: 'v',
//	AssignedMode: cli.AssignedNone,
//	Short: i18n.T(
//		"add `--verbose` to enable verbose mode",
//		"使用 `--verbose` 开启啰嗦模式",
//	),
//}

func NewSecureFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: SecureFlagName, AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--secure` to force https",
			"使用 `--secure` 开关强制使用https方式调用")}
}

func NewForceFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: ForceFlagName, AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--force` to skip api and parameters check",
			"添加 `--force` 开关可跳过API与参数的合法性检查")}
}

func NewEndpointFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: EndpointFlagName, AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--endpoint <endpoint>` to assign endpoint",
			"使用 `--endpoint <endpoint>` 来指定接入点地址")}
}

func NewVersionFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: VersionFlagName, AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--version <YYYY-MM-DD>` to assign product api version",
			"使用 `--version <YYYY-MM-DD>` 来指定访问的API版本")}
}

func NewHeaderFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: HeaderFlagName, AssignedMode: cli.AssignedRepeatable,
		Short: i18n.T(
			"use `--header X-foo=bar` to add custom HTTP header, repeatable",
			"使用 `--header X-foo=bar` 来添加特定的HTTP头, 可多次添加")}
}

func NewBodyFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: BodyFlagName, AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--body $(cat foo.json)` to assign http body in RESTful call",
			"使用 `--body $(cat foo.json)` 来指定在RESTful调用中的HTTP包体")}
}

func NewBodyFileFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: BodyFileFlagName, AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"assign http body in Restful call with local file",
			"使用 `--body-file foo.json` 来指定输入包体")}
}

func NewAcceptFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: AcceptFlagName, AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"add `--accept {json|xml}` to add Accept header",
			"使用 `--accept {json|xml}` 来指定Accept头")}
}

func NewRoaFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: RoaFlagName, AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"use `--roa {GET|PUT|POST|DELETE}` to assign restful call.[DEPRECATED]",
			"使用 `--roa {GET|PUT|POST|DELETE}` 使用restful方式调用[已过期]",
		),
	}
}

func NewDryRunFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name:         DryRunFlagName,
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"add `--dryrun` to validate and print request without running.",
			"使用 `--dryrun` 在执行校验后打印请求包体，跳过实际运行",
		),
		ExcludeWith: []string{PagerFlag.Name, WaiterFlag.Name},
	}
}

func NewQuietFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name:         QuietFlagName,
		Shorthand:    'q',
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"add `--quiet` to hide normal output",
			"使用 `--quiet` 关闭正常输出",
		),
		ExcludeWith: []string{DryRunFlagName},
	}
}
