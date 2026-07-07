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
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
	"time"

	credentialsv2 "github.com/aliyun/credentials-go/credentials"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cloudsso"

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

		RamRoleArn:           "",
		RoleSessionName:      "",
		PrivateKey:           "",
		KeyPairName:          "",
		ExpiredSeconds:       0,
		Verified:             "",
		RegionId:             "",
		Site:                 "",
		ReadTimeout:          0,
		ConnectTimeout:       0,
		RetryCount:           0,
		CloudSSOAccessConfig: "",
		CloudSSOAccountId:    "",
		CloudSSOSignInUrl:    "",
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
	assert.EqualError(t, err, "process_command is not configured for profile 'default'. Run `aliyun configure --profile default --mode External` to set it")

	actual.Mode = CredentialsURI
	err = actual.Validate()
	assert.EqualError(t, err, "credentials_uri is not configured for profile 'default'. Run `aliyun configure --profile default --mode CredentialsURI` or set ALIBABA_CLOUD_CREDENTIALS_URI environment variable")

	actual.Mode = ChainableRamRoleArn
	err = actual.Validate()
	assert.EqualError(t, err, "source_profile is not configured for profile 'default'. Run `aliyun configure --profile default --mode ChainableRamRoleArn` to set it")

	actual.SourceProfile = "source"
	err = actual.Validate()
	assert.EqualError(t, err, "ram_role_arn is not configured for profile 'default'. Run `aliyun configure --profile default --mode ChainableRamRoleArn` to set it")
	actual.RamRoleArn = "arn"
	err = actual.Validate()
	assert.EqualError(t, err, "role_session_name is not configured for profile 'default'. Run `aliyun configure --profile default --mode ChainableRamRoleArn` to set it")
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
	assert.EqualError(t, err, "private_key is not configured for profile 'default'. Run `aliyun configure --profile default --mode RsaKeyPair` to set it")
	actual.PrivateKey = "privateKey"
	err = actual.Validate()
	assert.EqualError(t, err, "key_pair_name is not configured for profile 'default'. Run `aliyun configure --profile default --mode RsaKeyPair` to set it")
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
	assert.EqualError(t, err, "ram_role_arn is not configured for profile 'default'. Run `aliyun configure --profile default --mode RamRoleArn` to set it")
	actual.RamRoleArn = "ramRoleArn"
	err = actual.Validate()
	assert.EqualError(t, err, "role_session_name is not configured for profile 'default'. Run `aliyun configure --profile default --mode RamRoleArn` to set it")
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
	assert.EqualError(t, err, "sts_token is not configured for profile 'default'. Run `aliyun configure --profile default --mode StsToken` to set it")
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
	assert.EqualError(t, err, "oidc_provider_arn is not configured for profile 'default'. Run `aliyun configure --profile default --mode OIDC` to set it")
	actual.OIDCProviderARN = "oidc_provider_arn"
	err = actual.Validate()
	assert.EqualError(t, err, "oidc_token_file is not configured for profile 'default'. Run `aliyun configure --profile default --mode OIDC` to set it")
	actual.OIDCTokenFile = "/path/to/oidc/token/file"
	err = actual.Validate()
	assert.EqualError(t, err, "ram_role_arn is not configured for profile 'default'. Run `aliyun configure --profile default --mode OIDC` to set it")
	actual.RamRoleArn = "ramrolearn"
	err = actual.Validate()
	assert.EqualError(t, err, "role_session_name is not configured for profile 'default'. Run `aliyun configure --profile default --mode OIDC` to set it")
	actual.RoleSessionName = "rsn"
	err = actual.Validate()
	assert.Nil(t, err)
}

func TestGetParent(t *testing.T) {
	profile := newProfile()
	p := profile.GetParent()
	assert.Nil(t, p)
}

func TestMergeProfileAfterCredentialRefresh_KeepsStoredNonCredentialFields(t *testing.T) {
	disk := Profile{
		Name:     "p",
		Mode:     CloudSSO,
		RegionId: "cn-hangzhou",
		Endpoint: "https://from-file.example.com",
	}
	cp := disk
	cp.Endpoint = "https://from-cli.example.com"
	cp.AccessKeyId = "sts-ak"
	cp.AccessKeySecret = "sts-sk"
	cp.StsToken = "sts-token"
	cp.StsExpiration = 999999
	cp.OAuthAccessToken = "oat"
	cp.OAuthRefreshToken = "ort"
	cp.OAuthAccessTokenExpire = 8888

	got := mergeProfileAfterCredentialRefresh(disk, &cp)
	assert.Equal(t, "https://from-file.example.com", got.Endpoint)
	assert.Equal(t, "sts-ak", got.AccessKeyId)
	assert.Equal(t, "sts-sk", got.AccessKeySecret)
	assert.Equal(t, "sts-token", got.StsToken)
	assert.Equal(t, int64(999999), got.StsExpiration)
	assert.Equal(t, "oat", got.OAuthAccessToken)
	assert.Equal(t, "ort", got.OAuthRefreshToken)
	assert.Equal(t, int64(8888), got.OAuthAccessTokenExpire)
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
	CloudSSOSignInUrlFlag(ctx.Flags()).SetAssigned(true)
	CloudSSOAccountIdFlag(ctx.Flags()).SetValue("111")
	CloudSSOAccountIdFlag(ctx.Flags()).SetAssigned(true)
	CloudSSOAccessConfigFlag(ctx.Flags()).SetValue("222")
	CloudSSOAccessConfigFlag(ctx.Flags()).SetAssigned(true)
	EndpointTypeFlag(ctx.Flags()).SetAssigned(true)
	EndpointTypeFlag(ctx.Flags()).SetValue("vpc")
	EndpointFlag(ctx.Flags()).SetAssigned(true)
	EndpointFlag(ctx.Flags()).SetValue("custom.endpoint.aliyuncs.com")
	ExternalAccountTypeFlag(ctx.Flags()).SetAssigned(true)
	ExternalAccountTypeFlag(ctx.Flags()).SetValue("buc")

	exp := &Profile{
		Name:                 "default",
		Mode:                 AuthenticateMode("ModeFlag"),
		OutputFormat:         "json",
		Language:             "LanguageFlag",
		AccessKeyId:          "AccessKeyIdFlag",
		AccessKeySecret:      "AccessKeySecretFlag",
		StsToken:             "StsTokenFlag",
		RamRoleName:          "RamRoleNameFlag",
		RamRoleArn:           "RamRoleArnFlag",
		RoleSessionName:      "RoleSessionNameFlag",
		PrivateKey:           "PrivateKeyFlag",
		KeyPairName:          "KeyPairNameFlag",
		ExpiredSeconds:       4,
		Verified:             "",
		RegionId:             "RegionFlag",
		Site:                 "",
		ReadTimeout:          1,
		ConnectTimeout:       2,
		RetryCount:           3,
		CloudSSOAccountId:    "111",
		CloudSSOAccessConfig: "222",
		EndpointType:         "vpc",
		Endpoint:             "custom.endpoint.aliyuncs.com",
		ExternalAccountType:  "buc",
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
	os.Setenv("ALIBABA_CLOUD_ENDPOINT_TYPE", "")
	os.Setenv("ALIBABACLOUD_ENDPOINT_TYPE", "")
	os.Setenv("ALICLOUD_ENDPOINT_TYPE", "")
	os.Setenv("ENDPOINT_TYPE", "")
	os.Setenv("ALIBABA_CLOUD_ENDPOINT", "")
	os.Setenv("ALIBABACLOUD_ENDPOINT", "")
	os.Setenv("ALICLOUD_ENDPOINT", "")
	os.Setenv("ENDPOINT", "")
	os.Setenv("ALIBABA_CLOUD_EXTERNAL_ACCOUNT_TYPE", "")
}

func TestOverwriteWithFlagsWithEndpointTypeEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	os.Setenv("ENDPOINT_TYPE", "vpc")
	actual.OverwriteWithFlags(ctx)
	exp.EndpointType = "vpc"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALICLOUD_ENDPOINT_TYPE", "vpc")
	actual.OverwriteWithFlags(ctx)
	exp.EndpointType = "vpc"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABACLOUD_ENDPOINT_TYPE", "vpc")
	actual.OverwriteWithFlags(ctx)
	exp.EndpointType = "vpc"
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABA_CLOUD_ENDPOINT_TYPE", "vpc")
	actual.OverwriteWithFlags(ctx)
	exp.EndpointType = "vpc"
	assert.Equal(t, exp, actual)
}

func TestOverwriteWithFlagsWithEndpointEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ENDPOINT", "endpoint1.aliyuncs.com")
	actual.OverwriteWithFlags(ctx)
	// endpoint env is intentionally narrowed to ALIBABA_CLOUD_ENDPOINT only
	exp.Endpoint = ""
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALICLOUD_ENDPOINT", "endpoint2.aliyuncs.com")
	actual.OverwriteWithFlags(ctx)
	exp.Endpoint = ""
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABACLOUD_ENDPOINT", "endpoint3.aliyuncs.com")
	actual.OverwriteWithFlags(ctx)
	exp.Endpoint = ""
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABA_CLOUD_ENDPOINT", "endpoint4.aliyuncs.com")
	actual.OverwriteWithFlags(ctx)
	exp.Endpoint = "endpoint4.aliyuncs.com"
	assert.Equal(t, exp, actual)

	resetEnv()
}

func TestOverwriteWithFlagsWithExternalAccountTypeEnv(t *testing.T) {
	buf := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	ctx := cli.NewCommandContext(buf, stderr)
	AddFlags(ctx.Flags())

	resetEnv()
	actual := newProfile()
	exp := newProfile()
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, exp, actual)

	actual = newProfile()
	os.Setenv("ALIBABA_CLOUD_EXTERNAL_ACCOUNT_TYPE", "buc")
	actual.OverwriteWithFlags(ctx)
	exp.ExternalAccountType = "buc"
	assert.Equal(t, exp, actual)

	resetEnv()
}

func TestValidateAk(t *testing.T) {
	actual := newProfile()
	err := actual.ValidateAK()
	assert.EqualError(t, err, "access_key_id is not configured for profile 'default'. Run `aliyun configure --profile default --mode ` to set it")
	actual.AccessKeyId = "accessKeyId"
	err = actual.ValidateAK()
	assert.EqualError(t, err, "access_key_secret is not configured for profile 'default'. Run `aliyun configure --profile default --mode ` to set it")
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

func TestNormalizeStsEndpointHost(t *testing.T) {
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", normalizeSTSEndpointHost("  https://sts-vpc.cn-hangzhou.aliyuncs.com/  "))
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", normalizeSTSEndpointHost("http://sts-vpc.cn-hangzhou.aliyuncs.com"))
	assert.Equal(t, "sts.aliyuncs.com", normalizeSTSEndpointHost("sts.aliyuncs.com"))
}

func TestResolveStsEndpoint(t *testing.T) {
	cp := &Profile{StsRegion: "cn-hangzhou"}
	assert.Equal(t, "sts.cn-hangzhou.aliyuncs.com", resolveSTSEndpoint(cp))

	cp = &Profile{StsRegion: "cn-beijing"}
	assert.Equal(t, "sts.cn-beijing.aliyuncs.com", resolveSTSEndpoint(cp))

	cp = &Profile{StsRegion: "cn-hangzhou", StsEndpoint: "https://sts-vpc.cn-hangzhou.aliyuncs.com/"}
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", resolveSTSEndpoint(cp))

	cp = &Profile{StsRegion: "cn-hangzhou", StsEndpoint: "sts-vpc.cn-hangzhou.aliyuncs.com"}
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", resolveSTSEndpoint(cp))
}

func TestOverwriteWithFlagsStsEndpointEnv(t *testing.T) {
	ctx := newCtx()
	actual := &Profile{Name: "default", RegionId: "cn-hangzhou"}

	os.Setenv("ALIBABA_CLOUD_STS_ENDPOINT", "sts-vpc.cn-hangzhou.aliyuncs.com")
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", actual.StsEndpoint)
	os.Unsetenv("ALIBABA_CLOUD_STS_ENDPOINT")

	os.Setenv("ALIBABACLOUD_STS_ENDPOINT", "sts-vpc.cn-shanghai.aliyuncs.com")
	actual.StsEndpoint = ""
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, "", actual.StsEndpoint)
	os.Unsetenv("ALIBABACLOUD_STS_ENDPOINT")
}

func TestOverwriteWithFlagsStsEndpointFlag(t *testing.T) {
	ctx := newCtx()
	StsEndpointFlag(ctx.Flags()).SetAssigned(true)
	StsEndpointFlag(ctx.Flags()).SetValue("sts-vpc.cn-beijing.aliyuncs.com")
	actual := &Profile{Name: "default", StsEndpoint: "sts-vpc.cn-hangzhou.aliyuncs.com"}
	actual.OverwriteWithFlags(ctx)
	assert.Equal(t, "sts-vpc.cn-beijing.aliyuncs.com", actual.StsEndpoint)
}

func credentialSTSEndpoint(t *testing.T, cred credentialsv2.Credential) string {
	t.Helper()
	rv := reflect.ValueOf(cred)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		t.Fatal("credential must be a non-nil pointer")
	}
	rv = rv.Elem()
	providerField := rv.FieldByName("provider")
	if !providerField.IsValid() {
		providerField = rv
	}
	for providerField.Kind() == reflect.Ptr || providerField.Kind() == reflect.Interface {
		if providerField.IsNil() {
			t.Fatal("credential provider is nil")
		}
		providerField = providerField.Elem()
	}
	if stsField := providerField.FieldByName("stsEndpoint"); stsField.IsValid() {
		return stsField.String()
	}
	if runtimeField := providerField.FieldByName("runtime"); runtimeField.IsValid() && !runtimeField.IsNil() {
		if runtimeField.Kind() == reflect.Ptr {
			runtimeField = runtimeField.Elem()
		}
		if endpoint := runtimeField.FieldByName("STSEndpoint"); endpoint.IsValid() {
			return endpoint.String()
		}
	}
	t.Fatal("provider has no sts endpoint field")
	return ""
}

