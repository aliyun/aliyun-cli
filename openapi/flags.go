/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/cli"
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
}

var SecureFlag = cli.Flag{Category: "caller",
	Name: "secure", AssignedMode: cli.AssignedNone,
	Usage: i18n.T(
		"use `--secure` to force https",
		"使用 `--secure` 开关强制使用https方式调用")}

var ForceFlag = cli.Flag{Category: "caller",
	Name: "force", AssignedMode: cli.AssignedNone,
	Usage: i18n.T(
		"use `--force` to skip api and parameters check",
		"添加 `--force` 开关可跳过API与参数的合法性检查")}

var EndpointFlag = cli.Flag{Category: "caller",
	Name: "endpoint", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use `--endpoint <endpoint>` to assign endpoint",
		"使用 `--endpoint <endpoint>` 来指定接入点地址")}

var VersionFlag = cli.Flag{Category: "caller",
	Name: "version", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use `--version <YYYY-MM-DD>` to assign product api version",
		"使用 `--version <YYYY-MM-DD>` 来指定访问的API版本")}

var HeaderFlag = cli.Flag{Category: "caller",
	Name: "header", AssignedMode: cli.AssignedRepeatable,
	Usage: i18n.T(
		"use `--header X-foo=bar` to add custom HTTP header, repeatable",
		"使用 `--header X-foo=bar` 来添加特定的HTTP头, 可多次添加")}

var BodyFlag = cli.Flag{Category: "caller",
	Name: "body", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use `--body $(cat foo.json)` to assign http body in RESTful call",
		"使用 `--body $(cat foo.json)` 来指定在RESTful调用中的HTTP包体")}

var BodyFileFlag = cli.Flag{Category: "caller",
	Name: "body-file",AssignedMode: cli.AssignedOnce, Hidden: true,
	Usage: i18n.T(
		"assign http body in Restful call with local file",
		"使用 `--body-file foo.json` 来指定输入包体")}

var PagerFlag = cli.Flag{Category: "caller",
	Name: "all-pages", AssignedMode: cli.AssignedDefault, Hidden: true,
	Usage: i18n.T(
		"use `--all-pages` to merge pages for pageable APIs",
		"使用 `--all-pages` 在访问分页的API时合并分页")}

var AcceptFlag = cli.Flag{Category: "caller",
	Name: "accept", AssignedMode: cli.AssignedOnce, Hidden: true,
	Usage: i18n.T(
		"add `--accept {json|xml}` to add Accept header",
		"使用 `--accept {json|xml}` 来指定Accept头")}