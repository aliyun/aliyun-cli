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
	"os"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"

	"github.com/aliyun/aliyun-cli/cli"

	"github.com/stretchr/testify/assert"
)

func newProfile() *Profile {
	return &Profile{
		Name:            "default",
		Mode:            "",
		OutputFormat:    "json",
		Language:        "en",
		AccessKeyId:     "",
		AccessKeySecret: "",
		StsToken:        "",
		RamRoleName:     "",

		RamRoleArn:      "",
		RoleSessionName: "",
		PrivateKey:      "",
		KeyPairName:     "",
		ExpiredSeconds:  0,
		Verified:        "",
		RegionId:        "",
		Site:            "",
		ReadTimeout:     0,
		ConnectTimeout:  0,
		RetryCount:      0,
	}
}
func TestNewProfile(t *testing.T) {
	exp := newProfile()
	exp.Mode = AK
	p := NewProfile("default")
	p.Mode = AK
	assert.Equal(t, exp, &p)
}

func TestValidate(t *testing.T) {
	actual := newProfile()
	var err error
	err = actual.Validate()
	assert.EqualError(t, err, "region can't be empty")

	actual.RegionId = "as*ds%%s*"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid region as*ds%%s*")

	actual.RegionId = "cn-hangzhou"
	err = actual.Validate()
	assert.EqualError(t, err, "profile default is not configure yet, run `aliyun configure --profile default` first")

	actual.Mode = EcsRamRole
	err = actual.Validate()
	assert.Nil(t, err)

	actual.Mode = RamRoleArnWithEcs
	err = actual.Validate()
	assert.Nil(t, err)

	actual.Mode = AuthenticateMode("NoMode")
	err = actual.Validate()
	assert.EqualError(t, err, "invalid mode: NoMode")

}

func TestValidateWithAK(t *testing.T) {
	actual := newProfile()
	var err error
	actual.RegionId = "cn-hangzhou"
	actual.Mode = AK
	actual.AccessKeyId = "accessKeyId"
	actual.AccessKeySecret = "accessKeySecret"
	err = actual.Validate()
	assert.Nil(t, err)

}

func TestValidateWithRsaKeyPair(t *testing.T) {
	actual := newProfile()
	var err error
	actual.RegionId = "cn-hangzhou"
	actual.Mode = RsaKeyPair
	err = actual.Validate()
	assert.EqualError(t, err, "invalid private_key")
	actual.PrivateKey = "privateKey"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid key_pair_name")
	actual.KeyPairName = "keyPairName"
	err = actual.Validate()
	assert.Nil(t, err)
}
func TestValidateWithRamRoleArn(t *testing.T) {
	actual := newProfile()
	var err error
	actual.RegionId = "cn-hangzhou"
	actual.Mode = RamRoleArn
	err = actual.Validate()
	assert.NotNil(t, err)
	actual.AccessKeyId = "accessKeyId"
	actual.AccessKeySecret = "accessKeySecret"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid ram_role_arn")
	actual.RamRoleArn = "ramRoleArn"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid role_session_name")
	actual.RoleSessionName = "roleSessionName"
	err = actual.Validate()
	assert.Nil(t, err)
}

