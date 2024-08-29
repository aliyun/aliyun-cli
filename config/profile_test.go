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
	"os"
	"testing"

	"github.com/aliyun/aliyun-cli/cli"

	"github.com/stretchr/testify/assert"
)

func newCtx() *cli.Context {
	buf := new(bytes.Buffer)
	buf2 := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, buf2)
	AddFlags(ctx.Flags())
	return ctx
}

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

	actual.Mode = External
	err = actual.Validate()
	assert.EqualError(t, err, "invalid process_command")

	actual.Mode = CredentialsURI
	err = actual.Validate()
	assert.EqualError(t, err, "invalid credentials_uri")

	actual.Mode = ChainableRamRoleArn
	err = actual.Validate()
	assert.EqualError(t, err, "invalid source_profile")

	actual.SourceProfile = "source"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid ram_role_arn")
	actual.RamRoleArn = "arn"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid role_session_name")
	actual.RoleSessionName = "rsn"
	err = actual.Validate()
	assert.Nil(t, err)

	// default
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

func TestValidateWithOIDC(t *testing.T) {
	actual := newProfile()
	var err error
	actual.Mode = OIDC
	actual.RegionId = "cn-hangzhou"

	err = actual.Validate()
	assert.EqualError(t, err, "invalid oidc_provider_arn")
	actual.OIDCProviderARN = "oidc_provider_arn"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid oidc_token_file")
	actual.OIDCTokenFile = "/path/to/oidc/token/file"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid ram_role_arn")
	actual.RamRoleArn = "ramrolearn"
	err = actual.Validate()
	assert.EqualError(t, err, "invalid role_session_name")
	actual.RoleSessionName = "rsn"
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

func TestGetStsEndpoint(t *testing.T) {
	assert.Equal(t, "sts.aliyuncs.com", getSTSEndpoint(""))
	assert.Equal(t, "sts.cn-hangzhou.aliyuncs.com", getSTSEndpoint("cn-hangzhou"))
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

	p = &Profile{ProcessCommand: "external"}
	AutoModeRecognition(p)
	assert.Equal(t, External, p.Mode)

	p = &Profile{OIDCProviderARN: "oidc_provider_arn", OIDCTokenFile: "/path/to/tokenfile", RamRoleArn: "ram/role/arn"}
	AutoModeRecognition(p)
	assert.Equal(t, OIDC, p.Mode)
}

func TestGetCredentialByAK(t *testing.T) {
	actual := newProfile()

	actual.Mode = AK
	actual.AccessKeyId = "accessKeyId"
	credential, err := actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.EqualError(t, err, "AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")

	actual.AccessKeySecret = "accessKeySecret"
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.EqualError(t, err, "default RegionId is empty! run `aliyun configure` first")

	actual.RegionId = "cn-hangzhou"
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, err)
	assert.NotNil(t, credential)

	assert.Equal(t, "access_key", *credential.GetType())
}

func TestGetCredentialBySts(t *testing.T) {
	actual := newProfile()

	actual.Mode = StsToken
	credential, err := actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.EqualError(t, err, "the access key id is empty")

	actual.AccessKeyId = "akid"
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.EqualError(t, err, "the access key secret is empty")

	actual.AccessKeySecret = "aksecret"
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.EqualError(t, err, "the security token is empty")

	actual.StsToken = "ststoken"
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, err)
	assert.NotNil(t, credential)

	assert.Equal(t, "sts", *credential.GetType())
}

func TestGetCredentialByRamRoleArn(t *testing.T) {
	// p := newProfile()
	// p.Mode = RamRoleArn

	// credential, err := p.GetCredential(newCtx())
	// assert.Nil(t, credential)
	// assert.EqualError(t, err, "AccessKeyId cannot be empty")

	// p.AccessKeyId = "akid"
	// credential, err = p.GetCredential(newCtx())
	// assert.Nil(t, credential)
	// assert.EqualError(t, err, "AccessKeySecret cannot be empty")
}

func TestGetProfileWithChainable(t *testing.T) {
	cf := NewConfiguration()
	sourceProfile := newProfile()
	sourceProfile.Name = "source"
	sourceProfile.Mode = AK
	sourceProfile.AccessKeyId = "invalidAKID"
	sourceProfile.AccessKeySecret = "invalidAKSecret"
	sourceProfile.RegionId = "cn-hangzhou"
	cf.PutProfile(*sourceProfile)

	p := newProfile()
	p.parent = cf
	p.Mode = ChainableRamRoleArn
	p.SourceProfile = "source"
	p.RegionId = "cn-hangzhou"
	p.RamRoleArn = "acs:ram::test:role/test"
	p.RoleSessionName = "sessionname"

	c, err := p.GetCredential(newCtx(), nil)
	assert.NotNil(t, c)
	assert.Nil(t, err)
}
