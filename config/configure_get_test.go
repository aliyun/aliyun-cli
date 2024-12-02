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
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())
	originhook := hookLoadConfiguration
	defer func() {
		hookLoadConfiguration = originhook
	}()

	//testcase 1
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}

	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	doConfigureGet(ctx, []string{})
	assert.Equal(t, "\x1b[1;31mload configuration failed error\x1b[0m\n", stderr.String())

	//testcase 2
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("")
	doConfigureGet(ctx, []string{})
	assert.Equal(t, "\x1b[1;31mprofile  not found!\x1b[0m\n", stderr.String())

	//testcase 3
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	stderr.Reset()
	ctx.Flags().Flags()[1].SetAssigned(false)
	doConfigureGet(ctx, []string{"profile", "mode", "access-key-id", "access-key-secret", "sts-token", "ram-role-name", "ram-role-arn", "role-session-name", "external-id", "private-key", "key-pair-name", "region", "language"})
	assert.Equal(t, "profile=default\nmode=AK\naccess-key-id=*************************_id\naccess-key-secret=*****************************ret\nsts-token=\nram-role-name=\nram-role-arn=\nrole-session-name=\nexternal-id=\nprivate-key=\nkey-pair-name=\nlanguage=\n\n", w.String())

	//TESTCASE 4
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("default")
	doConfigureGet(ctx, []string{})
	assert.Equal(t, "{\n\t\"name\": \"default\",\n\t\"mode\": \"AK\",\n\t\"access_key_id\": \"default_aliyun_access_key_id\",\n\t\"access_key_secret\": \"default_aliyun_access_key_secret\",\n\t\"output_format\": \"json\"\n}\n\n", w.String())

	//testcase 5
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	stderr.Reset()
	ctx.Flags().Get("profile").SetAssigned(true)
	ctx.Flags().Get("profile").SetValue("default")
	doConfigureGet(ctx, []string{"mode", "profile", "access-key-id", "language"})
	assert.Equal(t, "mode=AK\nprofile=default\naccess-key-id=*************************_id\nlanguage=\n\n", w.String())
}

func TestDoConfigureGetCloudSSO(t *testing.T) {
	// 设置测试环境
	w := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w, stderr)
	AddFlags(ctx.Flags())
	originhook := hookLoadConfiguration
	defer func() {
		hookLoadConfiguration = originhook
	}()

	// 创建包含 CloudSSO 相关配置的 Profile
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
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
	w.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOSignInUrlFlagName})
	assert.Equal(t, "cloud-sso-sign-in-url=https://signin.example.com\n\n", w.String())

	// 测试 CloudSSOAccessConfigFlagName
	w.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOAccessConfigFlagName})
	assert.Equal(t, "cloud-sso-access-config=access-config-example\n\n", w.String())

	// 测试 CloudSSOAccountIdFlagName
	w.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOAccountIdFlagName})
	assert.Equal(t, "cloud-sso-account-id=account-123456\n\n", w.String())

	// 测试同时获取所有 CloudSSO 相关配置
	w.Reset()
	stderr.Reset()
	doConfigureGet(ctx, []string{CloudSSOSignInUrlFlagName, CloudSSOAccessConfigFlagName, CloudSSOAccountIdFlagName})
	expected := "cloud-sso-sign-in-url=https://signin.example.com\n" +
		"cloud-sso-access-config=access-config-example\n" +
		"cloud-sso-account-id=account-123456\n\n"
	assert.Equal(t, expected, w.String())
}
