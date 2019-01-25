/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"bytes"
	"strings"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/signers"
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

	//GetClientByRoleArn
	// p.GetClientByRoleArn(sdkCF)

	//GetClientByPrivateKey
	sdkClient, err = p.GetClientByPrivateKey(sdkCF)
	assert.Nil(t, err)
	assert.NotNil(t, sdkClient)

}

func TestIsRegion(t *testing.T) {
	assert.False(t, IsRegion("#$adf"))
	assert.True(t, IsRegion("2kf"))
}
