package openapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestWaiter_CallWith(t *testing.T) {
	waiter := GetWaiter()
	assert.Nil(t, waiter)

	originWaiterFlag := WaiterFlag
	WaiterFlag.SetAssigned(true)
	waiter = GetWaiter()
	assert.NotNil(t, waiter)

	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	invoker := &RpcInvoker{
		BasicInvoker: &BasicInvoker{
			client:  client,
			request: requests.NewCommonRequest(),
		},
	}

	WaiterFlag.Fields[2].SetAssigned(true)
	WaiterFlag.Fields[3].SetAssigned(true)
	str, err := waiter.CallWith(invoker)
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Equal(t, "--waiter timeout= must be integer", err.Error())

	WaiterFlag.Fields[2].SetValue("180")
	str, err = waiter.CallWith(invoker)
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Equal(t, "--waiter interval= must be integer", err.Error())

	WaiterFlag.Fields[3].SetValue("5")
	str, err = waiter.CallWith(invoker)
	WaiterFlag = originWaiterFlag
	assert.Equal(t, "", str)
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[SDK.CanNotResolveEndpoint] Can not resolve endpoint")
}
