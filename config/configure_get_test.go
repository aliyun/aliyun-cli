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
	"bytes"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestDoConfigureGet(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	originhook := hookLoadConfigurationWithContext
	defer func() {
		hookLoadConfigurationWithContext = originhook
	}()

	//testcase 1
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}

	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	err := doConfigureGet(ctx, []string{})
	assert.Equal(t, "load configuration failed. Run `aliyun configure` to set up", err.Error())

	//testcase 2
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("ddd")
	err = doConfigureGet(ctx, []string{})
	assert.Equal(t, "profile ddd not found", err.Error())

	//testcase 3
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	ctx.Flags().Flags()[1].SetAssigned(false)
	err = doConfigureGet(ctx, []string{"profile", "mode", "access-key-id", "access-key-secret", "sts-token", "ram-role-name", "ram-role-arn", "role-session-name", "external-id", "private-key", "key-pair-name", "region", "language"})
	assert.Nil(t, err)
	assert.Equal(t, "profile=default\nmode=AK\naccess-key-id=*************************_id\naccess-key-secret=*****************************ret\nsts-token=\nram-role-name=\nram-role-arn=\nrole-session-name=\nexternal-id=\nprivate-key=\nkey-pair-name=\nlanguage=\n\n", stdout.String())

	//TESTCASE 4
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("default")
	doConfigureGet(ctx, []string{})
	assert.Equal(t, "{\n\t\"name\": \"default\",\n\t\"mode\": \"AK\",\n\t\"access_key_id\": \"default_aliyun_access_key_id\",\n\t\"access_key_secret\": \"default_aliyun_access_key_secret\",\n\t\"output_format\": \"json\"\n}\n\n", stdout.String())

	//testcase 5
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("default")
	doConfigureGet(ctx, []string{"mode", "profile", "access-key-id", "language"})
	assert.Equal(t, "mode=AK\nprofile=default\naccess-key-id=*************************_id\nlanguage=\n\n", stdout.String())
}

func TestDoConfigureGetCloudSSO(t *testing.T) {
	// 设置测试环境
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())
	originhook := hookLoadConfigurationWithContext
	defer func() {
		hookLoadConfigurationWithContext = originhook
	}()

	// 创建包含 CloudSSO 相关配置的 Profile
	hookLoadConfigurationWithContext = func(fn func(ctx *cli.Context) (*Configuration, error)) func(ctx *cli.Context) (*Configuration, error) {
		return func(ctx *cli.Context) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{
						Name:                 "default",
						Mode:                 AK,
						CloudSSOSignInUrl:    "https://signin.example.com",
						CloudSSOAccessConfig: "access-config-example",
						CloudSSOAccountId:    "account-123456",
					},
				},
			}, nil
		}
	}

	// 测试 CloudSSOSignInUrlFlagName
	stdout.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOSignInUrlFlagName})
	assert.Equal(t, "cloud-sso-sign-in-url=https://signin.example.com\n\n", stdout.String())

	// 测试 CloudSSOAccessConfigFlagName
	stdout.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOAccessConfigFlagName})
	assert.Equal(t, "cloud-sso-access-config=access-config-example\n\n", stdout.String())

	// 测试 CloudSSOAccountIdFlagName
	stdout.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOAccountIdFlagName})
	assert.Equal(t, "cloud-sso-account-id=account-123456\n\n", stdout.String())

	// 测试同时获取所有 CloudSSO 相关配置
	stdout.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOSignInUrlFlagName, CloudSSOAccessConfigFlagName, CloudSSOAccountIdFlagName})
	expected := "cloud-sso-sign-in-url=https://signin.example.com\n" +
		"cloud-sso-access-config=access-config-example\n" +
		"cloud-sso-account-id=account-123456\n\n"
	assert.Equal(t, expected, stdout.String())
}
