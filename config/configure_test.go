/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"io"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func TestNewConfigureCommand(t *testing.T) {
	excmd := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode <AuthenticateMode> --profile <profileName>",
	}
	excmd.AddSubCommand(NewConfigureGetCommand())
	excmd.AddSubCommand(NewConfigureSetCommand())
	excmd.AddSubCommand(NewConfigureListCommand())
	excmd.AddSubCommand(NewConfigureDeleteCommand())

	cmd := NewConfigureCommand()
	excmd.Run = cmd.Run
	assert.ObjectsAreEqualValues(excmd, cmd)
}

func TestDoConfigure(t *testing.T) {
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(w io.Writer) (Configuration, error)) func(w io.Writer) (Configuration, error) {
		return func(w io.Writer) (Configuration, error) {
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
	assert.Equal(t, "Configuring profile 'profile' in 'AK' authenticate mode...\nAccess Key Id []: Access Key Secret []: Default Region Id []: Default Output Format [json]: json (Only support json))\nDefault Language [zh|en] en: Saving profile[profile] ...Done.\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\nAccessKeyId/AccessKeySecret is empty! run `aliyun configure` first\n-----------------------------------------------\n!!! Configure Failed please configure again !!!\n-----------------------------------------------\n", w.String())
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