func TestGetCredentialWithOIDCStsEndpoint(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "oidc-token-*")
	assert.NoError(t, err)
	defer os.Remove(tmpFile.Name())
	_, err = tmpFile.WriteString("dummy-oidc-token")
	assert.NoError(t, err)
	tmpFile.Close()

	actual := newProfile()
	actual.Mode = OIDC
	actual.OIDCProviderARN = "acs:ram::123456789:oidc-provider/test"
	actual.OIDCTokenFile = tmpFile.Name()
	actual.RamRoleArn = "acs:ram::123456789:role/test"
	actual.RoleSessionName = "ack-session"
	actual.StsEndpoint = "https://sts-vpc.cn-hangzhou.aliyuncs.com/"

	credential, err := actual.GetCredential(newCtx(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, credential)
	assert.Equal(t, "oidc_role_arn", *credential.GetType())
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", credentialSTSEndpoint(t, credential))
}

func TestGetCredentialWithRamRoleArnStsEndpoint(t *testing.T) {
	actual := newProfile()
	actual.Mode = RamRoleArn
	actual.AccessKeyId = "akid"
	actual.AccessKeySecret = "skid"
	actual.RamRoleArn = "ramRoleArn"
	actual.RoleSessionName = "roleSessionName"
	actual.ExpiredSeconds = 3600
	actual.StsEndpoint = "sts-vpc.cn-hangzhou.aliyuncs.com"

	credential, err := actual.GetCredential(newCtx(), nil)
	assert.NoError(t, err)
	assert.NotNil(t, credential)
	assert.Equal(t, "ram_role_arn", *credential.GetType())
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", credentialSTSEndpoint(t, credential))
}

func TestGetRuntimeEnv_StsEndpoint(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = "ak"
	p.AccessKeySecret = "sk"
	p.RegionId = "cn-hangzhou"
	p.StsEndpoint = "sts-vpc.cn-hangzhou.aliyuncs.com"

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)
	assert.Equal(t, "sts-vpc.cn-hangzhou.aliyuncs.com", envs["ALIBABA_CLOUD_STS_ENDPOINT"])
}

func TestGetRuntimeEnv_StsEndpointOmitted(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = "ak"
	p.AccessKeySecret = "sk"
	p.RegionId = "cn-hangzhou"

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)
	_, has := envs["ALIBABA_CLOUD_STS_ENDPOINT"]
	assert.False(t, has)
}

func TestNormalizeMode(t *testing.T) {
	assert.Equal(t, OAuth, NormalizeMode("oauth"))
	assert.Equal(t, OAuth, NormalizeMode("OAuth"))
	assert.Equal(t, OAuth, NormalizeMode("OAUTH"))
	assert.Equal(t, CloudSSO, NormalizeMode("cloudsso"))
	assert.Equal(t, CloudSSO, NormalizeMode("CloudSSO"))
	assert.Equal(t, AK, NormalizeMode("ak"))
	assert.Equal(t, AK, NormalizeMode("AK"))
	assert.Equal(t, StsToken, NormalizeMode("ststoken"))
	assert.Equal(t, RamRoleArn, NormalizeMode("ramrolearn"))
	assert.Equal(t, EcsRamRole, NormalizeMode("ecsramrole"))
	assert.Equal(t, ChainableRamRoleArn, NormalizeMode("chainableramrolearn"))
	assert.Equal(t, OIDC, NormalizeMode("oidc"))
	assert.Equal(t, External, NormalizeMode("external"))
	assert.Equal(t, CredentialsURI, NormalizeMode("credentialsuri"))
	assert.Equal(t, AuthenticateMode("UnknownMode"), NormalizeMode("UnknownMode"))
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

	p = &Profile{BearerTokenValue: "my-token"}
	AutoModeRecognition(p)
	assert.Equal(t, BearerToken, p.Mode)
}

func TestValidateBearerToken(t *testing.T) {
	p := &Profile{Name: "default", Mode: BearerToken, RegionId: "cn-hangzhou"}
	assert.EqualError(t, p.Validate(), "bearer_token is not configured for profile 'default'. Run `aliyun configure --profile default --mode BearerToken` to set it")

	p.BearerTokenValue = "token"
	assert.Nil(t, p.Validate())

	p.BearerTokenHeaderKey = "x-custom-token"
	assert.Nil(t, p.Validate())
	assert.Equal(t, "x-custom-token", p.BearerTokenHeaderKey)

	p.BearerTokenHeaderKey = "bad\r\nkey"
	assert.Error(t, p.Validate())
}

func TestGetCredentialByBearerToken(t *testing.T) {
	actual := newProfile()
	actual.Mode = BearerToken
	actual.BearerTokenValue = "my-bearer-token"
	actual.RegionId = "cn-hangzhou"

	credential, err := actual.GetCredential(newCtx(), nil)
	assert.Nil(t, err)
	assert.NotNil(t, credential)
	assert.Equal(t, "bearer", *credential.GetType())
	model, err := credential.GetCredential()
	assert.Nil(t, err)
	assert.Equal(t, "my-bearer-token", *model.BearerToken)
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

func TestGetProfileWithCloudSSO(t *testing.T) {
	cf := NewConfiguration()
	p := newProfile()
	p.Mode = CloudSSO
	p.CloudSSOAccountId = "111"
	p.CloudSSOAccessConfig = "222"
	p.CloudSSOSignInUrl = "333"
	cf.PutProfile(*p)

	saveConfigurationFunc = func(config *Configuration) (err error) {
		// 模拟保存配置成功
		return nil
	}

	c, err := p.GetCredential(newCtx(), nil)
	assert.Nil(t, c)
	// err != nil and contain CloudSSO access token is expired
	assert.EqualError(t, err, "CloudSSO access token is expired, please re-login with command: aliyun configure --profile default")
}

func TestGetCredentialWithCloudSSOMockSuccess(t *testing.T) {
	// 保存原始函数并在测试后恢复
	originalFunc := tryRefreshStsTokenFunc
	defer func() { tryRefreshStsTokenFunc = originalFunc }()

	// 创建mock函数
	tryRefreshStsTokenFunc = func(signInUrl, accessToken, accessConfig, accountId *string, client *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		// 返回模拟数据
		return &cloudsso.CloudCredentialResponse{
			AccessKeyId:     "mock-ak-id",
			AccessKeySecret: "mock-ak-secret",
			SecurityToken:   "mock-security-token",
			ExpirationInt64: time.Now().Unix() + 3600,
		}, nil
	}

	saveConfigurationFunc = func(config *Configuration) (err error) {
		// 模拟保存配置成功
		return nil
	}

	// 准备测试数据
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "cloudsso-profile"
	p.Mode = CloudSSO
	p.RegionId = "cn-hangzhou"
	p.CloudSSOAccountId = "test-account-id"
	p.CloudSSOAccessConfig = "test-access-config"
	p.CloudSSOSignInUrl = "https://cloudsso.example.com"
	p.AccessToken = "test-access-token"
	p.CloudSSOAccessTokenExpire = time.Now().Unix() + 7200 // 确保令牌未过期
	cf.PutProfile(*p)
	p.parent = cf

	// hook loadConfiguration
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{CurrentProfile: "default", Profiles: []Profile{
				{Name: "default", Mode: AK, AccessKeyId: "default_aliyun_access_key_id", AccessKeySecret: "default_aliyun_access_key_secret", OutputFormat: "json"},
				{Name: "bbb", Mode: AK, AccessKeyId: "sdf", AccessKeySecret: "ddf", OutputFormat: "json"}}}, nil
		}
	}

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.Nil(t, err)
	assert.NotNil(t, cred)
	assert.Equal(t, "sts", *cred.GetType())
	assert.Equal(t, "mock-ak-id", p.AccessKeyId)
	assert.Equal(t, "mock-ak-secret", p.AccessKeySecret)
	assert.Equal(t, "mock-security-token", p.StsToken)
}

