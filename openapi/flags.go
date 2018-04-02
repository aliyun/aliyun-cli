/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

const (
	flagOutputTableRows = "output-table-rows"
	flagOutputTableCols = "output-table-cols"

	flagWaitForExpr   = "wait-for-expr"
	flagWaitForTarget = "wait-for-target"
	flagWaitTimeout   = "wait-timeout"
	flagWaitInterval  = "wait-interval"

	flagRetryTimeout = "retry-timeout"
	flagRetryCount   = "retry-count"
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

	//fs.Add(OutputTableRowFlag)
	//fs.Add(OutputTableColsFlag)
	//
	//fs.Add(WaitForExprFlag)
	//fs.Add(WaitForTargetFlag)
	//fs.Add(WaitTimeoutFlag)
	//fs.Add(WaitIntervalFlag)
	//
	//fs.Add(RetryTimeoutFlag)
	//fs.Add(RetryCountFlag)
}

var (
	SecureFlag = &cli.Flag{Category: "caller",
		Name: "secure", AssignedMode: cli.AssignedNone,
		Usage: i18n.T(
			"use `--secure` to force https",
			"使用 `--secure` 开关强制使用https方式调用")}

	ForceFlag = &cli.Flag{Category: "caller",
		Name: "force", AssignedMode: cli.AssignedNone,
		Usage: i18n.T(
			"use `--force` to skip api and parameters check",
			"添加 `--force` 开关可跳过API与参数的合法性检查")}

	EndpointFlag = &cli.Flag{Category: "caller",
		Name: "endpoint", AssignedMode: cli.AssignedOnce,
		Usage: i18n.T(
			"use `--endpoint <endpoint>` to assign endpoint",
			"使用 `--endpoint <endpoint>` 来指定接入点地址")}

	VersionFlag = &cli.Flag{Category: "caller",
		Name: "version", AssignedMode: cli.AssignedOnce,
		Usage: i18n.T(
			"use `--version <YYYY-MM-DD>` to assign product api version",
			"使用 `--version <YYYY-MM-DD>` 来指定访问的API版本")}

	HeaderFlag = &cli.Flag{Category: "caller",
		Name: "header", AssignedMode: cli.AssignedRepeatable,
		Usage: i18n.T(
			"use `--header X-foo=bar` to add custom HTTP header, repeatable",
			"使用 `--header X-foo=bar` 来添加特定的HTTP头, 可多次添加")}

	BodyFlag = &cli.Flag{Category: "caller",
		Name: "body", AssignedMode: cli.AssignedOnce,
		Usage: i18n.T(
			"use `--body $(cat foo.json)` to assign http body in RESTful call",
			"使用 `--body $(cat foo.json)` 来指定在RESTful调用中的HTTP包体")}

	BodyFileFlag = &cli.Flag{Category: "caller",
		Name: "body-file", AssignedMode: cli.AssignedOnce, Hidden: true,
		Usage: i18n.T(
			"assign http body in Restful call with local file",
			"使用 `--body-file foo.json` 来指定输入包体")}

	PagerFlag = &cli.Flag{Category: "caller",
		Name: "all-pages", AssignedMode: cli.AssignedDefault, Hidden: true,
		Usage: i18n.T(
			"use `--all-pages` to merge pages for pageable APIs",
			"使用 `--all-pages` 在访问分页的API时合并分页")}

	AcceptFlag = &cli.Flag{Category: "caller",
		Name: "accept", AssignedMode: cli.AssignedOnce, Hidden: true,
		Usage: i18n.T(
			"add `--accept {json|xml}` to add Accept header",
			"使用 `--accept {json|xml}` 来指定Accept头")}

	//ContentTypeFlag = &cli.Flag{Category: "caller",
	//	Name: "content-type", AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		"add `--content-type {json|xml}` to add Accept header",
	//		"使用 `--content-type {json|xml}` 来指定Accept头")}
	//}

	//OutputTableRowFlag = cli.Flag{Category: "caller",
	//	Name: flagOutputTableRows, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to assign fields of table row", flagOutputTableRows),
	//		fmt.Sprintf("使用 `--%s` 指定表格行的内容", flagOutputTableRows))}
	//
	//OutputTableColsFlag = cli.Flag{Category: "caller",
	//	Name: flagOutputTableCols, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to assign table column names", flagOutputTableCols),
	//		fmt.Sprintf("使用 `--%s` 指定表格的列名", flagOutputTableCols))}
	//
	//WaitForExprFlag = cli.Flag{Category: "caller",
	//	Name: flagWaitForExpr, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to assign jmes expr", flagWaitForExpr),
	//		fmt.Sprintf("使用 `--%s` 指定jmes表达式", flagWaitForExpr))}
	//
	//WaitForTargetFlag = cli.Flag{Category: "caller",
	//	Name: flagWaitForTarget, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to assign target", flagWaitForTarget),
	//		fmt.Sprintf("使用 `--%s` 指定目标的值", flagWaitForTarget))}
	//
	//WaitTimeoutFlag = cli.Flag{Category: "caller",
	//	Name: flagWaitTimeout, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to set timeout(seconds)", flagWaitTimeout),
	//		fmt.Sprintf("使用 `--%s` 指定等待超时时间(秒)", flagWaitTimeout))}
	//
	//WaitIntervalFlag = cli.Flag{Category: "caller",
	//	Name: flagWaitInterval, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to set interval(seconds)", flagWaitInterval),
	//		fmt.Sprintf("使用 `--%s` 指定请求间隔时间(秒)", flagWaitInterval))}
	//
	//RetryTimeoutFlag = cli.Flag{Category: "caller",
	//	Name: flagRetryTimeout, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to set retry timeout(seconds)", flagRetryTimeout),
	//		fmt.Sprintf("使用 `--%s` 指定请求超时时间(秒)", flagRetryTimeout))}
	//
	//RetryCountFlag = cli.Flag{Category: "caller",
	//	Name: flagRetryCount, AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T(
	//		fmt.Sprintf("use `--%s` to set retry count", flagRetryCount),
	//		fmt.Sprintf("使用 `--%s` 指定重试次数", flagRetryCount))}
	//
	//WaiterFlag = cli.Flag{Category: "helper",
	//	Name: "waiter", AssignedMode: cli.AssignedKeyValues, Hidden: true,
	//	Usage: i18n.T(
	//		"use `--waiter expr=<jmespath> to=<expectValue> [timeout=<seconds>] [interval=<seconds>]` to wait until response value equals to expect",
	//		"使用 `--waiter expr=<jmespath> to=<expectValue> [timeout=<秒>] [interval=<秒>]` 来等待返回值"),
	// }
)
