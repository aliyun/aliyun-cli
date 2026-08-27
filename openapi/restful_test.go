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
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"testing"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/stretchr/testify/assert"
)

func TestRestfulInvoker_Prepare(t *testing.T) {
	a := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
	}

	a.BasicInvoker.request.RegionId = "cn-hangzhou"
	a.BasicInvoker.request.Content = []byte("{")
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)

	bodyflag := NewBodyFlag()
	bodyflag.SetAssigned(true)
	ctx.Flags().Add(bodyflag)

	secureflag := NewSecureFlag()
	secureflag.SetAssigned(true)
	ctx.Flags().Add(secureflag)
	ctx.Flags().Add(NewInsecureFlag())

	bodyfile := NewBodyFileFlag()
	bodyfile.SetAssigned(true)
	ctx.Flags().Add(bodyfile)

	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.UnknownFlags().Add(NewBodyFlag())
	err := a.Prepare(ctx)
	assert.Nil(t, err)

	BodyFlag(ctx.Flags()).SetAssigned(false)
	BodyFileFlag(ctx.Flags()).SetAssigned(false)
	a.BasicInvoker.request.Content = []byte("{")
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	a.BasicInvoker.request.Headers = map[string]string{}
	a.BasicInvoker.request.Content = []byte("<")
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	// testcase 2 - using mock API since cs product not in canonical
	a = &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			request: requests.NewCommonRequest(),
		},
		path:   "/k8s/[ClusterId]/user_config",
		method: "GET",
	}
	a.request.RegionId = "cn-hangzhou"

	// Create mock API with ClusterId parameter
	mockApi := &canonicalmeta.API{
		Name: "DescribeClusterUserKubeconfig", Parameters: []canonicalmeta.Parameter{
			{
				Name: "ClusterId", RawName: "ClusterId",
				Location: "path",
				Type:     "String",
				Required: true,
			},
		},
	}
	a.api = mockApi

	w = new(bufio.Writer)
	stderr = new(bufio.Writer)
	ctx = cli.NewCommandContext(w, stderr)
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.Flags().Add(NewBodyFlag())
	ctx.Flags().Add(NewSecureFlag())
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(NewBodyFileFlag())
	ctx.UnknownFlags().AddByName("ClusterId")
	ctx.UnknownFlags().Get("ClusterId").SetValue("cluster_id")
	err = a.Prepare(ctx)
	assert.Nil(t, err)

	ctx.UnknownFlags().AddByName("TestFlag")
	ctx.UnknownFlags().Get("TestFlag").SetValue("testFlagValue")
	err = a.Prepare(ctx)
	assert.EqualError(t, err, `"--TestFlag" is not a valid parameter or flag. See `+"`aliyun help  DescribeClusterUserKubeconfig`"+`.`)
}

func TestRestfulInvokerPrepareWildcardPath(t *testing.T) {
	a := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{request: requests.NewCommonRequest()},
		path:         "/api/v1/providers/[provider]/products/[product]/resources/*",
		method:       "GET",
		api: &canonicalmeta.API{Parameters: []canonicalmeta.Parameter{{
			Name: "request_path", RawName: "requestPath", Location: "path",
			Type: "string", Required: true, IsWildcard: true,
		}}},
	}
	ctx := cli.NewCommandContext(new(bufio.Writer), new(bufio.Writer))
	ctx.SetUnknownFlags(cli.NewFlagSet())
	ctx.Flags().Add(NewBodyFlag())
	ctx.Flags().Add(NewSecureFlag())
	ctx.Flags().Add(NewInsecureFlag())
	ctx.Flags().Add(NewBodyFileFlag())
	ctx.UnknownFlags().AddByName("requestPath")
	ctx.UnknownFlags().Get("requestPath").SetAssigned(true)
	ctx.UnknownFlags().Get("requestPath").SetValue("/api/v1/providers/qqq/products/dd/resources/dddd:4")

	if err := a.Prepare(ctx); err != nil {
		t.Fatalf("Prepare: %v", err)
	}
	if got, want := a.request.PathPattern, "/api/v1/providers/qqq/products/dd/resources/dddd:4"; got != want {
		t.Fatalf("PathPattern = %q, want %q", got, want)
	}
	if len(a.request.PathParams) != 0 {
		t.Fatalf("wildcard path must not remain in PathParams: %#v", a.request.PathParams)
	}
	a.request.TransToAcsRequest()
	if got, want := a.request.BuildQueries(), "/api/v1/providers/qqq/products/dd/resources/dddd:4"; got != want {
		t.Fatalf("signature resource path = %q, want %q", got, want)
	}
}

func TestRestfulInvoker_Call(t *testing.T) {
	client, err := sdk.NewClientWithAccessKey("regionid", "accesskeyid", "accesskeysecret")
	assert.Nil(t, err)

	a := &RestfulInvoker{
		BasicInvoker: &BasicInvoker{
			client:  client,
			request: requests.NewCommonRequest(),
		},
	}
	_, err = a.Call()
	assert.NotNil(t, err)
	assert.Contains(t, err.Error(), "[SDK.CanNotResolveEndpoint] Can not resolve endpoint")
}

func Test_checkRestfulMethod(t *testing.T) {
	w := new(bufio.Writer)
	stderr := new(bufio.Writer)
	ctx := cli.NewCommandContext(w, stderr)
	methodOrPath := "get"
	pathPattern := "/user"
	ok, method, path, err := checkRestfulMethod(ctx, methodOrPath, "")
	assert.False(t, ok)
	assert.Nil(t, err)
	assert.Equal(t, "", method)
	assert.Equal(t, "", path)

	ok, method, path, err = checkRestfulMethod(ctx, methodOrPath, pathPattern)
	assert.True(t, ok)
	assert.Nil(t, err)
	assert.Equal(t, "GET", method)
	assert.Equal(t, "/user", path)

	pathPattern = "user"
	ok, method, path, err = checkRestfulMethod(ctx, methodOrPath, pathPattern)
	assert.True(t, ok)
	assert.NotNil(t, err)
	assert.Equal(t, "bad restful path user", err.Error())
	assert.Equal(t, "GET", method)
	assert.Equal(t, "", path)

	ctx.Flags().Add(NewRoaFlag())
	methodOrPath = "update"
	ok, method, path, err = checkRestfulMethod(ctx, methodOrPath, pathPattern)
	assert.False(t, ok)
	assert.Nil(t, err)
	assert.Equal(t, "", method)
	assert.Equal(t, "", path)

	RoaFlag(ctx.Flags()).SetAssigned(true)
	RoaFlag(ctx.Flags()).SetValue("get")
	ok, method, path, err = checkRestfulMethod(ctx, methodOrPath, pathPattern)
	assert.True(t, ok)
	assert.NotNil(t, err)
	assert.Equal(t, "bad restful path update", err.Error())
	assert.Equal(t, "get", method)
	assert.Equal(t, "", path)
}
