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
	"bytes"
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
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}

	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
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
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
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
	stderr.Reset()
	err = configureDelete.Run(ctx, []string{"delete"})
	assert.Nil(t, err)
	assert.Equal(t, "\x1b[1;31mmissing --profile <profileName>\n\x1b[0m\x1b[1;33m\nusage:\n  aliyun configure delete --profile <profileName>\n\x1b[0m", stderr.String())

	//testcase
	cmd := NewConfigureCommand()
	excmd.Run = cmd.Run
	assert.ObjectsAreEqualValues(excmd, cmd)

	//testcase
	w.Reset()
	stderr.Reset()
	err = cmd.Run(ctx, []string{"configure"})
	assert.Empty(t, w.String())
	assert.NotNil(t, err)

	//testcase
	w.Reset()
	stderr.Reset()
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
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:            "default",
						Mode:            AK,
						AccessKeyId:     "default_aliyun_access_key_id",
						AccessKeySecret: "default_aliyun_access_key_secret",
						OutputFormat:    "json",
					},
					{
						Name:            "aaa",
						Mode:            AK,
						AccessKeyId:     "sdf",
						AccessKeySecret: "ddf",
						OutputFormat:    "json",
					},
				},
			}, nil
		}
	}
	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
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
	assert.Equal(t, "Access Key Id [**********_id]: Access Key Secret [**************ret]: Sts Region []: Ram Role Arn [RamRoleArn]: Role Session Name [RoleSessionName]: Expired Seconds [900]: ", w.String())
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

func TestConfigureExternal(t *testing.T) {
	w := new(bytes.Buffer)
	err := configureExternal(w, &Profile{Name: "default", Mode: External, ProcessCommand: "process command", RegionId: "cn-hangzhou", OutputFormat: "json"})
	assert.Equal(t, "Process Command [process command]: ", w.String())
	assert.Nil(t, err)
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
