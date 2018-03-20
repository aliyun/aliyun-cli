/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

var ProfileFlag = cli.Flag{Category: "config",
	Name: "profile", DefaultValue: "default", Persistent: true,
	Usage: i18n.T(
		"use `--profile <profileName>` to select profile",
		"使用 `--profile <profileName>` 来指定操作的配置集")}

var ModeFlag = cli.Flag{Category: "config",
	Name: "mode", Persistent: true,
	Usage: i18n.T(
		"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` to assign authenticate mode",
		"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` 指定认证方式")}

var AccessKeyIdFlag = cli.Flag{Category: "config",
	Name: "access-key-id", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use --access-key-id <AccessKeyId> to assign AccessKeyId, required in AK/StsToken/RamRoleArn mode",
		"使用 --access-key-id <AccessKeyId> 来指定AccessKeyId, AK/Sts")}

var AccessKeySecretFlag = cli.Flag{Category: "config",
	Name: "access-key-secret", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign AccessKeySecret, required in AK/StsToken/RamRoleArn mode",
		"")}

var StsTokenFlag = cli.Flag{Category: "config",
	Name: "sts-token", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign StsToken, required in StsToken mode",
		"")}

var RamRoleNameFlag = cli.Flag{Category: "config",
	Name: "ram-role-name", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign RamRoleName, required in RamRoleArn/EcsRamRole mode",
		"")}

var RamRoleArnFlag = cli.Flag{Category: "config",
	Name: "ram-role-arn", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign RamRoleArn, required in RamRoleArn mode",
		"")}

var RoleSessionNameFlag = cli.Flag{Category: "config",
	Name: "role-session-name", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign RoleSessionName, required in RamRoleArn mode",
		"")}

var PrivateKeyFlag = cli.Flag{Category: "config",
	Name: "private-key", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign PrivateKey, required in RsaKeyPair mode",
		"")}

var KeyPairNameFlag = cli.Flag{Category: "config",
	Name: "key-pair-name", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"assign KeyPairName, required in RsaKeyPair mode",
		"")}

var RegionFlag = cli.Flag{Category: "config",
	Name: "region", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use --region <regionId> to assign region",
		"使用 --region <regionId> 来指定访问地域")}

var LanguageFlag = cli.Flag{Category: "config",
	Name: "language", AssignedMode: cli.AssignedOnce,
	Usage: i18n.T(
		"use --language [en|zh] to assign language",
		"使用 --language [en|zh] 来指定语言")}

var OutputFlag = cli.Flag{Category: "config",
	Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
	Usage: i18n.T(
		"* assign output format, only support json",
		"")}

//varfs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
//Usage: i18n.T("assign site, support china/international/japan", "")})