func TestGetCredentialWithCloudSSOMockError(t *testing.T) {
	// 保存原始函数并在测试后恢复
	originalFunc := tryRefreshStsTokenFunc
	defer func() { tryRefreshStsTokenFunc = originalFunc }()

	// 创建mock函数模拟错误情况
	tryRefreshStsTokenFunc = func(signInUrl, accessToken, accessConfig, accountId *string, client *http.Client) (*cloudsso.CloudCredentialResponse, error) {
		return nil, fmt.Errorf("mock cloudsso error")
	}

	// 准备测试数据
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "cloudsso-profile"
	p.Mode = CloudSSO
	p.RegionId = "cn-hangzhou"
	p.CloudSSOAccountId = "test-account-id"
	p.CloudSSOAccessConfig = "test-access-config"
	p.CloudSSOSignInUrl = "https://cloudsso.example.com"
	p.AccessToken = "test-access-token"
	p.CloudSSOAccessTokenExpire = time.Now().Unix() + 7200 // 确保令牌未过期
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.NotNil(t, err)
	assert.Nil(t, cred)
	assert.EqualError(t, err, "mock cloudsso error")
}

// when mode is CloudSSO, but CloudSSOSignInUrl is empty, should return error
func TestGetCredentialWithCloudSSOEmptySignInUrl(t *testing.T) {
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "cloudsso-profile"
	p.Mode = CloudSSO
	p.RegionId = "cn-hangzhou"
	p.CloudSSOAccountId = "test-account-id"
	p.CloudSSOAccessConfig = "test-access-config"
	p.CloudSSOSignInUrl = ""
	p.AccessToken = "test-access-token"
	p.CloudSSOAccessTokenExpire = time.Now().Unix() + 7200 // 确保令牌未过期
	cf.PutProfile(*p)
	p.parent = cf

	cred, err := p.GetCredential(newCtx(), nil)

	assert.NotNil(t, err)
	assert.Nil(t, cred)
	assert.Contains(t, err.Error(), "aliyun configure --profile cloudsso-profile --mode CloudSSO")
}

// GetCredential not support mode test
func TestGetCredentialWithMode(t *testing.T) {
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "test-profile"
	p.Mode = AuthenticateMode("NoMode")
	cf.PutProfile(*p)
	p.parent = cf

	cred, err := p.GetCredential(newCtx(), nil)

	assert.NotNil(t, err)
	assert.Nil(t, cred)
	assert.EqualError(t, err, "unexcepted certificate mode: NoMode")
}

// test RamRoleArn
func TestGetCredentialWithRamRoleArn(t *testing.T) {
	actual := newProfile()

	actual.Mode = RamRoleArn
	actual.AccessKeyId = "akid"
	actual.AccessKeySecret = "skid"
	actual.RamRoleArn = "ramRoleArn"
	actual.RoleSessionName = "roleSessionName"
	actual.ExpiredSeconds = 3600
	actual.StsRegion = "cn-hangzhou"
	credential, err := actual.GetCredential(newCtx(), nil)
	assert.NotNil(t, credential)
	assert.Nil(t, err)
	assert.Equal(t, "ram_role_arn", *credential.GetType())
}

// TestGetCredentialWithCredentialsURI tests credential retrieval using CredentialsURI mode
func TestGetCredentialWithCredentialsURI(t *testing.T) {
	// 创建测试HTTP服务器 - 成功情况
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Code": "Success",
			"AccessKeyId": "mock-access-key-id",
			"AccessKeySecret": "mock-access-key-secret", 
			"SecurityToken": "mock-security-token",
			"Expiration": "2023-01-01T00:00:00Z"
		}`))
	}))
	defer successServer.Close()

	// 创建测试HTTP服务器 - 错误状态码
	errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer errorServer.Close()

	// 创建测试HTTP服务器 - 无效JSON
	invalidJsonServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer invalidJsonServer.Close()

	// 创建测试HTTP服务器 - 非Success响应
	notSuccessServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Code": "Failed",
			"AccessKeyId": "mock-access-key-id",
			"AccessKeySecret": "mock-access-key-secret",
			"SecurityToken": "mock-security-token",
			"Expiration": "2023-01-01T00:00:00Z"
		}`))
	}))
	defer notSuccessServer.Close()

	// 测试成功情况
	actual := newProfile()
	actual.Mode = CredentialsURI
	actual.CredentialsURI = successServer.URL
	actual.RegionId = "cn-hangzhou"

	credential, err := actual.GetCredential(newCtx(), nil)
	assert.NotNil(t, credential)
	assert.Nil(t, err)
	assert.Equal(t, "sts", *credential.GetType())

	// 测试URI为空的情况
	actual.CredentialsURI = ""
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "invalid credentials uri")

	// 测试服务器返回错误状态码
	actual.CredentialsURI = errorServer.URL
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "status code 500")

	// 测试服务器返回无效JSON
	actual.CredentialsURI = invalidJsonServer.URL
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "unmarshal")

	// 测试服务器返回非Success响应
	actual.CredentialsURI = notSuccessServer.URL
	credential, err = actual.GetCredential(newCtx(), nil)
	assert.Nil(t, credential)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "Code is not Success")
}

