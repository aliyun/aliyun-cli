package lib

import (
	"testing"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/stretchr/testify/assert"
)

func TestCliBridge(t *testing.T) {
	NewCommandBridge(configCommand.command)
}
func TestGetSessionCredential(t *testing.T) {
	profile := config.Profile{
		Name:            "ramRoleArnProfile",
		Mode:            "RamRoleArn",
		AccessKeyId:     "yourAccessKeyId",
		AccessKeySecret: "yourAccessKeySecret",
		StsRegion:       "cn-hangzhou",
		RamRoleArn:      "yourRamRoleArn",
		RoleSessionName: "test",
		ExpiredSeconds:  900,
		RegionId:        "cn-hangzhou",
	}
	ak, sk, token, e := getSessionCredential(&profile, nil)
	if e != nil {
		assert.Error(t, e)
	} else {
		assert.NotNil(t, ak)
		assert.NotNil(t, sk)
		assert.NotNil(t, token)
	}
	ak, sk, token, e = getSessionCredential(&profile, tea.String("http://127.0.0.1"))
	if e != nil {
		assert.Error(t, e)
	} else {
		assert.NotNil(t, ak)
		assert.NotNil(t, sk)
		assert.NotNil(t, token)
	}
	ak, sk, token, e = getSessionCredential(&profile, tea.String("127.0.0.1:8089"))
	if e != nil {
		assert.EqualError(t, e, "refresh RoleArn sts token err: parse \"127.0.0.1:8089\": first path segment in URL cannot contain colon")
	} else {
		assert.NotNil(t, ak)
		assert.NotNil(t, sk)
		assert.NotNil(t, token)
	}
	ak, sk, token, e = getSessionCredential(&profile, tea.String("http://127.0.0.1:8089"))
	if e != nil {
		assert.Error(t, e)
	} else {
		assert.NotNil(t, ak)
		assert.NotNil(t, sk)
		assert.NotNil(t, token)
	}
}
