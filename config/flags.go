// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

const (
	ProfileFlagName         = "profile"
	ModeFlagName            = "mode"
	AccessKeyIdFlagName     = "access-key-id"
	AccessKeySecretFlagName = "access-key-secret"
	StsTokenFlagName        = "sts-token"
	StsRegionFlagName       = "sts-region"
	RamRoleNameFlagName     = "ram-role-name"
	RamRoleArnFlagName      = "ram-role-arn"
	RoleSessionNameFlagName = "role-session-name"
	SourceProfileFlagName   = "source-profile"
	PrivateKeyFlagName      = "private-key"
	KeyPairNameFlagName     = "key-pair-name"
	RegionFlagName          = "region"
	LanguageFlagName        = "language"
	ReadTimeoutFlagName     = "read-timeout"
	ConnectTimeoutFlagName  = "connect-timeout"
	RetryCountFlagName      = "retry-count"
	SkipSecureVerifyName    = "skip-secure-verify"
	ConfigurePathFlagName   = "config-path"
	ExpiredSecondsFlagName  = "expired-seconds"
	ProcessCommandFlagName  = "process-command"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(NewModeFlag())
	fs.Add(NewProfileFlag())
	fs.Add(NewLanguageFlag())
	fs.Add(NewRegionFlag())
	fs.Add(NewConfigurePathFlag())
	fs.Add(NewAccessKeyIdFlag())
	fs.Add(NewAccessKeySecretFlag())
	fs.Add(NewStsTokenFlag())
	fs.Add(NewStsRegionFlag())
	fs.Add(NewRamRoleNameFlag())
	fs.Add(NewRamRoleArnFlag())
	fs.Add(NewRoleSessionNameFlag())
	fs.Add(NewPrivateKeyFlag())
	fs.Add(NewKeyPairNameFlag())
	fs.Add(NewReadTimeoutFlag())
	fs.Add(NewConnectTimeoutFlag())
	fs.Add(NewRetryCountFlag())
	fs.Add(NewSkipSecureVerify())
	fs.Add(NewExpiredSecondsFlag())
	fs.Add(NewProcessCommandFlag())
}

func ConnectTimeoutFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ConnectTimeoutFlagName)
}

func ProfileFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ProfileFlagName)
}

func ModeFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ModeFlagName)
}

func AccessKeyIdFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(AccessKeyIdFlagName)
}

func AccessKeySecretFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(AccessKeySecretFlagName)
}

func StsTokenFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(StsTokenFlagName)
}

func StsRegionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(StsRegionFlagName)
}

func RamRoleNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RamRoleNameFlagName)
}

func RamRoleArnFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RamRoleArnFlagName)
}

func SourceProfileFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(SourceProfileFlagName)
}

func RoleSessionNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RoleSessionNameFlagName)
}

func PrivateKeyFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(PrivateKeyFlagName)
}

func KeyPairNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(KeyPairNameFlagName)
}

func RegionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RegionFlagName)
}

func LanguageFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(LanguageFlagName)
}

func ReadTimeoutFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ReadTimeoutFlagName)
}

func RetryCountFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(RetryCountFlagName)
}

func SkipSecureVerify(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(SkipSecureVerifyName)
}
func ConfigurePathFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ConfigurePathFlagName)
}
func ExpiredSecondsFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ExpiredSecondsFlagName)
}
func ProcessCommandFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get(ProcessCommandFlagName)
}

//var OutputFlag = &cli.Flag{Category: "config",
//	Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
//	Usage: i18n.T(
//		"* assign output format, only support json",
//		"* 指定输出格式, 目前仅支持Json")}

//varfs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
//Usage: i18n.T("assign site, support china/international/japan", "")})

func NewProfileFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         ProfileFlagName,
		Shorthand:    'p',
		DefaultValue: "default", Persistent: true,
		Short: i18n.T(
			"use `--profile <profileName>` to select profile",
			"使用 `--profile <profileName>` 指定操作的配置集")}
}

func NewModeFlag() *cli.Flag {
	return &cli.Flag{
		Category: "config",
		Name:     ModeFlagName, DefaultValue: "AK", Persistent: true,
		Short: i18n.T(
			"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName}` to assign authenticate mode",
			"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName}` 指定认证方式")}
}

func NewAccessKeyIdFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         AccessKeyIdFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--access-key-id <AccessKeyId>` to assign AccessKeyId, required in AK/StsToken/RamRoleArn mode",
			"使用 `--access-key-id <AccessKeyId>` 指定AccessKeyId")}
}

func NewAccessKeySecretFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         AccessKeySecretFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--access-key-secret <AccessKeySecret>` to assign AccessKeySecret",
			"使用 `--access-key-secret <AccessKeySecret>` 指定AccessKeySecret")}
}

func NewStsTokenFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         StsTokenFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--sts-token <StsToken>` to assign StsToken",
			"使用 `--sts-token <StsToken>` 指定StsToken")}
}

func NewStsRegionFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         StsRegionFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--sts-region <StsRegion>` to assign StsRegion",
			"使用 `--sts-region <StsRegion>` 指定StsRegion")}
}

func NewRamRoleNameFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         RamRoleNameFlagName,
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"use `--ram-role-name <RamRoleName>` to assign RamRoleName",
			"使用 `--ram-role-name <RamRoleName>` 指定RamRoleName")}
}

func NewRamRoleArnFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         RamRoleArnFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--ram-role-arn <RamRoleArn>` to assign RamRoleArn",
			"使用 `--ram-role-arn <RamRoleArn>` 指定RamRoleArn")}
}

func NewRoleSessionNameFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name:         RoleSessionNameFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--role-session-name <RoleSessionName>` to assign RoleSessionName",
			"使用 `--role-session-name <RoleSessionName>` 指定RoleSessionName")}
}
func NewExpiredSecondsFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         ExpiredSecondsFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--expired-seconds <seconds>` to specify expiration time",
			"使用 `--expired-seconds <seconds>` 指定凭证过期时间")}
}
func NewPrivateKeyFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name:         PrivateKeyFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--private-key <PrivateKey>` to assign RSA PrivateKey",
			"使用 `--private-key <PrivateKey>` 指定RSA私钥")}
}

func NewKeyPairNameFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name:         KeyPairNameFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--key-pair-name <KeyPairName>` to assign KeyPairName",
			"使用 `--key-pair-name <KeyPairName>` 指定KeyPairName")}
}

func NewProcessCommandFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         ProcessCommandFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--process-command <ProcessCommand>` to specify external program execution command",
			"使用 `--process-command <ProcessCommand>` 指定外部程序运行命令",
		),
	}
}

func NewRegionFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name:         RegionFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--region <regionId>` to assign region",
			"使用 `--region <regionId>` 来指定访问大区")}
}

func NewLanguageFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         LanguageFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--language [en|zh]` to assign language",
			"使用 `--language [en|zh]` 来指定语言")}
}
func NewConfigurePathFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "config",
		Name:         ConfigurePathFlagName,
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"use `--config-path` to specify the configuration file path",
			"使用 `--config-path` 指定配置文件路径",
		),
	}
}
func NewReadTimeoutFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ReadTimeoutFlagName,
		AssignedMode: cli.AssignedOnce,
		Aliases:      []string{"retry-timeout"},
		Short: i18n.T(
			"use `--read-timeout <seconds>` to set I/O timeout(seconds)",
			"使用 `--read-timeout <seconds>` 指定I/O超时时间(秒)"),
	}
}

func NewConnectTimeoutFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         ConnectTimeoutFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--connect-timeout <seconds>` to set connect timeout(seconds)",
			"使用 `--connect-timeout <seconds>` 指定请求连接超时时间(秒)"),
	}
}

func NewRetryCountFlag() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         RetryCountFlagName,
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--retry-count <count>` to set retry count",
			"使用 `--retry-count <count>` 指定重试次数"),
	}
}

func NewSkipSecureVerify() *cli.Flag {
	return &cli.Flag{
		Category:     "caller",
		Name:         SkipSecureVerifyName,
		AssignedMode: cli.AssignedNone,
		Persistent:   true,
		Short: i18n.T(
			"use `--skip-secure-verify` to skip https certification validate [Not recommended]",
			"使用 `--skip-secure-verify` 跳过https的证书校验 [不推荐使用]",
		),
	}
}
