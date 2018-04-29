/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func AddFlags(fs *cli.FlagSet) {
	fs.Add(ModeFlag)
	fs.Add(ProfileFlag)
	fs.Add(LanguageFlag)
	fs.Add(RegionFlag)

	fs.Add(AccessKeyIdFlag)
	fs.Add(AccessKeySecretFlag)
	fs.Add(StsTokenFlag)
	fs.Add(RamRoleNameFlag)
	fs.Add(RamRoleArnFlag)
	fs.Add(RoleSessionNameFlag)
	fs.Add(PrivateKeyFlag)
	fs.Add(KeyPairNameFlag)
	fs.Add(RetryTimeoutFlag)
	fs.Add(RetryCountFlag)
	fs.Add(SkipSecureVerify)
}

var ProfileFlag = &cli.Flag{Category: "config",
	Name:         "profile",
	Shorthand:    'p',
	DefaultValue: "default", Persistent: true,
	Short: i18n.T(
		"use `--profile <profileName>` to select profile",
		"使用 `--profile <profileName>` 指定操作的配置集")}

var ModeFlag = &cli.Flag{Category: "config",
	Name: "mode", DefaultValue: "AK", Persistent: true,
	Short: i18n.T(
		"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` to assign authenticate mode",
		"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` 指定认证方式")}

var AccessKeyIdFlag = &cli.Flag{Category: "config",
	Name: "access-key-id", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--access-key-id <AccessKeyId>` to assign AccessKeyId, required in AK/StsToken/RamRoleArn mode",
		"使用 `--access-key-id <AccessKeyId>` 指定AccessKeyId")}

var AccessKeySecretFlag = &cli.Flag{Category: "config",
	Name: "access-key-secret", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--access-key-secret <AccessKeySecret>` to assign AccessKeySecret",
		"使用 `--access-key-secret <AccessKeySecret>` 指定AccessKeySecret")}

var StsTokenFlag = &cli.Flag{Category: "config",
	Name: "sts-token", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--sts-token <StsToken>` to assign StsToken",
		"使用 `--sts-token <StsToken>` 指定StsToken")}

var RamRoleNameFlag = &cli.Flag{Category: "config",
	Name: "ram-role-name", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--ram-role-name <RamRoleName>` to assign RamRoleName",
		"使用 `--ram-role-name <RamRoleName>` 指定RamRoleName")}

var RamRoleArnFlag = &cli.Flag{Category: "config",
	Name: "ram-role-arn", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--ram-role-arn <RamRoleArn>` to assign RamRoleArn",
		"使用 `--ram-role-arn <RamRoleArn>` 指定RamRoleArn")}

var RoleSessionNameFlag = &cli.Flag{Category: "config",
	Name: "role-session-name", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--role-session-name <RoleSessionName>` to assign RoleSessionName",
		"使用 `--role-session-name <RoleSessionName>` 指定RoleSessionName")}

var PrivateKeyFlag = &cli.Flag{Category: "config",
	Name: "private-key", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--private-key <PrivateKey>` to assign RSA PrivateKey",
		"使用 `--private-key <PrivateKey>` 指定RSA私钥")}

var KeyPairNameFlag = &cli.Flag{Category: "config",
	Name: "key-pair-name", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--key-pair-name <KeyPairName>` to assign KeyPairName",
		"使用 `--key-pair-name <KeyPairName>` 指定KeyPairName")}

var RegionFlag = &cli.Flag{Category: "config",
	Name: "region", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--region <regionId>` to assign region",
		"使用 `--region <regionId>` 来指定访问大区")}

var LanguageFlag = &cli.Flag{Category: "config",
	Name: "language", AssignedMode: cli.AssignedOnce,
	Short: i18n.T(
		"use `--language [en|zh]` to assign language",
		"使用 `--language [en|zh]` 来指定语言")}

var RetryTimeoutFlag = &cli.Flag{Category: "caller",
	Name: "retry-timeout", AssignedMode: cli.AssignedOnce, Hidden: true,
	Short: i18n.T(
		"use `--retry-timeout <seconds>` to set retry timeout(seconds)",
		"使用 `--retry-timeout <seconds>` 指定请求超时时间(秒)"),
}

var RetryCountFlag = &cli.Flag{Category: "caller",
	Name: "retry-count", AssignedMode: cli.AssignedOnce, Hidden: true,
	Short: i18n.T(
		"use `--retry-count <count>` to set retry count",
		"使用 `--retry-count <count>` 指定重试次数"),
}

var SkipSecureVerify = &cli.Flag{Category: "caller",
	Name: "skip-secure-verify", AssignedMode: cli.AssignedNone, Hidden: true,
	Short: i18n.T(
		"use `--skip-secure-verify` to skip https certification validate",
		"使用 `--skip-secure-verify` 跳过https的证书校验",
	),
}

//var OutputFlag = &cli.Flag{Category: "config",
//	Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
//	Usage: i18n.T(
//		"* assign output format, only support json",
//		"* 指定输出格式, 目前仅支持Json")}

//varfs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
//Usage: i18n.T("assign site, support china/international/japan", "")})
