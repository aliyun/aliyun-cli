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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/auth/credentials"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

func GetClient(cp *config.Profile, ctx *cli.Context) (client *sdk.Client, err error) {
	credential, err := cp.GetCredential(ctx, nil)
	if err != nil {
		return
	}

	model, err := credential.GetCredential()
	if err != nil {
		return
	}

	var cred auth.Credential
	if model.SecurityToken != nil {
		cred = credentials.NewStsTokenCredential(*model.AccessKeyId, *model.AccessKeySecret, *model.SecurityToken)
	} else {
		cred = credentials.NewAccessKeyCredential(*model.AccessKeyId, *model.AccessKeySecret)
	}

	if cp.RegionId == "" {
		err = fmt.Errorf("default RegionId is empty! run `aliyun configure` first")
		return
	}

	conf := sdk.NewConfig()
	client, err = sdk.NewClientWithOptions(cp.RegionId, conf, cred)
	if err != nil {
		return
	}
	// get UserAgent from env
	conf.UserAgent = os.Getenv("ALIYUN_USER_AGENT")

	if cp.RetryCount > 0 {
		// when use --retry-count, enable auto retry
		conf.WithAutoRetry(true)
		conf.WithMaxRetryTime(cp.RetryCount)
	}

	if client != nil {
		if cp.ReadTimeout > 0 {
			client.SetReadTimeout(time.Duration(cp.ReadTimeout) * time.Second)
		}
		if cp.ConnectTimeout > 0 {
			client.SetConnectTimeout(time.Duration(cp.ConnectTimeout) * time.Second)
		}
		if config.SkipSecureVerify(ctx.Flags()).IsAssigned() {
			client.SetHTTPSInsecure(true)
		}
	}
	return client, err
}

// implementations:
// - RpcInvoker,
// - RpcForceInvoker
// - RestfulInvoker
type Invoker interface {
	getClient() *sdk.Client
	getRequest() *requests.CommonRequest
	Prepare(ctx *cli.Context) error
	Call() (*responses.CommonResponse, error)
}

// implementations
// - Waiter
// - Pager
type InvokeHelper interface {
	CallWith(invoker Invoker) (string, error)
}

// basic invoker to init common object and headers
type BasicInvoker struct {
	profile *config.Profile
	client  *sdk.Client
	request *requests.CommonRequest
	product *meta.Product
}

func NewBasicInvoker(cp *config.Profile) *BasicInvoker {
	return &BasicInvoker{profile: cp}
}

func (a *BasicInvoker) getClient() *sdk.Client {
	return a.client
}

func (a *BasicInvoker) getRequest() *requests.CommonRequest {
	return a.request
}

func (a *BasicInvoker) Init(ctx *cli.Context, product *meta.Product) error {
	var err error
	a.product = product
	a.request = requests.NewCommonRequest()
	a.request.Product = product.Code

	a.request.RegionId = a.profile.RegionId
	if v, ok := config.RegionFlag(ctx.Flags()).GetValue(); ok {
		a.request.RegionId = v
	} else if v, ok := config.RegionIdFlag(ctx.Flags()).GetValue(); ok {
		a.request.RegionId = v
	}

	a.request.Version = product.Version
	if v, ok := VersionFlag(ctx.Flags()).GetValue(); ok {
		a.request.Version = v
	}

	if v, ok := EndpointFlag(ctx.Flags()).GetValue(); ok {
		a.request.Domain = v
	}

	for _, s := range HeaderFlag(ctx.Flags()).GetValues() {
		if k, v, ok := cli.SplitStringWithPrefix(s, "="); ok {
			a.request.Headers[k] = v
			if k == "Accept" {
				if strings.Contains(v, "xml") {
					a.request.AcceptFormat = "XML"
				} else if strings.Contains(v, "json") {
					a.request.AcceptFormat = "JSON"
				}
			}
			if k == "Content-Type" {
				a.request.SetContentType(v)
			}
		} else {
			return fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", s)
		}
	}

	hint := "you can find it on https://help.aliyun.com"
	if product.Version != "" {
		hint = fmt.Sprintf("please use `aliyun help %s` get more information.", product.GetLowerCode())
	}

	if a.request.Version == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing version for product %s", product.Code),
			"Use flag `--version <YYYY-MM-DD>` to assign version, "+hint)
	}

	if a.request.RegionId == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing region for product %s", product.Code),
			"Use flag --region <regionId> to assign region, "+hint)
	}

	a.client, err = GetClient(a.profile, ctx)
	if err != nil {
		return fmt.Errorf("init client failed %s", err)
	}
	if vendorEnv, ok := os.LookupEnv("ALIBABA_CLOUD_VENDOR"); ok {
		a.client.AppendUserAgent("vendor", vendorEnv)
	}
	a.client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())

	if a.request.Domain == "" {
		a.request.Domain, err = product.GetEndpoint(a.request.RegionId, a.client)
		if err != nil {
			return cli.NewErrorWithTip(
				fmt.Errorf("unknown endpoint for %s/%s! failed %s", product.GetLowerCode(), a.request.RegionId, err),
				"Use flag --endpoint xxx.aliyuncs.com to assign endpoint, "+hint)
		}
	}

	return nil
}
