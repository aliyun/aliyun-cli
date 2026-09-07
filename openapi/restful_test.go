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
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRestfulMethodsCompatibility(t *testing.T) {
	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	ctx.Flags().Add(NewRoaFlag())
	for _, input := range []string{"GET", "POST", "PUT", "DELETE", "PATCH", "patch", "PaTcH"} {
		t.Run(input, func(t *testing.T) {
			ok, method, path, err := checkRestfulMethod(ctx, input, "/items/1")
			require.NoError(t, err)
			require.True(t, ok)
			assert.Equal(t, strings.ToUpper(input), method)
			assert.Equal(t, "/items/1", path)
			_, _, _, err = checkRestfulMethod(ctx, input, "items/1")
			assert.EqualError(t, err, "bad restful path items/1")
			ok, _, _, err = checkRestfulMethod(ctx, input, "")
			assert.NoError(t, err)
			assert.False(t, ok)
		})
	}
	for _, input := range []string{"HEAD", "OPTIONS", "TRACE", "GET|POST", "PATCH|PUT", "PATC", " PATCH "} {
		_, ok := checkHttpMethod(input)
		assert.False(t, ok, "%q must remain unsupported", input)
	}
}

func TestCreateInvokerPatch(t *testing.T) {
	for _, product := range []string{"restful-demo", "unknown-demo"} {
		t.Run(product, func(t *testing.T) {
			profile := config.Profile{Mode: "AK", AccessKeyId: "testid", AccessKeySecret: "testsecret", RegionId: "cn-hangzhou", Endpoint: "example.com"}
			command := NewCommando(new(bytes.Buffer), profile)
			repo, err := meta.MockLoadRepository([]meta.Product{{Code: "restful-demo", Version: "2026-01-01", ApiStyle: "restful"}})
			require.NoError(t, err)
			command.library.builtinRepo = repo
			ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
			ctx.EnterCommand(&cli.Command{Name: "PATCH", EnableUnknownFlag: true})
			config.AddFlags(ctx.Flags())
			AddFlags(ctx.Flags())
			ForceFlag(ctx.Flags()).SetAssigned(true)
			DryRunJsonFlag(ctx.Flags()).SetAssigned(true)
			if product == "unknown-demo" {
				VersionFlag(ctx.Flags()).SetAssigned(true)
				VersionFlag(ctx.Flags()).SetValue("2026-01-01")
			}
			BodyFlag(ctx.Flags()).SetAssigned(true)
			BodyFlag(ctx.Flags()).SetValue(`{"name":"updated"}`)
			invoker, err := command.createInvoker(ctx, product, "PATCH", "/items/1")
			require.NoError(t, err)
			roa, ok := invoker.(*RestfulInvoker)
			require.True(t, ok, "PATCH must not fall back to an RPC action")
			require.NoError(t, roa.Prepare(ctx))
			assert.Equal(t, "PATCH", roa.request.Method)
			assert.Equal(t, "/items/1", roa.request.PathPattern)
			assert.Equal(t, `{"name":"updated"}`, string(roa.request.Content))
			assert.Equal(t, "application/json", roa.request.Headers["Content-Type"])
		})
	}
}

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

	missing := filepath.Join(t.TempDir(), "missing.json")
	BodyFileFlag(ctx.Flags()).SetAssigned(true)
	BodyFileFlag(ctx.Flags()).SetValue(missing)
	err = a.Prepare(ctx)
	var invalidBodyFile *InvalidBodyFileError
	assert.ErrorAs(t, err, &invalidBodyFile)
	assert.Equal(t, missing, invalidBodyFile.Path)
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