func TestValidateWithStsToken(t *testing.T) {
	actual := newProfile()
	var err error
	actual.Mode = StsToken
	actual.RegionId = "cn-hangzhou"

	err = actual.Validate()
	assert.NotNil(t, err)
	actual.AccessKeyId = "accessKeyId"
	actual.AccessKeySecret = "accessKeySecret"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid sts_token")
	actual.StsToken = "stsToken"
	err = actual.Validate()
	assert.Nil(t, err)
}
func TestGetParent(t *testing.T) {
	profile := newProfile()
	p := profile.GetParent()
	assert.Nil(t, p)
}
func TestOverwriteWithFlags(t *testing.T) {
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())
	resetEnv()
	actual := newProfile()

	ModeFlag(ctx.Flags()).SetAssigned(true)
	ModeFlag(ctx.Flags()).SetValue("ModeFlag")
	AccessKeyIdFlag(ctx.Flags()).SetAssigned(true)
	AccessKeyIdFlag(ctx.Flags()).SetValue("AccessKeyIdFlag")
	AccessKeySecretFlag(ctx.Flags()).SetAssigned(true)
	AccessKeySecretFlag(ctx.Flags()).SetValue("AccessKeySecretFlag")
	StsTokenFlag(ctx.Flags()).SetAssigned(true)
	StsTokenFlag(ctx.Flags()).SetValue("StsTokenFlag")
	RamRoleNameFlag(ctx.Flags()).SetAssigned(true)
	RamRoleNameFlag(ctx.Flags()).SetValue("RamRoleNameFlag")
	RamRoleArnFlag(ctx.Flags()).SetAssigned(true)
	RamRoleArnFlag(ctx.Flags()).SetValue("RamRoleArnFlag")
	RoleSessionNameFlag(ctx.Flags()).SetAssigned(true)
	RoleSessionNameFlag(ctx.Flags()).SetValue("RoleSessionNameFlag")
	KeyPairNameFlag(ctx.Flags()).SetAssigned(true)
	KeyPairNameFlag(ctx.Flags()).SetValue("KeyPairNameFlag")
	PrivateKeyFlag(ctx.Flags()).SetAssigned(true)
	PrivateKeyFlag(ctx.Flags()).SetValue("PrivateKeyFlag")
	RegionFlag(ctx.Flags()).SetAssigned(true)
	RegionFlag(ctx.Flags()).SetValue("RegionFlag")
	LanguageFlag(ctx.Flags()).SetAssigned(true)
	LanguageFlag(ctx.Flags()).SetValue("LanguageFlag")
	ReadTimeoutFlag(ctx.Flags()).SetAssigned(true)
	ReadTimeoutFlag(ctx.Flags()).SetValue("1")
	ConnectTimeoutFlag(ctx.Flags()).SetAssigned(true)
	ConnectTimeoutFlag(ctx.Flags()).SetValue("2")
	RetryCountFlag(ctx.Flags()).SetAssigned(true)
	RetryCountFlag(ctx.Flags()).SetValue("3")
	ExpiredSecondsFlag(ctx.Flags()).SetAssigned(true)
	ExpiredSecondsFlag(ctx.Flags()).SetValue("4")

	exp := &Profile{
		Name:            "default",
		Mode:            AuthenticateMode("ModeFlag"),
		OutputFormat:    "json",
		Language:        "LanguageFlag",
		AccessKeyId:     "AccessKeyIdFlag",
		AccessKeySecret: "AccessKeySecretFlag",
		StsToken:        "StsTokenFlag",
		RamRoleName:     "RamRoleNameFlag",
		RamRoleArn:      "RamRoleArnFlag",
		RoleSessionName: "RoleSessionNameFlag",
		PrivateKey:      "PrivateKeyFlag",
		KeyPairName:     "KeyPairNameFlag",
		ExpiredSeconds:  4,
		Verified:        "",
		RegionId:        "RegionFlag",
		Site:            "",
		ReadTimeout:     1,
		ConnectTimeout:  2,
		RetryCount:      3,
	}

	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

}
func TestOverwriteWithFlagsWithRegionIDEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	os.Setenv("REGION", "regionId")
	actual.OverwriteWithFlags(ctx)
	exp.RegionId = "regionId"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALICLOUD_REGION_ID", "alicloud")
	actual.OverwriteWithFlags(ctx)
	exp.RegionId = "alicloud"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABACLOUD_REGION_ID", "alibaba")
	actual.OverwriteWithFlags(ctx)
	exp.RegionId = "alibaba"
	assert.Equal(t, exp, actual)
}

func TestOverwriteWithFlagsWithStsTokenEnv(t *testing.T) {
	resetEnv()
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	os.Setenv("SECURITY_TOKEN", "stsToken")
	actual.OverwriteWithFlags(ctx)
	exp.StsToken = "stsToken"
	assert.Equal(t, exp, actual)
}

func TestOverwriteWithFlagsWithAccessKeySecretEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, buf2)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	os.Setenv("ACCESS_KEY_SECRET", "accessKeySecret")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeySecret = "accessKeySecret"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALICLOUD_ACCESS_KEY_SECRET", "alicloud")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeySecret = "alicloud"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABACLOUD_ACCESS_KEY_SECRET", "alibaba")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeySecret = "alibaba"
	assert.Equal(t, exp, actual)

}

func TestOverwriteWithFlagsWithAccessKeyIDEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, buf2)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	os.Setenv("ACCESS_KEY_ID", "accessKeyId")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeyId = "accessKeyId"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALICLOUD_ACCESS_KEY_ID", "alicloud")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeyId = "alicloud"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABACLOUD_ACCESS_KEY_ID", "alibaba")
	actual.OverwriteWithFlags(ctx)
	exp.AccessKeyId = "alibaba"
	assert.Equal(t, exp, actual)
}

