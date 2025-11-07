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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestDoConfigureSet(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())

	originhook := hookLoadOrCreateConfiguration
	originhookSave := hookSaveConfigurationWithContext
	defer func() {
		hookLoadOrCreateConfiguration = originhook
		hookSaveConfigurationWithContext = originhookSave
	}()
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}
	err := doConfigureSet(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "fail to set configuration: region can't be empty", err.Error())

	//testcase2
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}

	//ERROR
	hookSaveConfigurationWithContext = func(fn func(ctx *cli.Context, config *Configuration) error) func(ctx *cli.Context, config *Configuration) error {
		return func(ctx *cli.Context, config *Configuration) error {
			return nil
		}
	}

	stdout.Reset()
	stderr.Reset()
	err = doConfigureSet(ctx)
	assert.NotNil(t, err)
	assert.Equal(t, "fail to set configuration: region can't be empty", err.Error())

	//AK
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"},
				},
			}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	//StsToken
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: StsToken, StsToken: "StsToken", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	//RamRoleArn
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: RamRoleArn, RoleSessionName: "RoleSessionName", RamRoleArn: "RamRoleArn", ExternalId: "ExternalId", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	//EcsRamRole
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: EcsRamRole, RamRoleName: "RamRoleName", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	// RamRoleArnWithEcs
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: RamRoleArnWithEcs, RamRoleName: "RamRoleName", RoleSessionName: "RoleSessionName", RamRoleArn: "RamRoleArn", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	// RsaKeyPair
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default", Profiles: []Profile{
					{Name: "default", Mode: RsaKeyPair, KeyPairName: "KeyPairName", PrivateKey: "PrivateKey", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	// External
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: External, ProcessCommand: "process command", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())
	// OIDC
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: OIDC, OIDCProviderARN: "OIDCProviderARN", OIDCTokenFile: "OIDCTokenFile",
						RoleSessionName: "RoleSessionName", RamRoleArn: "RamRoleArn", AccessKeyId: "default_aliyun_access_key_id",
						AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou", StsRegion: "eu-central-1"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())

	// CloudSSO
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: CloudSSO,
						CloudSSOAccessConfig: "CloudSSOAccessConfig",
						CloudSSOAccountId:    "CloudSSOAccountId",
						CloudSSOSignInUrl:    "CloudSSOSignInUrl",
						OutputFormat:         "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())
}

func TestDoConfigureSetWithMock(t *testing.T) {
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(stdout, stderr)
	AddFlags(ctx.Flags())

	originhook := hookLoadOrCreateConfiguration
	originhookSave := hookSaveConfigurationWithContext
	defer func() {
		hookLoadOrCreateConfiguration = originhook
		hookSaveConfigurationWithContext = originhookSave
	}()

	// EndpointType
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: AK, EndpointType: "vpc", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	hookSaveConfigurationWithContext = func(fn func(ctx *cli.Context, config *Configuration) error) func(ctx *cli.Context, config *Configuration) error {
		return func(ctx *cli.Context, config *Configuration) error {
			return nil
		}
	}
	endpointTypeFlag := EndpointTypeFlag(ctx.Flags())
	if endpointTypeFlag != nil {
		endpointTypeFlag.SetAssigned(true)
		endpointTypeFlag.SetValue("testabc")
	}
	stdout.Reset()
	stderr.Reset()
	doConfigureSet(ctx)
	assert.Empty(t, stdout.String())
}
