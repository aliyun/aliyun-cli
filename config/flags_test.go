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
package config

import (
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/stretchr/testify/assert"
)

func TestAddFlag(t *testing.T) {
	var (
		newProfileFlag = &cli.Flag{
			Category:     "config",
			Name:         ProfileFlagName,
			Shorthand:    'p',
			DefaultValue: "default",
			Persistent:   true,
			Short: i18n.T(
				"use `--profile <profileName>` to select profile",
				"使用 `--profile <profileName>` 指定操作的配置集",
			),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			AssignedMode: 0,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
		}
		newModeFlag = &cli.Flag{
			Category:     "config",
			Name:         ModeFlagName,
			DefaultValue: "AK",
			Persistent:   true,
			Short: i18n.T(
				"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName}` to assign authenticate mode",
				"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair|RamRoleArnWithRoleName}` 指定认证方式"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			AssignedMode: 0,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
		}
		newAccessKeyIDFlag = &cli.Flag{
			Category:     "config",
			Name:         AccessKeyIdFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--access-key-id <AccessKeyId>` to assign AccessKeyId, required in AK/StsToken/RamRoleArn mode",
				"使用 `--access-key-id <AccessKeyId>` 指定AccessKeyId"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newAccessKeySecretFlag = &cli.Flag{
			Category:     "config",
			Name:         AccessKeySecretFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--access-key-secret <AccessKeySecret>` to assign AccessKeySecret",
				"使用 `--access-key-secret <AccessKeySecret>` 指定AccessKeySecret"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newStsTokenFlag = &cli.Flag{
			Category:     "config",
			Name:         StsTokenFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--sts-token <StsToken>` to assign StsToken",
				"使用 `--sts-token <StsToken>` 指定StsToken"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newRamRoleNameFlag = &cli.Flag{
			Category:     "config",
			Name:         RamRoleNameFlagName,
			AssignedMode: cli.AssignedOnce,
			Persistent:   true,
			Short: i18n.T(
				"use `--ram-role-name <RamRoleName>` to assign RamRoleName",
				"使用 `--ram-role-name <RamRoleName>` 指定RamRoleName"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
		}
		newRamRoleArnFlag = &cli.Flag{
			Category:     "config",
			Name:         RamRoleArnFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--ram-role-arn <RamRoleArn>` to assign RamRoleArn",
				"使用 `--ram-role-arn <RamRoleArn>` 指定RamRoleArn"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newSourceProfileFlag = &cli.Flag{
			Category:     "config",
			Name:         SourceProfileFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--source-profile <SourceProfile>` to assign SourceProfile",
				"使用 `--source-profile <SourceProfile>` 指定SourceProfile"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newRoleSessionNameFlag = &cli.Flag{
			Category:     "config",
			Name:         RoleSessionNameFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--role-session-name <RoleSessionName>` to assign RoleSessionName",
				"使用 `--role-session-name <RoleSessionName>` 指定RoleSessionName"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newExternalIdFlag = &cli.Flag{
			Category:     "config",
			Name:         ExternalIdFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--external-id <ExternalId>` to assign ExternalId",
				"使用 `--external-id <ExternalId>` 指定ExternalId"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newPrivateKeyFlag = &cli.Flag{
			Category:     "config",
			Name:         PrivateKeyFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--private-key <PrivateKey>` to assign RSA PrivateKey",
				"使用 `--private-key <PrivateKey>` 指定RSA私钥"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newKeyPairNameFlag = &cli.Flag{
			Category:     "config",
			Name:         KeyPairNameFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--key-pair-name <KeyPairName>` to assign KeyPairName",
				"使用 `--key-pair-name <KeyPairName>` 指定KeyPairName"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newProcessCommandFlag = &cli.Flag{
			Category:     "config",
			Name:         ProcessCommandFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--process-command <ProcessCommand>` to specify external program execution command",
				"使用 `--process-command <ProcessCommand>` 指定外部程序运行命令",
			),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newRegionFlag = &cli.Flag{
			Category:     "config",
			Name:         RegionFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--region <regionId>` to assign region",
				"使用 `--region <regionId>` 来指定访问大区"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newLanguageFlag = &cli.Flag{
			Category:     "config",
			Name:         LanguageFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--language [en|zh]` to assign language",
				"使用 `--language [en|zh]` 来指定语言"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newReadTimeoutFlag = &cli.Flag{
			Category:     "caller",
			Name:         ReadTimeoutFlagName,
			AssignedMode: cli.AssignedOnce,
			Hidden:       false,
			Short: i18n.T(
				"use `--read-timeout <seconds>` to set I/O timeout(seconds)",
				"使用 `--read-timeout <seconds>` 指定I/O超时时间(秒)"),
			Long:         nil,
			Required:     false,
			Aliases:      []string{"retry-timeout"},
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newConnectTimeFlag = &cli.Flag{
			Category:     "caller",
			Name:         ConnectTimeoutFlagName,
			AssignedMode: cli.AssignedOnce,
			Hidden:       false,
			Short: i18n.T(
				"use `--connect-timeout <seconds>` to set connect timeout(seconds)",
				"使用 `--connect-timeout <seconds>` 指定请求连接超时时间(秒)"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newRetryCountFlag = &cli.Flag{
			Category:     "caller",
			Name:         RetryCountFlagName,
			AssignedMode: cli.AssignedOnce,
			Hidden:       false,
			Short: i18n.T(
				"use `--retry-count <count>` to set retry count",
				"使用 `--retry-count <count>` 指定重试次数"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
		newSkipSecureVerify = &cli.Flag{
			Category:     "caller",
			Name:         SkipSecureVerifyName,
			AssignedMode: cli.AssignedNone,
			Hidden:       false,
			Persistent:   true,
			Short: i18n.T(
				"use `--skip-secure-verify` to skip https certification validate [Not recommended]",
				"使用 `--skip-secure-verify` 跳过https的证书校验 [不推荐使用]",
			),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
		}
		newExpiredSecondsFlag = &cli.Flag{
			Category:     "config",
			Name:         ExpiredSecondsFlagName,
			AssignedMode: cli.AssignedOnce,
			Short: i18n.T(
				"use `--expired-seconds <seconds>` to specify expiration time",
				"使用 `--expired-seconds <seconds>` 指定凭证过期时间"),
			Long:         nil,
			Required:     false,
			Aliases:      nil,
			Hidden:       false,
			Validate:     nil,
			Fields:       nil,
			ExcludeWith:  nil,
			Shorthand:    0,
			DefaultValue: "",
			Persistent:   false,
		}
	)
	f := NewExpiredSecondsFlag()
	assert.Equal(t, newExpiredSecondsFlag, f)

	f = NewProfileFlag()
	assert.Equal(t, newProfileFlag, f)

	f = NewModeFlag()
	assert.Equal(t, newModeFlag, f)

	f = NewAccessKeyIdFlag()
	assert.Equal(t, newAccessKeyIDFlag, f)

	f = NewAccessKeySecretFlag()
	assert.Equal(t, newAccessKeySecretFlag, f)

	f = NewStsTokenFlag()
	assert.Equal(t, newStsTokenFlag, f)

	f = NewRamRoleNameFlag()
	assert.Equal(t, newRamRoleNameFlag, f)

	f = NewRamRoleArnFlag()
	assert.Equal(t, newRamRoleArnFlag, f)

	f = NewSourceProfileFlag()
	assert.Equal(t, newSourceProfileFlag, f)

	f = NewRoleSessionNameFlag()
	assert.Equal(t, newRoleSessionNameFlag, f)

	f = NewExternalIdFlag()
	assert.Equal(t, newExternalIdFlag, f)

	f = NewPrivateKeyFlag()
	assert.Equal(t, newPrivateKeyFlag, f)

	f = NewKeyPairNameFlag()
	assert.Equal(t, newKeyPairNameFlag, f)

	f = NewProcessCommandFlag()
	assert.Equal(t, newProcessCommandFlag, f)

	f = NewRegionFlag()
	assert.Equal(t, newRegionFlag, f)

	f = NewLanguageFlag()
	assert.Equal(t, newLanguageFlag, f)

	f = NewReadTimeoutFlag()
	assert.Equal(t, newReadTimeoutFlag, f)

	f = NewConnectTimeoutFlag()
	assert.Equal(t, newConnectTimeFlag, f)

	f = NewRetryCountFlag()
	assert.Equal(t, newRetryCountFlag, f)

	f = NewSkipSecureVerify()
	assert.Equal(t, newSkipSecureVerify, f)

}

func TestNewOAuthSiteTypeFlag(t *testing.T) {
	var a = NewOAuthSiteTypeFlag()
	assert.Equal(t, OAuthSiteTypeName, a.Name)
	assert.Equal(t, "config", a.Category)
}

func TestAddFlags(t *testing.T) {
	flagSet := cli.NewFlagSet()
	AddFlags(flagSet)
	// 结果包含OAuthSiteTypeName
	assert.NotNil(t, flagSet.Get(OAuthSiteTypeName))
}
