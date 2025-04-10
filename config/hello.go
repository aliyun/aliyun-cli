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
package config

import (
	"fmt"
	"os"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	util "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
)

func doHello(ctx *cli.Context, profile *Profile) (err error) {
	profile.OverwriteWithFlags(ctx)
	credential, err := profile.GetCredential(ctx, nil)
	if err != nil {
		return
	}

	config := &openapi.Config{
		Credential: credential,
	}

	config.Endpoint = tea.String(getSTSEndpoint(profile.StsRegion))
	client, err := openapi.NewClient(config)
	if err != nil {
		return
	}

	params := &openapi.Params{
		// 接口名称
		Action: tea.String("GetCallerIdentity"),
		// 接口版本
		Version: tea.String("2015-04-01"),
		// 接口协议
		Protocol: tea.String("HTTPS"),
		// 接口 HTTP 方法
		Method:   tea.String("POST"),
		AuthType: tea.String("AK"),
		Style:    tea.String("RPC"),
		// 接口 PATH
		Pathname: tea.String("/"),
		// 接口请求体内容格式
		ReqBodyType: tea.String("json"),
		// 接口响应体内容格式
		BodyType: tea.String("json"),
	}
	// runtime options
	runtime := &util.RuntimeOptions{}
	request := &openapi.OpenApiRequest{}

	ua := "Aliyun-CLI/" + cli.GetVersion()
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		ua += " vendor/" + vendorEnv
	}

	client.UserAgent = tea.String(ua)
	_, err = client.CallApi(params, request, runtime)
	return
}

func DoHello(ctx *cli.Context, profile *Profile) {
	w := ctx.Stdout()
	err := doHello(ctx, profile)
	if err != nil {
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, err)
		cli.Println(w, "-----------------------------------------------")
		cli.Println(w, "!!! Configure Failed please configure again !!!")
		cli.Println(w, "-----------------------------------------------")
		return
	}

	fmt.Println(icon)
}

var icon = string(`
Configure Done!!!
..............888888888888888888888 ........=8888888888888888888D=..............
...........88888888888888888888888 ..........D8888888888888888888888I...........
.........,8888888888888ZI: ...........................=Z88D8888888888D..........
.........+88888888 ..........................................88888888D..........
.........+88888888 .......Welcome to use Alibaba Cloud.......O8888888D..........
.........+88888888 ............. ************* ..............O8888888D..........
.........+88888888 .... Command Line Interface(Reloaded) ....O8888888D..........
.........+88888888...........................................88888888D..........
..........D888888888888DO+. ..........................?ND888888888888D..........
...........O8888888888888888888888...........D8888888888888888888888=...........
............ .:D8888888888888888888.........78888888888888888888O ..............`)
