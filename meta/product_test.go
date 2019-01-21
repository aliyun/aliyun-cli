package meta

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/stretchr/testify/assert"

	"testing"
)


func TestProduct_GetLowerCodeAndGetDocumentLink(t *testing.T) {
	product := &Product{
		Code: "code",
	}
	code := product.GetLowerCode()
	assert.Equal(t, code, "code")
	link := product.GetDocumentLink("link")
	assert.Equal(t, link, "https://help.aliyun.com/api/code")
}

func TestProduct_GetEndpoint(t *testing.T) {
	product := &Product{
		Code: "arms",
		RegionalEndpoints: map[string]string{
			"cn-hangzhou": "arms.cn-hangzhou.aliyuncs.com",
	    },
	    LocationServiceCode: "arms",
	}
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)
	endpoint, err := product.GetEndpoint("cn-hangzhou", client)
	assert.Nil(t, err)
	assert.Equal(t, endpoint, "arms.cn-hangzhou.aliyuncs.com")

	//endpoint, err = product.GetEndpoint("cn-hangzhou", client)
	//assert.Nil(t, err)
	//assert.Equal(t, endpoint, "arms-cn-hangzhou.aliyuncs.com")

	product.LocationServiceCode = ""
	product.GlobalEndpoint = "arms.aliyuncs.com"
	endpoint, err = product.GetEndpoint("us-west-1", client)
	assert.Nil(t, err)
	assert.Equal(t, endpoint, "arms.aliyuncs.com")

	product.GlobalEndpoint = ""
	endpoint, err = product.GetEndpoint("us-west-1", client)
	assert.NotNil(t, err)
	assert.Contains(t,  err.Error(), "us-west-1")
}

func TestProduct_TryGetEndpoints(t *testing.T) {
	product := &Product{
		Code: "arms",
		RegionalEndpoints: map[string]string{
			"cn-hangzhou": "arms.cn-hanghzou.aliyuncs.com",
		},
		LocationServiceCode: "arms",
	}
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)
	//endpoint, lcEndpoint := product.TryGetEndpoints("cn-hangzhou", client)
	//assert.Equal(t,"arms.cn-hanghzou.aliyuncs.com", endpoint)
	//assert.Equal(t, "", lcEndpoint)

	endpoint, lcEndpoint := product.TryGetEndpoints("cn-hangzhou", client)
	assert.Equal(t, "arms.cn-hanghzou.aliyuncs.com", endpoint)
	assert.Equal(t, "arms.cn-hangzhou.aliyuncs.com", lcEndpoint)
}