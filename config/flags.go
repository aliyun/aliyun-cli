/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(NewModeFlag())
	fs.Add(NewProfileFlag())
	fs.Add(NewLanguageFlag())
	fs.Add(NewRegionFlag())

	fs.Add(NewAccessKeyIdFlag())
	fs.Add(NewAccessKeySecretFlag())
	fs.Add(NewStsTokenFlag())
	fs.Add(NewRamRoleNameFlag())
	fs.Add(NewRamRoleArnFlag())
	fs.Add(NewRoleSessionNameFlag())
	fs.Add(NewPrivateKeyFlag())
	fs.Add(NewKeyPairNameFlag())
	fs.Add(NewRetryTimeoutFlag())
	fs.Add(NewRetryCountFlag())
	fs.Add(NewSkipSecureVerify())
}

func ProfileFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("profile")
}

func ModeFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("mode")
}

func AccessKeyIdFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("access-key-id")
}

func AccessKeySecretFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("access-key-secret")
}

func StsTokenFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("sts-token")
}

func RamRoleNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("ram-role-name")
}

func RamRoleArnFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("ram-role-arn")
}

func RoleSessionNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("role-session-name")
}

func PrivateKeyFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("private-key")
}

func KeyPairNameFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("key-pair-name")
}

func RegionFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("region")
}

func LanguageFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("language")
}

func RetryTimeoutFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("retry-timeout")
}

func RetryCountFlag(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("retry-count")
}

func SkipSecureVerify(fs *cli.FlagSet) *cli.Flag {
	return fs.Get("skip-secure-verify")
}

//var OutputFlag = &cli.Flag{Category: "config",
//	Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
//	Usage: i18n.T(
//		"* assign output format, only support json",
//		"* 指定输出格式, 目前仅支持Json")}

//varfs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
//Usage: i18n.T("assign site, support china/international/japan", "")})

func NewProfileFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name:         "profile",
		Shorthand:    'p',
		DefaultValue: "default", Persistent: true,
		Short: i18n.T(
			"use `--profile <profileName>` to select profile",
			"使用 `--profile <profileName>` 指定操作的配置集")}
}

func NewModeFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "mode", DefaultValue: "AK", Persistent: true,
		Short: i18n.T(
			"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` to assign authenticate mode",
			"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` 指定认证方式")}
}

func NewAccessKeyIdFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "access-key-id", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--access-key-id <AccessKeyId>` to assign AccessKeyId, required in AK/StsToken/RamRoleArn mode",
			"使用 `--access-key-id <AccessKeyId>` 指定AccessKeyId")}
}

func NewAccessKeySecretFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "access-key-secret", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--access-key-secret <AccessKeySecret>` to assign AccessKeySecret",
			"使用 `--access-key-secret <AccessKeySecret>` 指定AccessKeySecret")}
}

func NewStsTokenFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "sts-token", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--sts-token <StsToken>` to assign StsToken",
			"使用 `--sts-token <StsToken>` 指定StsToken")}
}

func NewRamRoleNameFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "ram-role-name", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--ram-role-name <RamRoleName>` to assign RamRoleName",
			"使用 `--ram-role-name <RamRoleName>` 指定RamRoleName")}
}

func NewRamRoleArnFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "ram-role-arn", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--ram-role-arn <RamRoleArn>` to assign RamRoleArn",
			"使用 `--ram-role-arn <RamRoleArn>` 指定RamRoleArn")}
}

func NewRoleSessionNameFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "role-session-name", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--role-session-name <RoleSessionName>` to assign RoleSessionName",
			"使用 `--role-session-name <RoleSessionName>` 指定RoleSessionName")}
}

func NewPrivateKeyFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "private-key", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--private-key <PrivateKey>` to assign RSA PrivateKey",
			"使用 `--private-key <PrivateKey>` 指定RSA私钥")}
}

func NewKeyPairNameFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "key-pair-name", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--key-pair-name <KeyPairName>` to assign KeyPairName",
			"使用 `--key-pair-name <KeyPairName>` 指定KeyPairName")}
}

func NewRegionFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "region", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--region <regionId>` to assign region",
			"使用 `--region <regionId>` 来指定访问大区")}
}

func NewLanguageFlag() *cli.Flag {
	return &cli.Flag{Category: "config",
		Name: "language", AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"use `--language [en|zh]` to assign language",
			"使用 `--language [en|zh]` 来指定语言")}
}

func NewRetryTimeoutFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: "retry-timeout", AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"use `--retry-timeout <seconds>` to set retry timeout(seconds)",
			"使用 `--retry-timeout <seconds>` 指定请求超时时间(秒)"),
	}
}

func NewRetryCountFlag() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: "retry-count", AssignedMode: cli.AssignedOnce, Hidden: true,
		Short: i18n.T(
			"use `--retry-count <count>` to set retry count",
			"使用 `--retry-count <count>` 指定重试次数"),
	}
}

func NewSkipSecureVerify() *cli.Flag {
	return &cli.Flag{Category: "caller",
		Name: "skip-secure-verify", AssignedMode: cli.AssignedNone, Hidden: true,
		Short: i18n.T(
			"use `--skip-secure-verify` to skip https certification validate",
			"使用 `--skip-secure-verify` 跳过https的证书校验",
		),
	}
}