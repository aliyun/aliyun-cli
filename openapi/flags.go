// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(NewSecureFlag())
	fs.Add(NewInsecureFlag())
	fs.Add(NewForceFlag())
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
	fs.Add(NewYesFlag())
	fs.Add(NewQueryFlag())
	fs.Add(NewRoaFlag())
	fs.Add(NewMethodFlag())
	fs.Add(NewUserAgentFlag())
	fs.Add(NewCliAIModeFlag())
	fs.Add(NewCliNoAIModeFlag())
}

const (
	SecureFlagName      = "secure"
	InsecureFlagName    = "insecure"
	ForceFlagName       = "force"
	VersionFlagName     = "version"
	HeaderFlagName      = "header"
	BodyFlagName        = "body"
	BodyFileFlagName    = "body-file"
	AcceptFlagName      = "accept"
	RoaFlagName         = "roa"
	DryRunFlagName      = "dryrun"
	QuietFlagName       = "quiet"
	YesFlagName         = "yes"
	QueryFlagName       = "cli-query"
	OutputFlagName      = "output"
	MethodFlagName      = "method"
	UserAgentFlagName   = "user-agent"
	CliAIModeFlagName   = "cli-ai-mode"
	CliNoAIModeFlagName = "no-cli-ai-mode"
)

func OutputFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(OutputFlagName)
}

func SecureFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(SecureFlagName)
}

func InsecureFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(InsecureFlagName)
}

func ForceFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ForceFlagName)
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

func MethodFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(MethodFlagName)
}

func QueryFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(QueryFlagName)
}

func YesFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(YesFlagName)
}

func NewYesFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         YesFlagName,
		Shorthand:    'y',
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"skip safety policy confirmation prompt (for non-interactive/agent use)",
			"跳过安全策略的确认提示（用于非交互式/Agent 场景）",
		),
	}
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
	return &cli.Flag{
		Category:     "caller",
		Name:         SecureFlagName,
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--secure` to force https",
			"使用 `--secure` 开关强制使用https方式调用")}
}

func NewInsecureFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         InsecureFlagName,
		AssignedMode: cli.AssignedNone,
		Hidden:       true,
		Short: i18n.T(
			"use `--insecure` to force http(not recommend)",
			"使用 `--insecure` 开关强制使用http方式调用（不推荐）")}
}

func NewForceFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ForceFlagName,
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"use `--force` to skip api and parameters check",
			"添加 `--force` 开关可跳过API与参数的合法性检查")}
}

func NewVersionFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         VersionFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--version <YYYY-MM-DD>` to assign product api version",
			"使用 `--version <YYYY-MM-DD>` 来指定访问的API版本")}
}

func NewHeaderFlag() *cli.Flag {
	return &cli.Flag{
		Category: "caller",
		Name:     HeaderFlagName, AssignedMode: cli.AssignedRepeatable,
		Short: i18n.T(
			"use `--header X-foo=bar` to add custom HTTP header, repeatable",
			"使用 `--header X-foo=bar` 来添加特定的HTTP头, 可多次添加")}
}

func NewBodyFlag() *cli.Flag {
	return &cli.Flag{
		Category: "caller",
		Name:     BodyFlagName, AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--body $(cat foo.json)` to assign http body in RESTful call",
			"使用 `--body $(cat foo.json)` 来指定在RESTful调用中的HTTP包体")}
}

func NewBodyFileFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         BodyFileFlagName,
		AssignedMode: cli.AssignedOnce,
		Hidden:       true,
		Short: i18n.T(
			"assign http body in Restful call with local file",
			"使用 `--body-file foo.json` 来指定输入包体")}
}

func NewAcceptFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         AcceptFlagName,
		AssignedMode: cli.AssignedOnce,
		Hidden:       true,
		Short: i18n.T(
			"add `--accept {json|xml}` to add Accept header",
			"使用 `--accept {json|xml}` 来指定Accept头")}
}

func NewRoaFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         RoaFlagName,
		AssignedMode: cli.AssignedOnce,
		Hidden:       true,
		Short: i18n.T(
			"use `--roa {GET|PUT|POST|DELETE}` to assign restful call.[DEPRECATED]",
			"使用 `--roa {GET|PUT|POST|DELETE}` 使用restful方式调用[已过期]",
		),
	}
}

func NewDryRunFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
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
	return &cli.Flag{
		Category:     "caller",
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

func NewMethodFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         MethodFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"add `--method {GET|POST}` to assign rpc call method.",
			"使用 `--method {GET|POST}` 来指定 RPC 请求的 Method",
		),
	}
}

func NewQueryFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         QueryFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--cli-query <jmespath>` to filter output with JMESPath expression",
			"使用 `--cli-query <jmespath>` 通过 JMESPath 表达式过滤输出结果",
		),
	}
}

func NewUserAgentFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         UserAgentFlagName,
		AssignedMode: cli.AssignedOnce,
		Hidden:       true,
		Short: i18n.T(
			"use `--user-agent <value>` to append custom User-Agent identifier",
			"使用 `--user-agent <value>` 追加自定义 User-Agent 标识",
		),
	}
}

func UserAgentFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(UserAgentFlagName)
}

func NewCliAIModeFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         CliAIModeFlagName,
		AssignedMode: cli.AssignedNone,
		Short: i18n.T(
			"for this command only, append AI-mode User-Agent segment (skills from configure ai-mode) even if global ai-mode is off",
			"仅本次命令追加 AI 模式 UA 段（skills 来自 configure ai-mode），即使全局 ai-mode 未开启",
		),
	}
}

func NewCliNoAIModeFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         CliNoAIModeFlagName,
		AssignedMode: cli.AssignedNone,
		Hidden:       true,
		Short: i18n.T(
			"for this command only, do not append AI-mode User-Agent segment even if global ai-mode is on",
			"仅本次命令不追加 AI 模式 UA 段，即使全局 ai-mode 已开启",
		),
	}
}

func CliAIModeFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(CliAIModeFlagName)
}

func CliNoAIModeFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(CliNoAIModeFlagName)
}

// CliAIOverrides returns per-command AI User-Agent behavior from root flags.
// If both --no-cli-ai-mode and --cli-ai-mode are present, --no-cli-ai-mode wins.
func CliAIOverrides(fs *cli.FlagSet) (forceOn bool, forceOff bool) {
	if fs == nil {
		return false, false
	}
	if CliNoAIModeFlag(fs) != nil && CliNoAIModeFlag(fs).IsAssigned() {
		return false, true
	}
	if CliAIModeFlag(fs) != nil && CliAIModeFlag(fs).IsAssigned() {
		return true, false
	}
	return false, false
}
