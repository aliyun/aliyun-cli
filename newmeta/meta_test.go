package newmeta

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMetadataPrefix(t *testing.T) {
	assert.Equal(t, "zh-CN", GetMetadataPrefix(""))
	assert.Equal(t, "en-US", GetMetadataPrefix("en"))
	assert.Equal(t, "zh-CN", GetMetadataPrefix("zh"))
}

func TestGetMetadata(t *testing.T) {
	content, err := GetMetadata("en", "/products.json")
	assert.Nil(t, err)
	assert.Greater(t, len(content), 100)
}

func TestGetProductName(t *testing.T) {
	name, err := GetProductName("en", "ecs")
	assert.Nil(t, err)
	assert.Equal(t, "Elastic Compute Service", name)
	name, err = GetProductName("zh", "ecs")
	assert.Nil(t, err)
	assert.Equal(t, "云服务器 ECS", name)
}

func TestGetAPI(t *testing.T) {
	api, err := GetAPI("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.Equal(t, "DescribeRegions", api.Title)
	assert.Equal(t, "Queries available Alibaba Cloud regions.", api.Summary)
	assert.Equal(t, false, api.Deprecated)

	api2, err := GetAPI("en", "ecs", "Invalid")
	assert.Nil(t, err)
	assert.Nil(t, api2)
}

func TestGetAPIDetail(t *testing.T) {
	api, err := GetAPIDetail("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.Equal(t, "DescribeRegions", api.Name)
	assert.Equal(t, "GET|POST", api.Method)
	assert.Equal(t, false, api.Deprecated)
}

func TestIsAnonymousAPI(t *testing.T) {
	akapi, err := GetAPIDetail("en", "ecs", "DescribeRegions")
	assert.Nil(t, err)
	assert.False(t, akapi.IsAnonymousAPI())
	api, err := GetAPIDetail("en", "sts", "AssumeRoleWithOIDC")
	assert.Nil(t, err)
	assert.True(t, api.IsAnonymousAPI())
}