func TestProfile_GetCredential_External(t *testing.T) {
	// if is windows, ignore
	s := runtime.GOOS
	if s == "windows" {
		t.Skip("Skip external test on Windows")
	}
	// 创建临时目录用于存放测试脚本
	tempDir, err := ioutil.TempDir("", "external_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 1. 成功场景：正确的JSON输出
	successScript := filepath.Join(tempDir, "success.sh")
	err = ioutil.WriteFile(successScript, []byte(`#!/bin/bash
echo '{"Mode":"AK", "access_key_id":"test-ak", "access_key_secret":"test-secret", "region_id": "cn-hangzhou"}'
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create success script: %v", err)
	}

	// 2. 错误场景：命令执行失败
	failScript := filepath.Join(tempDir, "fail.sh")
	err = ioutil.WriteFile(failScript, []byte(`#!/bin/bash
echo "Error message" >&2
exit 1
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create fail script: %v", err)
	}

	// 3. 使用stdin的场景
	stdinScript := filepath.Join(tempDir, "stdin.sh")
	err = ioutil.WriteFile(stdinScript, []byte(`#!/bin/bash
read input
if [ "$input" = "test-input" ]; then
  echo '{"Mode":"AK", "access_key_id":"stdin-ak", "access_key_secret":"stdin-secret", "region_id": "cn-hangzhou"}'
else
  echo "Invalid input" >&2
  exit 1
fi
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create stdin script: %v", err)
	}

	// 4. 混合stdout和stderr的场景
	mixedScript := filepath.Join(tempDir, "mixed.sh")
	err = ioutil.WriteFile(mixedScript, []byte(`#!/bin/bash
echo "Warning: using default values" >&2
echo '{"Mode":"AK", "access_key_id":"mixed-ak", "access_key_secret":"mixed-secret", "region_id": "cn-hangzhou"}'
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create mixed script: %v", err)
	}

	// 5. 无效JSON输出的场景
	invalidJsonScript := filepath.Join(tempDir, "invalid.sh")
	err = ioutil.WriteFile(invalidJsonScript, []byte(`#!/bin/bash
echo 'This is not a valid JSON'
`), 0755)
	if err != nil {
		t.Fatalf("Failed to create invalid JSON script: %v", err)
	}

	tests := []struct {
		name           string
		profile        Profile
		expectError    bool
		expectAkId     string
		expectAkSecret string
		setupStdin     func() (*os.File, *os.File, error)
		cleanupStdin   func(*os.File, *os.File)
	}{
		{
			name: "success",
			profile: Profile{
				Name:           "success",
				Mode:           External,
				ProcessCommand: successScript,
			},
			expectError:    false,
			expectAkId:     "test-ak",
			expectAkSecret: "test-secret",
		},
		{
			name: "command_failure",
			profile: Profile{
				Name:           "fail",
				Mode:           External,
				ProcessCommand: failScript,
			},
			expectError: true,
		},
		{
			name: "with_stdin",
			profile: Profile{
				Name:           "stdin",
				Mode:           External,
				ProcessCommand: stdinScript,
			},
			expectError:    false,
			expectAkId:     "stdin-ak",
			expectAkSecret: "stdin-secret",
			setupStdin: func() (*os.File, *os.File, error) {
				// 保存原始stdin
				oldStdin := os.Stdin
				// 创建管道
				r, w, err := os.Pipe()
				if err != nil {
					return nil, nil, err
				}
				// 将管道的读取端设置为stdin
				os.Stdin = r
				// 写入测试输入
				_, err = w.WriteString("test-input\n")
				if err != nil {
					return nil, nil, err
				}
				return oldStdin, w, nil
			},
			cleanupStdin: func(oldStdin *os.File, w *os.File) {
				// 关闭管道写入端
				w.Close()
				// 恢复原始stdin
				os.Stdin = oldStdin
			},
		},
		{
			name: "mixed_output",
			profile: Profile{
				Name:           "mixed",
				Mode:           External,
				ProcessCommand: mixedScript,
			},
			expectError:    false,
			expectAkId:     "mixed-ak",
			expectAkSecret: "mixed-secret",
		},
		{
			name: "invalid_json",
			profile: Profile{
				Name:           "invalid",
				Mode:           External,
				ProcessCommand: invalidJsonScript,
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 设置测试的stdin(如果需要)
			if tt.setupStdin != nil {
				oldStdin, w, err := tt.setupStdin()
				if err != nil {
					t.Fatalf("Failed to setup stdin: %v", err)
				}
				defer tt.cleanupStdin(oldStdin, w)
			}

			// 创建配置并设置当前测试的profile
			config := &Configuration{
				CurrentProfile: tt.profile.Name,
				Profiles:       []Profile{tt.profile},
			}
			tt.profile.parent = config

			// 调用GetCredential方法
			buf := new(bytes.Buffer)
			buf2 := new(bytes.Buffer)
			ctx := cli.NewCommandContext(buf, buf2)
			cred, err := tt.profile.GetCredential(ctx, nil)

			// 检查结果
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
				}

				// 验证凭证
				credential, err := cred.GetCredential()
				if err != nil {
					t.Errorf("Failed to get credential: %v", err)
					return
				}

				if *credential.AccessKeyId != tt.expectAkId {
					t.Errorf("AccessKeyId mismatch, expected %s but got %s", tt.expectAkId, *credential.AccessKeyId)
				}

				if *credential.AccessKeySecret != tt.expectAkSecret {
					t.Errorf("AccessKeySecret mismatch, expected %s but got %s", tt.expectAkSecret, *credential.AccessKeySecret)
				}
			}
		})
	}
}

func TestProfile_GetCredential_ExternalProcessDisabled(t *testing.T) {
	orig := os.Getenv(EnvDisableExternalProcess)
	os.Setenv(EnvDisableExternalProcess, "1")
	defer func() {
		if orig == "" {
			os.Unsetenv(EnvDisableExternalProcess)
		} else {
			os.Setenv(EnvDisableExternalProcess, orig)
		}
	}()

	profile := Profile{
		Name:           "external",
		Mode:           External,
		ProcessCommand: "env",
	}
	config := &Configuration{
		CurrentProfile: profile.Name,
		Profiles:       []Profile{profile},
	}
	profile.parent = config

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	_, err := profile.GetCredential(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), EnvDisableExternalProcess)
}

