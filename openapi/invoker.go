/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"strings"
)

//
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

//
// implementations
// - Waiter
// - Pager
type InvokeHelper interface {
	CallWith(invoker Invoker) (string, error)
}

//
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
	a.client, err = a.profile.GetClient(ctx)
	if err != nil {
		return fmt.Errorf("init client failed %s", err)
	}

	a.client.AppendUserAgent("Aliyun-CLI", cli.GetVersion())
	a.request = requests.NewCommonRequest()
	a.request.Product = product.Code

	a.request.RegionId = a.profile.RegionId
	if v, ok := config.RegionFlag(ctx.Flags()).GetValue(); ok {
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
		hint = fmt.Sprintf("see '%s' or `aliyun help %s` get more information.",
			product.GetDocumentLink(i18n.GetLanguage()), product.GetLowerCode())
	}

	if a.request.Version == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing version for product %s", product.Code),
			"Use flag `--version <YYYY-MM-DD>` to assign version, "+hint)
	}

	if a.request.RegionId == "" {
		return cli.NewErrorWithTip(fmt.Errorf("missing region for product %s", product.Code),
			"Use flag --region <regionId> to assign region, "+hint)
	}

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
