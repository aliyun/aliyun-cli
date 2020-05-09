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
		ReadTimeout:     0,
		ConnectTimeout:  0,
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
	ALIBABACLOUD_REGION_ID_env := os.Getenv("ALIBABACLOUD_REGION_ID")
	if ALIBABACLOUD_REGION_ID_env != "" {
		os.Setenv("ALIBABACLOUD_REGION_ID", "")
		defer func() {
			os.Setenv("ALIBABACLOUD_REGION_ID", ALIBABACLOUD_REGION_ID_env)
		}()
	}
	REGION_env := os.Getenv("REGION")
	if REGION_env != "" {
		os.Setenv("REGION", "")
		defer func() {
			os.Setenv("REGION", REGION_env)
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
	assert.Equal(t, Profile{Name: "default", Mode: "free", RamRoleArn: "RamRoleArn", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	p.Mode = StsToken
	p.AccessKeyId = "****"
	p.AccessKeySecret = "++++"
	p.StsToken = "----"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: StsToken, StsToken: "----", AccessKeyId: "****", AccessKeySecret: "++++", RamRoleArn: "RamRoleArn", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	p.Mode = RamRoleArn
	p.StsToken = ""
	p.RamRoleArn = "----"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: RamRoleArn, RamRoleArn: "----", AccessKeyId: "****", AccessKeySecret: "++++", PrivateKey: "******", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	p.Mode = RsaKeyPair
	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.PrivateKey = "****"
	p.KeyPairName = "++++"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: RsaKeyPair, KeyPairName: "++++", RamRoleArn: "----", PrivateKey: "****", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	p.Mode = EcsRamRole
	p.PrivateKey = ""
	p.KeyPairName = ""
	p.RamRoleName = "RamRoleName"
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: EcsRamRole, RamRoleArn: "----", RamRoleName: "RamRoleName", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	p.Mode = AK
	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	os.Setenv("ACCESS_KEY_ID", "accessKeyID")
	os.Setenv("ACCESS_KEY_SECRET", "accessKeySecret")
	os.Setenv("REGION", "cn-beijing")
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret", RamRoleName: "RamRoleName", RamRoleArn: "----", RegionId: "cn-beijing", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)
	p.RegionId = ""
	os.Setenv("ALIBABACLOUD_REGION_ID", "cn-hangzhou")
	p.OverwriteWithFlags(ctx)
	assert.Equal(t, Profile{Name: "default", Mode: AK, AccessKeyId: "accessKeyID", AccessKeySecret: "accessKeySecret", RamRoleName: "RamRoleName", RamRoleArn: "----", RegionId: "cn-hangzhou", OutputFormat: "json", RetryCount: 0, ReadTimeout: 0, Language: "en"}, p)

	os.Setenv("ACCESS_KEY_ID", "")
	os.Setenv("ACCESS_KEY_SECRET", "")
	os.Setenv("ALIBABACLOUD_REGION_ID", "")
	os.Setenv("REGION", "")

	p.AccessKeyId = ""
	p.AccessKeySecret = ""
	p.RegionId = ""

	//GetClient
	p.ReadTimeout = 1
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
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	p.Mode = EcsRamRole
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	p.Mode = RamRoleArnWithEcs
	sdkClient, err = p.GetClient(ctx)
	assert.NotNil(t, err)
	assert.Nil(t, sdkClient)

	p.Mode = RsaKeyPair
	sdkClient, err = p.GetClient(ctx)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

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

	//GetClientByRoleArn
	sdkClient, err = p.GetClientByRoleArn(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

	//GetClientByRamRoleArnWithEcs
	sdkClient, err = p.GetClientByRamRoleArnWithEcs(sdkCF)
	assert.NotNil(t, err)
	assert.Nil(t, sdkClient)

	//GetClientByPrivateKey
	sdkClient, err = p.GetClientByPrivateKey(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

}

func TestIsRegion(t *testing.T) {
	assert.False(t, IsRegion("#$adf"))
	assert.True(t, IsRegion("2kf"))
}
