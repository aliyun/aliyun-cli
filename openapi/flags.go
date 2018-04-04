/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(SecureFlag)
	fs.Add(ForceFlag)
	fs.Add(EndpointFlag)
	fs.Add(VersionFlag)
	fs.Add(HeaderFlag)
	fs.Add(BodyFlag)
	fs.Add(BodyFileFlag)
	fs.Add(PagerFlag)
	fs.Add(AcceptFlag)
	fs.Add(OutputFlag)
	fs.Add(WaiterFlag)
	fs.Add(DryRunFlag)
	fs.Add(QuietFlag)
}

var (
	SecureFlag = &cli.Flag{Category: "caller",
		Name: "secure", AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--secure` to force https",
			"使用 `--secure` 开关强制使用https方式调用")}

	ForceFlag = &cli.Flag{Category: "caller",
		Name: "force", AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--force` to skip api and parameters check",
			"添加 `--force` 开关可跳过API与参数的合法性检查")}

	EndpointFlag = &cli.Flag{Category: "caller",
		Name: "endpoint", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--endpoint <endpoint>` to assign endpoint",
			"使用 `--endpoint <endpoint>` 来指定接入点地址")}

	VersionFlag = &cli.Flag{Category: "caller",
		Name: "version", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--version <YYYY-MM-DD>` to assign product api version",
			"使用 `--version <YYYY-MM-DD>` 来指定访问的API版本")}

	HeaderFlag = &cli.Flag{Category: "caller",
		Name: "header", AssignedMode: cli.AssignedRepeatable,
		Short: i18n.T(
			"use `--header X-foo=bar` to add custom HTTP header, repeatable",
			"使用 `--header X-foo=bar` 来添加特定的HTTP头, 可多次添加")}

	BodyFlag = &cli.Flag{Category: "caller",
		Name: "body", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--body $(cat foo.json)` to assign http body in RESTful call",
			"使用 `--body $(cat foo.json)` 来指定在RESTful调用中的HTTP包体")}

	BodyFileFlag = &cli.Flag{Category: "caller",
		Name: "body-file", AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"assign http body in Restful call with local file",
			"使用 `--body-file foo.json` 来指定输入包体")}

	AcceptFlag = &cli.Flag{Category: "caller",
		Name: "accept", AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"add `--accept {json|xml}` to add Accept header",
			"使用 `--accept {json|xml}` 来指定Accept头")}

	RoaFlag = &cli.Flag{Category: "caller",
		Name: "roa", AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"use `--roa {GET|PUT|POST|DELETE}` to assign restful call.[DEPRECATED]",
			"使用 `--roa {GET|PUT|POST|DELETE}` 使用restful方式调用[已过期]",
		),
	}

	DryRunFlag = &cli.Flag{Category: "caller",
		Name: "dry-run",
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"add `--dry-run` to validate and print request without running.",
			"使用 `--dry-run` 在执行校验后打印请求包体，跳过实际运行",
		),
		ExcludeWith: []string{PagerFlag.Name, WaiterFlag.Name},
	}

	QuietFlag = &cli.Flag{Category: "caller",
		Name: "quiet",
		Shorthand: 'q',
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"add `--quiet` to hide normal output",
			"使用 `--quiet` 关闭正常输出",
		),
		ExcludeWith: []string{DryRunFlag.Name},
	}

	// TODO
	//VerboseFlag = &cli.Flag{Category: "caller",
	//	Name: "verbose",
	//	Shorthand: 'v',
	//	AssignedMode: cli.AssignedNone,
	//	Short: i18n.T(
	//		"add `--verbose` to enable verbose mode",
	//		"使用 `--verbose` 开启啰嗦模式",
	//	),
	//}
)