func TestProfile_GetCredential_CredentialsURIDisabled(t *testing.T) {
	successServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Code": "Success",
			"AccessKeyId": "mock-access-key-id",
			"AccessKeySecret": "mock-access-key-secret",
			"SecurityToken": "mock-security-token",
			"Expiration": "2023-01-01T00:00:00Z"
		}`))
	}))
	defer successServer.Close()

	orig := os.Getenv(EnvDisableExternalProcess)
	os.Setenv(EnvDisableExternalProcess, "true")
	defer func() {
		if orig == "" {
			os.Unsetenv(EnvDisableExternalProcess)
		} else {
			os.Setenv(EnvDisableExternalProcess, orig)
		}
	}()

	profile := Profile{
		Name:           "uri",
		Mode:           CredentialsURI,
		CredentialsURI: successServer.URL,
	}
	config := &Configuration{
		CurrentProfile: profile.Name,
		Profiles:       []Profile{profile},
	}
	profile.parent = config

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	_, err := profile.GetCredential(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), EnvDisableExternalProcess)
}

// TestGetCredentialWithOAuthStsExpired 测试OAuth模式中STS过期时的刷新逻辑
func TestGetCredentialWithOAuthStsExpired(t *testing.T) {
	// 保存原始函数并在测试后恢复
	originalSaveConfigurationFunc := saveConfigurationFunc
	originalHookLoadConfiguration := hookLoadOrCreateConfiguration
	originalExchangeFromOAuthFunc := exchangeFromOAuthFunc
	defer func() {
		saveConfigurationFunc = originalSaveConfigurationFunc
		hookLoadOrCreateConfiguration = originalHookLoadConfiguration
		exchangeFromOAuthFunc = originalExchangeFromOAuthFunc
	}()

	// Mock保存配置函数
	saveConfigurationFunc = func(config *Configuration) error {
		return nil
	}

	// Mock hookLoadConfiguration函数
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "oauth-test",
				Profiles: []Profile{
					{
						Name:                   "oauth-test",
						Mode:                   OAuth,
						RegionId:               "cn-hangzhou",
						OAuthSiteType:          "CN",
						OAuthAccessToken:       "mock-access-token",
						OAuthRefreshToken:      "mock-refresh-token",
						OAuthAccessTokenExpire: time.Now().Unix() + 3600,
						AccessKeyId:            "mock-ak-id",
						AccessKeySecret:        "mock-ak-secret",
						StsToken:               "mock-sts-token",
						StsExpiration:          time.Now().Unix() + 1800,
					},
				},
			}, nil
		}
	}

	// Mock exchangeFromOAuth函数
	exchangeFromOAuthFunc = func(w io.Writer, cp *Profile) error {
		// 模拟OAuth刷新成功，更新STS凭证
		cp.AccessKeyId = "refreshed-ak-id"
		cp.AccessKeySecret = "refreshed-ak-secret"
		cp.StsToken = "refreshed-sts-token"
		cp.StsExpiration = time.Now().Unix() + 3600
		return nil
	}

	// 创建测试Profile，设置STS已过期
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "oauth-test"
	p.Mode = OAuth
	p.RegionId = "cn-hangzhou"
	p.OAuthSiteType = "CN"
	p.OAuthAccessToken = "mock-access-token"
	p.OAuthRefreshToken = "mock-refresh-token"
	p.OAuthAccessTokenExpire = time.Now().Unix() + 3600 // Access token未过期
	p.AccessKeyId = "old-ak-id"
	p.AccessKeySecret = "old-ak-secret"
	p.StsToken = "old-sts-token"
	p.StsExpiration = time.Now().Unix() - 300 // STS已过期（5分钟前）
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.Nil(t, err)
	assert.NotNil(t, cred)
	assert.Equal(t, "sts", *cred.GetType())

	// 验证STS凭证已被刷新
	assert.Equal(t, "refreshed-ak-id", p.AccessKeyId)
	assert.Equal(t, "refreshed-ak-secret", p.AccessKeySecret)
	assert.Equal(t, "refreshed-sts-token", p.StsToken)
	assert.True(t, p.StsExpiration > time.Now().Unix()) // 确保过期时间已更新
}

// TestGetCredentialWithOAuthStsNotExpired 测试OAuth模式中STS未过期时直接使用现有凭证
func TestGetCredentialWithOAuthStsNotExpired(t *testing.T) {
	// 创建测试Profile，设置STS未过期
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "oauth-test"
	p.Mode = OAuth
	p.RegionId = "cn-hangzhou"
	p.OAuthSiteType = "CN"
	p.OAuthAccessToken = "mock-access-token"
	p.OAuthRefreshToken = "mock-refresh-token"
	p.OAuthAccessTokenExpire = time.Now().Unix() + 3600
	p.AccessKeyId = "valid-ak-id"
	p.AccessKeySecret = "valid-ak-secret"
	p.StsToken = "valid-sts-token"
	p.StsExpiration = time.Now().Unix() + 1800 // STS未过期（30分钟后）
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.Nil(t, err)
	assert.NotNil(t, cred)
	assert.Equal(t, "sts", *cred.GetType())

	// 验证使用的是原有凭证（未被刷新）
	assert.Equal(t, "valid-ak-id", p.AccessKeyId)
	assert.Equal(t, "valid-ak-secret", p.AccessKeySecret)
	assert.Equal(t, "valid-sts-token", p.StsToken)
}

// TestGetCredentialWithOAuthRefreshError 测试OAuth刷新失败的情况
func TestGetCredentialWithOAuthRefreshError(t *testing.T) {
	// 保存原始函数并在测试后恢复
	originalSaveConfigurationFunc := saveConfigurationFunc
	originalHookLoadConfiguration := hookLoadOrCreateConfiguration
	originalExchangeFromOAuthFunc := exchangeFromOAuthFunc
	defer func() {
		saveConfigurationFunc = originalSaveConfigurationFunc
		hookLoadOrCreateConfiguration = originalHookLoadConfiguration
		exchangeFromOAuthFunc = originalExchangeFromOAuthFunc
	}()

	// Mock保存配置函数
	saveConfigurationFunc = func(config *Configuration) error {
		return nil
	}

	// Mock hookLoadConfiguration函数
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "oauth-test",
				Profiles: []Profile{
					{
						Name:          "oauth-test",
						Mode:          OAuth,
						RegionId:      "cn-hangzhou",
						OAuthSiteType: "CN",
					},
				},
			}, nil
		}
	}

	// Mock exchangeFromOAuth函数返回错误
	exchangeFromOAuthFunc = func(w io.Writer, cp *Profile) error {
		return fmt.Errorf("mock refresh error: invalid token")
	}

	// 创建测试Profile，设置STS已过期
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "oauth-test"
	p.Mode = OAuth
	p.RegionId = "cn-hangzhou"
	p.OAuthSiteType = "CN"
	p.OAuthAccessToken = "mock-access-token"
	p.OAuthRefreshToken = "mock-refresh-token"
	p.OAuthAccessTokenExpire = time.Now().Unix() + 3600
	p.AccessKeyId = "old-ak-id"
	p.AccessKeySecret = "old-ak-secret"
	p.StsToken = "old-sts-token"
	p.StsExpiration = time.Now().Unix() - 300 // STS已过期
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.NotNil(t, err)
	assert.Nil(t, cred)
	assert.Contains(t, err.Error(), "mock refresh error: invalid token")
}

// TestGetCredentialWithOAuthInvalidSiteType 测试OAuth模式中无效的站点类型
func TestGetCredentialWithOAuthInvalidSiteType(t *testing.T) {
	// 创建测试Profile，设置无效的OAuthSiteType
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "oauth-test"
	p.Mode = OAuth
	p.RegionId = "cn-hangzhou"
	p.OAuthSiteType = "INVALID" // 无效的站点类型
	p.OAuthAccessToken = "mock-access-token"
	p.OAuthRefreshToken = "mock-refresh-token"
	p.OAuthAccessTokenExpire = time.Now().Unix() + 3600
	p.AccessKeyId = "ak-id"
	p.AccessKeySecret = "ak-secret"
	p.StsToken = "sts-token"
	p.StsExpiration = time.Now().Unix() - 300 // STS已过��，触发刷新
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果 - 应该在验证阶段就失败
	assert.NotNil(t, err)
	assert.Nil(t, cred)
	assert.Contains(t, err.Error(), "invalid OAuth site type")
}

// TestGetCredentialWithOAuthMissingCredentials 测试OAuth模式中STS凭证为空的情况
func TestGetCredentialWithOAuthMissingCredentials(t *testing.T) {
	// 保存原始函数并在测试后恢复
	originalSaveConfigurationFunc := saveConfigurationFunc
	originalHookLoadConfiguration := hookLoadOrCreateConfiguration
	originalExchangeFromOAuthFunc := exchangeFromOAuthFunc
	defer func() {
		saveConfigurationFunc = originalSaveConfigurationFunc
		hookLoadOrCreateConfiguration = originalHookLoadConfiguration
		exchangeFromOAuthFunc = originalExchangeFromOAuthFunc
	}()

	// Mock保存配置函数
	saveConfigurationFunc = func(config *Configuration) error {
		return nil
	}

	// Mock hookLoadConfiguration函数
	hookLoadOrCreateConfiguration = func(fn func(path string) (*Configuration, error)) func(path string) (*Configuration, error) {
		return func(path string) (*Configuration, error) {
			return &Configuration{
				CurrentProfile: "oauth-test",
				Profiles: []Profile{
					{
						Name:          "oauth-test",
						Mode:          OAuth,
						RegionId:      "cn-hangzhou",
						OAuthSiteType: "CN",
					},
				},
			}, nil
		}
	}

	// Mock exchangeFromOAuth函数
	exchangeFromOAuthFunc = func(w io.Writer, cp *Profile) error {
		// 模拟OAuth刷新成功，更新STS凭证
		cp.AccessKeyId = "new-ak-id"
		cp.AccessKeySecret = "new-ak-secret"
		cp.StsToken = "new-sts-token"
		cp.StsExpiration = time.Now().Unix() + 3600
		return nil
	}

	// 创建测试Profile，设置STS凭证为空
	cf := NewConfiguration()
	p := newProfile()
	p.Name = "oauth-test"
	p.Mode = OAuth
	p.RegionId = "cn-hangzhou"
	p.OAuthSiteType = "CN"
	p.OAuthAccessToken = "mock-access-token"
	p.OAuthRefreshToken = "mock-refresh-token"
	p.OAuthAccessTokenExpire = time.Now().Unix() + 3600
	p.AccessKeyId = ""     // 空的AK
	p.AccessKeySecret = "" // 空的Secret
	p.StsToken = ""        // 空的STS Token
	p.StsExpiration = 0    // 过期时间为0
	cf.PutProfile(*p)
	p.parent = cf

	// 执行测试
	cred, err := p.GetCredential(newCtx(), nil)

	// 验证结果
	assert.Nil(t, err)
	assert.NotNil(t, cred)
	assert.Equal(t, "sts", *cred.GetType())

	// 验证STS凭证已被刷新
	assert.Equal(t, "new-ak-id", p.AccessKeyId)
	assert.Equal(t, "new-ak-secret", p.AccessKeySecret)
	assert.Equal(t, "new-sts-token", p.StsToken)
	assert.True(t, p.StsExpiration > time.Now().Unix())
}

func TestGetRuntimeEnv_AK(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = "test-ak-id"
	p.AccessKeySecret = "test-ak-secret"
	p.RegionId = "cn-hangzhou"

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)
	assert.NotNil(t, envs)

	assert.Equal(t, "test-ak-id", envs["ALIBABA_CLOUD_ACCESS_KEY_ID"])
	assert.Equal(t, "test-ak-secret", envs["ALIBABA_CLOUD_ACCESS_KEY_SECRET"])
	assert.Equal(t, "cn-hangzhou", envs["ALIBABA_CLOUD_REGION_ID"])
}

func TestGetRuntimeEnv_StsToken(t *testing.T) {
	p := newProfile()
	p.Mode = StsToken
	p.AccessKeyId = "test-ak-id"
	p.AccessKeySecret = "test-ak-secret"
	p.StsToken = "test-sts-token"
	p.RegionId = "cn-hangzhou"

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)
	assert.Equal(t, "test-sts-token", envs["ALIBABA_CLOUD_SECURITY_TOKEN"])
}

func TestGetRuntimeEnv_BearerToken(t *testing.T) {
	p := &Profile{
		Name:                 "default",
		Mode:                 BearerToken,
		BearerTokenValue:     "my-bearer-token",
		BearerTokenHeaderKey: "x-custom-token",
		RegionId:             "cn-hangzhou",
	}

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)
	assert.Equal(t, "my-bearer-token", envs["ALIBABA_CLOUD_BEARER_TOKEN"])
	assert.Equal(t, "x-custom-token", envs["ALIBABA_CLOUD_BEARER_TOKEN_HEADER_KEY"])
	assert.Equal(t, "cn-hangzhou", envs["ALIBABA_CLOUD_REGION_ID"])
	_, hasAK := envs["ALIBABA_CLOUD_ACCESS_KEY_ID"]
	assert.False(t, hasAK)

	p.BearerTokenValue = ""
	envs, err = p.GetRuntimeEnv(newCtx())
	assert.Error(t, err)
	assert.Nil(t, envs)
}

func TestGetRuntimeEnv_OptionalFields(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = "ak"
	p.AccessKeySecret = "sk"
	p.RegionId = "cn-hangzhou"
	p.Language = "zh"
	p.EndpointType = "vpc"
	p.Endpoint = "custom.endpoint.aliyuncs.com"
	p.ExternalAccountType = "buc"
	p.ReadTimeout = 30
	p.ConnectTimeout = 10
	p.RetryCount = 3

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)

	assert.Equal(t, "zh", envs["ALIBABA_CLOUD_LANGUAGE"])
	assert.Equal(t, "vpc", envs["ALIBABA_CLOUD_ENDPOINT_TYPE"])
	assert.Equal(t, "custom.endpoint.aliyuncs.com", envs["ALIBABA_CLOUD_ENDPOINT"])
	assert.Equal(t, "buc", envs["ALIBABA_CLOUD_EXTERNAL_ACCOUNT_TYPE"])
	assert.Equal(t, "30", envs["ALIBABA_CLOUD_READ_TIMEOUT"])
	assert.Equal(t, "10", envs["ALIBABA_CLOUD_CONNECT_TIMEOUT"])
	assert.Equal(t, "3", envs["ALIBABA_CLOUD_RETRY_COUNT"])
}

func TestGetRuntimeEnv_OptionalFieldsOmitted(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = "ak"
	p.AccessKeySecret = "sk"
	p.RegionId = "cn-hangzhou"
	p.Language = ""
	p.EndpointType = ""
	p.Endpoint = ""
	p.ExternalAccountType = ""
	p.ReadTimeout = 0
	p.ConnectTimeout = 0
	p.RetryCount = 0

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.NoError(t, err)

	_, hasLang := envs["ALIBABA_CLOUD_LANGUAGE"]
	assert.False(t, hasLang)
	_, hasEndpoint := envs["ALIBABA_CLOUD_ENDPOINT_TYPE"]
	assert.False(t, hasEndpoint)
	_, hasEndpointVal := envs["ALIBABA_CLOUD_ENDPOINT"]
	assert.False(t, hasEndpointVal)
	_, hasExternalAcct := envs["ALIBABA_CLOUD_EXTERNAL_ACCOUNT_TYPE"]
	assert.False(t, hasExternalAcct)
	_, hasReadTimeout := envs["ALIBABA_CLOUD_READ_TIMEOUT"]
	assert.False(t, hasReadTimeout)
	_, hasConnTimeout := envs["ALIBABA_CLOUD_CONNECT_TIMEOUT"]
	assert.False(t, hasConnTimeout)
	_, hasRetry := envs["ALIBABA_CLOUD_RETRY_COUNT"]
	assert.False(t, hasRetry)
}

func TestGetRuntimeEnv_CredentialError(t *testing.T) {
	p := newProfile()
	p.Mode = AK
	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.RegionId = ""

	envs, err := p.GetRuntimeEnv(newCtx())
	assert.Error(t, err)
	assert.Nil(t, envs)
}

func TestOpenAPIAuthType(t *testing.T) {
	assert.Equal(t, "AK", (&Profile{Mode: AK}).OpenAPIAuthType())
	assert.Equal(t, "bearer", (&Profile{Mode: BearerToken}).OpenAPIAuthType())
	assert.Equal(t, "Anonymous", (&Profile{
		Mode:                 BearerToken,
		BearerTokenHeaderKey: "x-custom-token",
	}).OpenAPIAuthType())
	assert.Equal(t, "Anonymous", (&Profile{Mode: Anonymous}).OpenAPIAuthType())
}

func TestAnonymousFromFlag(t *testing.T) {
	ctx := newCtx()
	ModeFlag(ctx.Flags()).SetAssigned(true)
	ModeFlag(ctx.Flags()).SetValue("Anonymous")
	p := &Profile{Name: "default", RegionId: "cn-hangzhou"}
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Anonymous, p.Mode)
	assert.NoError(t, p.Validate())
	cred, err := p.GetCredential(ctx, nil)
	assert.NoError(t, err)
	assert.Nil(t, cred)
}

func TestAnonymousFromEnv(t *testing.T) {
	t.Setenv("ALIBABA_CLOUD_PROFILE_MODE", "anonymous")
	ctx := newCtx()
	p := &Profile{Name: "default", RegionId: "cn-hangzhou"}
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Anonymous, p.Mode)
	cred, err := p.GetCredential(ctx, nil)
	assert.NoError(t, err)
	assert.Nil(t, cred)
	envs, err := p.GetRuntimeEnv(ctx)
	assert.NoError(t, err)
	assert.Equal(t, "Anonymous", envs["ALIBABA_CLOUD_PROFILE_MODE"])
}

func TestAnonymousValidateNoRegion(t *testing.T) {
	// P-04: Validate fails when RegionId is empty
	p := &Profile{Name: "default", Mode: Anonymous, RegionId: ""}
	err := p.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "region")
}

func TestNormalizeModeAnonymous(t *testing.T) {
	// P-08: NormalizeMode recognizes Anonymous (case-insensitive)
	assert.Equal(t, Anonymous, NormalizeMode("anonymous"))
	assert.Equal(t, Anonymous, NormalizeMode("ANONYMOUS"))
	assert.Equal(t, Anonymous, NormalizeMode("Anonymous"))
}

func TestModeAnonymousOverridesDefault(t *testing.T) {
	// P-10: --mode Anonymous overrides default AK mode
	ctx := newCtx()
	ModeFlag(ctx.Flags()).SetAssigned(true)
	ModeFlag(ctx.Flags()).SetValue("Anonymous")
	p := &Profile{
		Name:            "default",
		RegionId:        "cn-hangzhou",
		AccessKeyId:     "test-ak",
		AccessKeySecret: "test-sk",
	}
	p.OverwriteWithFlags(ctx)
	// --mode Anonymous sets mode directly
	assert.Equal(t, Anonymous, p.Mode)
}

func TestModeFlagRegistered(t *testing.T) {
	// F-01: mode flag is registered and supports Anonymous
	ctx := newCtx()
	flag := ModeFlag(ctx.Flags())
	assert.NotNil(t, flag)
	assert.Equal(t, ModeFlagName, flag.Name)
}

func TestModeFlagValue(t *testing.T) {
	// F-02: flag assignment
	ctx := newCtx()
	ModeFlag(ctx.Flags()).SetAssigned(true)
	ModeFlag(ctx.Flags()).SetValue("Anonymous")
	v, ok := ModeFlag(ctx.Flags()).GetValue()
	assert.True(t, ok)
	assert.Equal(t, "Anonymous", v)
}

func TestInjectBearerTokenHeader(t *testing.T) {
	headers := map[string]*string{}
	(&Profile{Mode: AK, BearerTokenValue: "t"}).InjectBearerTokenHeader(headers)
	assert.Empty(t, headers)

	(&Profile{
		Mode:                 BearerToken,
		BearerTokenValue:     "secret",
		BearerTokenHeaderKey: "x-custom-token",
	}).InjectBearerTokenHeader(headers)
	assert.Equal(t, "secret", *headers["x-custom-token"])
}

func TestNormalizeBearerTokenHeaderKey(t *testing.T) {
	key, err := NormalizeBearerTokenHeaderKey("  x-custom-token  ")
	assert.Nil(t, err)
	assert.Equal(t, "x-custom-token", key)

	_, err = NormalizeBearerTokenHeaderKey("x-bad\r\nInjected: 1")
	assert.Error(t, err)

	_, err = NormalizeBearerTokenHeaderKey("x bad")
	assert.Error(t, err)

	key, err = NormalizeBearerTokenHeaderKey("")
	assert.Nil(t, err)
	assert.Equal(t, "", key)
}

func TestSanitizeBearerTokenValue(t *testing.T) {
	assert.Equal(t, "abcX-Injected: yes", SanitizeBearerTokenValue("abc\r\nX-Injected: yes"))
	assert.Equal(t, "token", SanitizeBearerTokenValue("token"))
}

func TestErrBearerTokenRequiresPlugin(t *testing.T) {
	err := ErrBearerTokenRequiresPlugin("devops")
	assert.Contains(t, err.Error(), "devops")
	assert.Contains(t, err.Error(), "aliyun devops")

	err = ErrBearerTokenRequiresPlugin("")
	assert.Contains(t, err.Error(), "product plugin")
}
