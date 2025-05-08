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
	fs := cli.NewFlagSet()
	w := new(bytes.Buffer)
	AddFlags(fs)
	originhook := hookLoadConfiguration
	originhookSave := hookSaveConfiguration
	defer func() {
		hookLoadConfiguration = originhook
		hookSaveConfiguration = originhookSave
	}()
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{}, errors.New("error")
		}
	}
	doConfigureSet(w, fs)
	assert.Equal(t, "\x1b[1;31mload configuration failed error\x1b[0m", w.String())

	//testcase2
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}

	//ERROR
	hookSaveConfiguration = func(fn func(config *Configuration) error) func(config *Configuration) error {
		return func(config *Configuration) error {
			return nil
		}
	}

	w.Reset()
	doConfigureSet(w, fs)
	assert.Equal(t, "\x1b[1;31mfail to set configuration: region can't be empty\x1b[0m", w.String())

	//AK
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
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
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	//StsToken
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: StsToken, StsToken: "StsToken", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	//RamRoleArn
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: RamRoleArn, RoleSessionName: "RoleSessionName", RamRoleArn: "RamRoleArn", ExternalId: "ExternalId", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	//EcsRamRole
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{{Name: "default", Mode: EcsRamRole, RamRoleName: "RamRoleName", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"}, {Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	// RamRoleArnWithEcs
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: RamRoleArnWithEcs, RamRoleName: "RamRoleName", RoleSessionName: "RoleSessionName", RamRoleArn: "RamRoleArn", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	// RsaKeyPair
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default", Profiles: []Profile{
					{Name: "default", Mode: RsaKeyPair, KeyPairName: "KeyPairName", PrivateKey: "PrivateKey", AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())

	// External
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "default",
				Profiles: []Profile{
					{Name: "default", Mode: External, ProcessCommand: "process command", OutputFormat: "json", RegionId: "cn-hangzhou"},
					{Name: "aaa", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())
	// OIDC
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
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
	w.Reset()
	doConfigureSet(w, fs)

	// CloudSSO
	hookLoadConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
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
	w.Reset()
	doConfigureSet(w, fs)
	assert.Empty(t, w.String())
}
