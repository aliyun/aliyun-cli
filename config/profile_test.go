// Copyright 1999-2019 Alibaba Group Holding Limited
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
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/services/sts"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/signers"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"

	"github.com/stretchr/testify/assert"
)

func TestNewProfile(t *testing.T) {
	exp := Profile{
		Name:            "default",
		Mode:            AK,
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
		RetryTimeout:    0,
		RetryCount:      0,
	}

	p := NewProfile("default")
	assert.Equal(t, exp, p)
}

func TestProfile(t *testing.T) {
	ACCESS_KEY_ID_env := os.Getenv("ACCESS_KEY_ID")
	if ACCESS_KEY_ID_env != "" {
		os.Setenv("ACCESS_KEY_ID", "")
		defer func() {
			os.Setenv("ACCESS_KEY_ID", ACCESS_KEY_ID_env)
		}()
	}
	ACCESS_KEY_SECRET_env := os.Getenv("ACCESS_KEY_SECRET")
	if ACCESS_KEY_ID_env != "" {
		os.Setenv("ACCESS_KEY_SECRET", "")
		defer func() {
			os.Setenv("ACCESS_KEY_SECRET", ACCESS_KEY_SECRET_env)
		}()
	}

	//ValidateAK
	p := NewProfile("default")
	assert.EqualError(t, p.ValidateAK(), "invalid access_key_id: ")
	p.AccessKeyId = "*****"
	assert.EqualError(t, p.ValidateAK(), "invaild access_key_secret: ")
	p.AccessKeySecret = "++++++"
	assert.Nil(t, p.ValidateAK())

	//Validate
	assert.EqualError(t, p.Validate(), "region can't be empty")
	p.RegionId = "*dflsj"
	assert.EqualError(t, p.Validate(), "invailed region *dflsj")
	p.RegionId = "cn-hangzhou"

	p.Mode = ""
	assert.EqualError(t, p.Validate(), "profile default is not configure yet, run `aliyun configure --profile default` first")

	p.Mode = AK
	assert.Nil(t, p.Validate())

	p.Mode = StsToken
	assert.EqualError(t, p.Validate(), "invailed sts_token")

	p.Mode = RamRoleArn
	assert.EqualError(t, p.Validate(), "invailed ram_role_arn")
	p.RamRoleArn = "RamRoleArn"
	assert.EqualError(t, p.Validate(), "invailed role_session_name")

	p.Mode = EcsRamRole
	//TODO maybe there is a problem

	p.Mode = RsaKeyPair
	assert.EqualError(t, p.Validate(), "invailed private_key")
	p.PrivateKey = "******"
	assert.EqualError(t, p.Validate(), "invailed key_pair_name")

	p.Mode = "free"
	assert.EqualError(t, p.Validate(), "invailed mode: free")

	//GetParent
	assert.Nil(t, p.GetParent())

	//OverwriteWithFlags
	w := new(bytes.Buffer)
	ctx := cli.NewCommandContext(w)
	AddFlags(ctx.Flags())

	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.StsToken = ""
	p.RegionId = ""
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: "free", RamRoleArn: "RamRoleArn", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, RetryTimeout: 0, Language: "en"}, p)

	p.AccessKeyId = "****"
	p.AccessKeySecret = "++++"
	p.StsToken = "----"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: StsToken, StsToken: "----", AccessKeyId: "****", AccessKeySecret: "++++", RamRoleArn: "RamRoleArn", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, RetryTimeout: 0, Language: "en"}, p)

	p.StsToken = ""
	p.RamRoleArn = "----"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: RamRoleArn, RamRoleArn: "----", AccessKeyId: "****", AccessKeySecret: "++++", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, RetryTimeout: 0, Language: "en"}, p)

	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.PrivateKey = "****"
	p.KeyPairName = "++++"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: RsaKeyPair, KeyPairName: "++++", RamRoleArn: "----", PrivateKey: "****", OutputFormat: "json", RetryCount: 0, RetryTimeout: 0, Language: "en"}, p)

	p.PrivateKey = ""
	p.KeyPairName = ""
	p.RamRoleName = "RamRoleName"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: EcsRamRole, RamRoleArn: "----", RamRoleName: "RamRoleName", OutputFormat: "json", RetryCount: 0, RetryTimeout: 0, Language: "en"}, p)

	//GetClient
	p.RetryTimeout = 1
	p.RetryCount = 1
	p.Mode = "free"
	sdkClient, err := p.GetClient(ctx)
	assert.Nil(t, sdkClient)
	assert.EqualError(t, err, "unexcepted certificate mode: free")

	p.Mode = AK
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, sdkClient)
	assert.NotNil(t, err)

	p.Mode = StsToken
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	p.Mode = RamRoleArn
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, sdkClient)
	assert.NotNil(t, err)

	p.Mode = EcsRamRole
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, sdkClient)
	assert.NotNil(t, err)

	p.Mode = RsaKeyPair
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	//GetSessionCredential
	p.Mode = AK
	sessionCredential, err := p.GetSessionCredential()
	assert.Nil(t, err)
	assert.Equal(t, &signers.SessionCredential{AccessKeyId: p.AccessKeyId, AccessKeySecret: p.AccessKeySecret}, sessionCredential)

	p.Mode = StsToken
	sessionCredential, err = p.GetSessionCredential()
	assert.Nil(t, err)
	assert.Equal(t, &signers.SessionCredential{AccessKeyId: p.AccessKeyId, AccessKeySecret: p.AccessKeySecret, StsToken: p.StsToken}, sessionCredential)

	p.Mode = RamRoleArn
	sessionCredential, err = p.GetSessionCredential()
	assert.NotNil(t, err)
	assert.Nil(t, sessionCredential)

	p.Mode = EcsRamRole
	sessionCredential, err = p.GetSessionCredential()
	assert.NotNil(t, err)
	assert.Nil(t, sessionCredential)

	p.Mode = "free"
	sessionCredential, err = p.GetSessionCredential()
	assert.EqualError(t, err, "unsupported mode 'free' to GetSessionCredential")
	assert.Nil(t, sessionCredential)

	//GetClientByAK
	sdkCF := &sdk.Config{}
	sdkClient, err = p.GetClientByAK(sdkCF)
	assert.Nil(t, sdkClient)
	assert.EqualError(t, err, "AccessKeyId/AccessKeySecret is empty! run `aliyun configure` first")

	p.AccessKeyId = "AccessKeyId"
	p.AccessKeySecret = "AccessKeySecret"
	p.RegionId = ""
	sdkClient, err = p.GetClientByAK(sdkCF)
	assert.Nil(t, sdkClient)
	assert.EqualError(t, err, "default RegionId is empty! run `aliyun configure` first")

	p.RegionId = "cn-hangzhou"
	sdkClient, err = p.GetClientByAK(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	//GetClientBySts
	p.StsToken = "StsToken"
	sdkClient, err = p.GetClientBySts(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	//GetSessionCredentialByRoleArn
	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	sessionCredential, err = p.GetSessionCredentialByRoleArn()
	assert.Nil(t, sessionCredential)
	assert.True(t, strings.HasPrefix(err.Error(), "sts:AssumeRole() failed"))

	//start hook
	orighookAssumeRole := hookAssumeRole
	hookAssumeRole = func(fn func(request *sts.AssumeRoleRequest) (response *sts.AssumeRoleResponse, err error)) func(request *sts.AssumeRoleRequest) (response *sts.AssumeRoleResponse, err error) {
		return func(request *sts.AssumeRoleRequest) (response *sts.AssumeRoleResponse, err error) {
			return &sts.AssumeRoleResponse{RequestId: "RequestId", Credentials: sts.Credentials{SecurityToken: "SecurityToken", AccessKeySecret: "AccessKeySecret", AccessKeyId: "AccessKeyId", Expiration: "Expiration"}}, nil
		}
	}
	sessionCredential, err = p.GetSessionCredentialByRoleArn()
	assert.Nil(t, err)
	assert.Equal(t, &signers.SessionCredential{AccessKeyId: "AccessKeyId", AccessKeySecret: "AccessKeySecret", StsToken: "SecurityToken"}, sessionCredential)

	//GetClientByRoleArn
	sdkClient, err = p.GetClientByRoleArn(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	hookAssumeRole = orighookAssumeRole

	//GetSessionCredentialByEcsRamRole
	orighookHTTPGet := hookHTTPGet
	orighookUnmarshal := hookUnmarshal

	//testcase 1
	hookHTTPGet = func(fn func(url string) (resp *http.Response, err error)) func(url string) (resp *http.Response, err error) {
		return func(url string) (resp *http.Response, err error) {
			return nil, errors.New("mock err")
		}
	}
	p.RamRoleName = ""
	sessionCredential, err = p.GetSessionCredentialByEcsRamRole()
	assert.Nil(t, sessionCredential)
	assert.EqualError(t, err, "Get default RamRole error: mock err. Or Run `aliyun configure` to configure it.")

	//testcase 2
	hookHTTPGet = func(fn func(url string) (resp *http.Response, err error)) func(url string) (resp *http.Response, err error) {
		return func(url string) (resp *http.Response, err error) {
			return new(http.Response), nil
		}
	}
	hookUnmarshal = func(fn func(response responses.AcsResponse, httpResponse *http.Response, format string) (err error)) func(response responses.AcsResponse, httpResponse *http.Response, format string) (err error) {
		return func(response responses.AcsResponse, httpResponse *http.Response, format string) (err error) {
			return nil
		}
	}
	sessionCredential, err = p.GetSessionCredentialByEcsRamRole()
	assert.Nil(t, sessionCredential)
	assert.EqualError(t, err, "Get meta-data status=0 please check RAM settings. Or Run `aliyun configure` to configure it.")

	hookHTTPGet = orighookHTTPGet
	hookUnmarshal = orighookUnmarshal

	//GetClientByPrivateKey
	sdkClient, err = p.GetClientByPrivateKey(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

}

func TestIsRegion(t *testing.T) {
	assert.False(t, IsRegion("#$adf"))
	assert.True(t, IsRegion("2kf"))
}
