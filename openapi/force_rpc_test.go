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
package openapi

import (
	"bufio"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/stretchr/testify/assert"
)

func TestForceRpcInvoker_Prepare(t *testing.T) {
	a := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		method: "DescribeRegion",
	}
	a.BasicInvoker.request.QueryParams = make(map[string]string)
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	cmd := config.NewConfigureCommand()
	cmd.EnableUnknownFlag = true
	ctx.EnterCommand(cmd)

	secureflag := NewSecureFlag()
	methodflag := NewMethodFlag()
	secureflag.SetAssigned(true)
	methodflag.SetAssigned(true)
	methodflag.SetValue("POST")
	ctx.Flags().Add(secureflag)
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(methodflag)
	ctx.UnknownFlags().Add(NewSecureFlag())
	err := a.Prepare(ctx)
	assert.Nil(t, err)
}

func TestForceRpcInvoker_Call(t *testing.T) {
	a := &ForceRpcInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		method: "DescribeRegion",
	}
	client, err := sdk.NewClientWithAccessKey("regionid", "acesskeyid", "accesskeysecret")
	assert.Nil(t, err)
	a.client = client
	_, err = a.Call()
	assert.NotNil(t, err)
}