func resetEnv() {
	os.Setenv("ALIBABACLOUD_ACCESS_KEY_ID", "")
	os.Setenv("ALICLOUD_ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ALIBABACLOUD_ACCESS_KEY_SECRET", "")
	os.Setenv("ALICLOUD_ACCESS_KEY_SECRET", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	os.Setenv("SECURITY_TOKEN", "")
	os.Setenv("ALIBABACLOUD_REGION_ID", "")
	os.Setenv("ALICLOUD_REGION_ID", "")
	os.Setenv("REGION", "")
}

func TestGetClient(t *testing.T) {
	actual := newProfile()
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, buf2)
	AddFlags(ctx.Flags())
	actual.RetryCount = 2

	actual.Mode = AK
	client, err := actual.GetClient(ctx)
	assert.Nil(t, client)
	assert.NotNil(t, err)

	actual.Mode = RamRoleArnWithEcs
	client, err = actual.GetClient(ctx)
	assert.Nil(t, client)
	assert.NotNil(t, err)

	actual.Mode = StsToken
	client, err = actual.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	actual.Mode = RamRoleArn
	client, err = actual.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	actual.Mode = EcsRamRole
	client, err = actual.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	actual.Mode = RsaKeyPair
	client, err = actual.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// config to client
	actual.Mode = StsToken
	actual.ReadTimeout = 2
	actual.ConnectTimeout = 2
	SkipSecureVerify(ctx.Flags()).SetAssigned(true)
	client, err = actual.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, 2, client.GetConfig().MaxRetryTime)
	assert.Equal(t, float64(2), client.GetReadTimeout().Seconds())
	assert.Equal(t, float64(2), client.GetConnectTimeout().Seconds())
	assert.True(t, client.GetHTTPSInsecure())
}
func TestGetClientByAK(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()

	actual.AccessKeyId = "accessKeyId"
	client, err := actual.GetClientByAK(config)
	assert.Nil(t, client)
	assert.EqualError(t, err, "AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")

	actual.AccessKeySecret = "accessKeySecret"
	client, err = actual.GetClientByAK(config)
	assert.Nil(t, client)
	assert.EqualError(t, err, "default RegionId is empty! run `aliyun configure` first")

	actual.RegionId = "cn-hangzhou"
	client, err = actual.GetClientByAK(config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

}
func TestGetClientWithNoError(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()

	// GetClientBySts
	client, err := actual.GetClientBySts(config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByRoleArn
	client, err = actual.GetClientByRoleArn(config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByEcsRamRole
	client, err = actual.GetClientByEcsRamRole(config)
	assert.Nil(t, err)
	assert.NotNil(t, client)

	// GetClientByPrivateKey
	client, err = actual.GetClientByPrivateKey(config)
	assert.Nil(t, err)
	assert.NotNil(t, client)
}

func TestGetClientByRamRoleArnWithEcs(t *testing.T) {
	actual := newProfile()
	config := sdk.NewConfig()
	client, err := actual.GetClientByRamRoleArnWithEcs(config)
	assert.Nil(t, client)
	assert.NotNil(t, err)
}

func TestValidateAk(t *testing.T) {
	actual := newProfile()
	err := actual.ValidateAK()
	assert.EqualError(t, err, "invalid access_key_id: ")
	actual.AccessKeyId = "accessKeyId"
	err = actual.ValidateAK()
	assert.EqualError(t, err, "invaild access_key_secret: ")
	actual.AccessKeySecret = "accessKeySecret"
	err = actual.ValidateAK()
	assert.Nil(t, err)
}
func TestIsRegion(t *testing.T) {
	assert.False(t, IsRegion("#$adf"))
	assert.True(t, IsRegion("2kf"))
}

func TestAutoModeRecognition(t *testing.T) {

	p := &Profile{AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret"}
	assert.Equal(t, AuthenticateMode(""), p.Mode)
	AutoModeRecognition(p)
	assert.Equal(t, AK, p.Mode)

	p = &Profile{AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret", StsToken: "stsToken"}
	AutoModeRecognition(p)
	assert.Equal(t, StsToken, p.Mode)

	p = &Profile{AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret", RamRoleArn: "ramRoleArn"}
	AutoModeRecognition(p)
	assert.Equal(t, RamRoleArn, p.Mode)

	p = &Profile{PrivateKey: "privateKey", KeyPairName: "keyPairName"}
	AutoModeRecognition(p)
	assert.Equal(t, RsaKeyPair, p.Mode)

	p = &Profile{RamRoleName: "ramRoleName"}
	AutoModeRecognition(p)
	assert.Equal(t, EcsRamRole, p.Mode)

	p = &Profile{AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret", StsToken: "stsToken", Mode: AK}
	AutoModeRecognition(p)
	assert.Equal(t, AK, p.Mode)

}
