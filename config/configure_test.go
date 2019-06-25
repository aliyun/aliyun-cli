/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"io"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func TestNewConfigureCommand(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config Configuration) error) func(config Configuration) error {
		return func(config Configuration) error {
			return nil
		}
	}
	excmd := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode <AuthenticateMode> --profile <profileName>",
	}
	configureGet := NewConfigureGetCommand()
	configureSet := NewConfigureSetCommand()
	configureList := NewConfigureListCommand()
	configureDelete := NewConfigureDeleteCommand()
	excmd.AddSubCommand(configureGet)
	excmd.AddSubCommand(configureSet)
	excmd.AddSubCommand(configureList)
	excmd.AddSubCommand(configureDelete)

	//testcase
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	AddFlags(ctx.Flags())

	//testcase
	err := configureGet.Run(ctx, []string{"get"})
	assert.Nil(t, err)
	assert.Equal(t, "\n", w.String())

	//testcase
	w.Reset()
	err = configureSet.Run(ctx, []string{"set"})
	assert.Nil(t, err)
	assert.Equal(t, "\x1b[1;31mfail to set configuration: region can't be empty\x1b[0m", w.String())

	//testcase
	w.Reset()
	err = configureList.Run(ctx, []string{"list"})
	assert.Nil(t, err)
	assert.Equal(t, "Profile   | Credential         | Valid   | Region           | Language\n--------- | ------------------ | ------- | ---------------- | --------\ndefault * | AK:***_id          | Invalid |                  | \naaa       | AK:******          | Invalid |                  | \n", w.String())

	//testcase
	w.Reset()
	err = configureDelete.Run(ctx, []string{"delete"})
	assert.Nil(t, err)
	assert.Equal(t, "\x1b[1;31mmissing --profile <profileName>\n\x1b[0m\x1b[1;33m\nusage:\n  aliyun configure delete --profile <profileName>\n\x1b[0m", w.String())

	//testcase
	cmd := NewConfigureCommand()
	excmd.Run = cmd.Run
	assert.ObjectsAreEqualValues(excmd, cmd)

	//testcase
	w.Reset()
	err = cmd.Run(ctx, []string{"configure"})
	assert.Empty(t, w.String())
	assert.NotNil(t, err)

	//testcase
	w.Reset()
	err = cmd.Run(ctx, []string{})
	assert.Nil(t, err)
	assert.Equal(t, "Configuring profile 'default' in 'AK' authenticate mode...\nAccess Key Id [*************************_id]: Access Key Secret [*****************************ret]: Default Region Id []: Default Output Format [json]: json (Only support json)\nDefault Language [zh|en] : Saving profile[default] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\ndefault RegionId is empty! run `aliyun configure` first\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n", w.String())
}

func TestDoConfigure(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(path string, w io.Writer) (Configuration, error)) func(path string, w io.Writer) (Configuration, error) {
		return func(path string, w io.Writer) (Configuration, error) {
			return Configuration{CurrentProfile: "default", Profiles: []Profile{Profile{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"}, Profile{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config Configuration) error) func(config Configuration) error {
		return func(config Configuration) error {
			return nil
		}
	}
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	AddFlags(ctx.Flags())
	err := doConfigure(ctx, "profile", "AK")
	assert.Nil(t, err)
	assert.Equal(t, "Configuring profile 'profile' in 'AK' authenticate mode...\nAccess Key Id []: Access Key Secret []: Default Region Id []: Default Output Format [json]: json (Only support json)\nDefault Language [zh|en] en: Saving profile[profile] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\nAccessKeyId/AccessKeySecret is empty! run `aliyun configure` first\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n", w.String())
	w.Reset()

	err = doConfigure(ctx, "", "")
	assert.Nil(t, err)
	assert.Equal(t, "Configuring profile 'default' in 'AK' authenticate mode...\nAccess Key Id [*************************_id]: Access Key Secret [*****************************ret]: Default Region Id []: Default Output Format [json]: json (Only support json)\nDefault Language [zh|en] : Saving profile[default] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\ndefault RegionId is empty! run `aliyun configure` first\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n", w.String())
	w.Reset()

	err = doConfigure(ctx, "", "StsToken")
	assert.Nil(t, err)
	assert.True(t, strings.Contains(w.String(), "Warning: You are changing the authentication type of profile 'default' from 'AK' to 'StsToken'\nConfiguring profile 'default' in 'StsToken' authenticate mode...\nAccess Key Id [*************************_id]: Access Key Secret [*****************************ret]: Sts Token []: Default Region Id []: Default Output Format [json]: json (Only support json)\nDefault Language [zh|en] : Saving profile[default] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n"))
	w.Reset()
}

func TestConfigureAK(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureAK(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Nil(t, err)
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: ", w.String())
}

func TestConfigureStsToken(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureStsToken(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", StsToken: "ststoken", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: Sts Token [ststoken]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRamRoleArn(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRamRoleArn(w, &Profile{Name: "default", Mode: AK, AccessKeyId: "access_key_id", AccessKeySecret: "access_key_secret", RamRoleArn: "RamRoleArn", RoleSessionName: "RoleSessionName", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: Ram Role Arn [RamRoleArn]: Role Session Name [RoleSessionName]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureEcsRamRole(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureEcsRamRole(w, &Profile{Name: "default", Mode: AK, RamRoleName: "RamRoleName", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Ecs Ram Role [RamRoleName]: ", w.String())
	assert.Nil(t, err)
}

func TestConfigureRsaKeyPair(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureRsaKeyPair(w, &Profile{Name: "default", Mode: AK, RamRoleName: "RamRoleName", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Rsa Private Key File: ", w.String())
	if runtime.GOOS == "windows" {
		assert.EqualError(t, err, "read key file  failed open : The system cannot find the file specified.")
	} else {
		assert.EqualError(t, err, "read key file  failed open : no such file or directory")
	}
}
func TestReadInput(t *testing.T) {
	assert.Equal(t, "default", ReadInput("default"))
}
func TestMosaicString(t *testing.T) {
	assert.Equal(t, "****rX", MosaicString("IamMrX", 2))
	assert.Equal(t, "******", MosaicString("IamMrX", 10))
}
func TestGetLastChars(t *testing.T) {

	assert.Equal(t, "rX", GetLastChars("IamMrX", 2))
	assert.Equal(t, "******", GetLastChars("IamMrX", 10))
}
