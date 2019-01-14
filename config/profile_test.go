package config

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestProfile(t *testing.T) {
	exProfile := Profile{
		Name:            "MrX",
		Mode:            "AK",
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
		parent:          nil,
	}
	acProfile := NewProfile("MrX")
	assert.True(t, assert.ObjectsAreEqual(exProfile, acProfile))

	assert.EqualError(t, acProfile.Validate(), "region can't be empty")

	acProfile.RegionId = "324$%#"
	assert.EqualError(t, acProfile.Validate(), "invailed region 324$%#")

	acProfile.RegionId = "cn-nicai"
	assert.EqualError(t, acProfile.Validate(), "invalid access_key_id: ")

	acProfile.Mode = ""
	assert.EqualError(t, acProfile.Validate(), "profile MrX is not configure yet, run `aliyun configure --profile MrX` first")

	acProfile.Mode = "AK"
	acProfile.AccessKeyId = "AccessKeyID"
	acProfile.Validate()
	assert.EqualError(t, acProfile.Validate(), "invaild access_key_secret: ")
	acProfile.AccessKeySecret = "AccessKeySecret"
	assert.Nil(t, acProfile.Validate())

	acProfile.Mode = "StsToken"
	assert.EqualError(t, acProfile.Validate(), "invailed sts_token")
	acProfile.StsToken = "StsToken"
	assert.Nil(t, acProfile.Validate())

	acProfile.Mode = "RamRoleArn"
	assert.EqualError(t, acProfile.Validate(), "invailed ram_role_arn")
	acProfile.RamRoleArn = "RamRoleArn"
	assert.EqualError(t, acProfile.Validate(), "invailed role_session_name")
	acProfile.RoleSessionName = "RoleSessionName"
	assert.Nil(t, acProfile.Validate())

	//Test problem
	acProfile.Mode = "EcsRamRole"
	// assert.EqualError(t, acProfile.Validate(), "invailed ram_role_name")
	assert.Nil(t, acProfile.Validate())

	acProfile.Mode = "RsaKeyPair"
	assert.EqualError(t, acProfile.Validate(), "invailed private_key")
	acProfile.PrivateKey = "PrivateKey"
	assert.EqualError(t, acProfile.Validate(), "invailed key_pair_name")
	acProfile.KeyPairName = "KeyPairName"
	assert.Nil(t, acProfile.Validate())

	acProfile.Mode = "MrX"
	assert.EqualError(t, acProfile.Validate(), "invailed mode: MrX")

}
