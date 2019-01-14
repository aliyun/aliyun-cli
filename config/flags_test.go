package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/stretchr/testify/assert"
	"testing"
)

var NewMethodFlag func() *cli.Flag

func NewFlagTest(t *testing.T, fn func() *cli.Flag, expectFlag *cli.Flag, values []string) {
	flag := fn()
	value, ok := flag.GetValue()
	assert.Equal(t, false, ok)
	assert.Equal(t, value, "")
	assert.Nil(t, flag.GetValues())
	assert.Equal(t, false, flag.IsAssigned())
	assert.Subset(t, flag.GetFormations(), values)
	assert.Len(t, flag.GetFormations(), len(values))
	assert.Equal(t, expectFlag, flag)

}

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
				"use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` to assign authenticate mode",
				"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` 指定认证方式"),
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
		newRetryTimeoutFlag = &cli.Flag{
			Category:     "caller",
			Name:         RetryTimeoutFlagName,
			AssignedMode: cli.AssignedOnce,
			Hidden:       true,
			Short: i18n.T(
				"use `--retry-timeout <seconds>` to set retry timeout(seconds)",
				"使用 `--retry-timeout <seconds>` 指定请求超时时间(秒)"),
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
			Hidden:       true,
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
			Hidden:       true,
			Persistent:   true,
			Short: i18n.T(
				"use `--skip-secure-verify` to skip https certification validate",
				"使用 `--skip-secure-verify` 跳过https的证书校验",
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
	)

	NewFlagTest(t, NewProfileFlag, newProfileFlag, []string{"--profile", "-p"})
	NewFlagTest(t, NewModeFlag, newModeFlag, []string{"--mode"})
	NewFlagTest(t, NewAccessKeyIdFlag, newAccessKeyIDFlag, []string{"--access-key-id"})
	NewFlagTest(t, NewAccessKeySecretFlag, newAccessKeySecretFlag, []string{"--access-key-secret"})
	NewFlagTest(t, NewStsTokenFlag, newStsTokenFlag, []string{"--sts-token"})
	NewFlagTest(t, NewRamRoleNameFlag, newRamRoleNameFlag, []string{"--ram-role-name"})
	NewFlagTest(t, NewRamRoleArnFlag, newRamRoleArnFlag, []string{"--ram-role-arn"})
	NewFlagTest(t, NewRoleSessionNameFlag, newRoleSessionNameFlag, []string{"--role-session-name"})
	NewFlagTest(t, NewPrivateKeyFlag, newPrivateKeyFlag, []string{"--private-key"})
	NewFlagTest(t, NewKeyPairNameFlag, newKeyPairNameFlag, []string{"--key-pair-name"})
	NewFlagTest(t, NewRegionFlag, newRegionFlag, []string{"--region"})
	NewFlagTest(t, NewLanguageFlag, newLanguageFlag, []string{"--language"})
	NewFlagTest(t, NewRetryTimeoutFlag, newRetryTimeoutFlag, []string{"--retry-timeout"})
	NewFlagTest(t, NewRetryCountFlag, newRetryCountFlag, []string{"--retry-count"})
	NewFlagTest(t, NewSkipSecureVerify, newSkipSecureVerify, []string{"--skip-secure-verify"})
}
