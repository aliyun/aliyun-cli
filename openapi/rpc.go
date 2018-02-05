package openapi

import (
	"github.com/aliyun/aliyun-cli/meta"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
)

func (c *Caller) InvokeRpc(ctx *cli.Context, product *meta.Product, apiName string) {
	api, ok := c.library.GetApi(product.Name, product.Version, apiName)
	if !ok {
		ctx.Command().PrintFailed(fmt.Errorf("invailed api: %s", apiName),
			fmt.Sprintf("Use\n `aliyun help %s` to view product list\n  or add --force to skip check", product.Name))
		return
	}

	//
	// call OpenApi
	// return: if check failed return error, otherwise return nil
	client, request, err := c.InitClient(ctx, product, true)
	if err != nil {
		ctx.Command().PrintFailed(fmt.Errorf("init failed: %v", err), "")
		return
	}

	request.ApiName = apiName
	err = c.FillRpcParameters(ctx, request, &api)
	if err != nil {
		ctx.Command().PrintFailed(fmt.Errorf("init failed: %v", err), "")
		return
	}
	resp, err := client.ProcessCommonRequest(request)

	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}
	fmt.Println(resp.GetHttpContentString())
}

func (c *Caller) InvokeRpcForce(ctx *cli.Context, product *meta.Product, apiName string) {
	//
	// call OpenApi
	// return: if check failed return error, otherwise return nil
	client, request, err := c.InitClient(ctx, product, true)

	if err != nil {
		ctx.Command().PrintFailed(fmt.Errorf("init client failed: %v", err), "")
		return
	}

	request.ApiName = apiName

	c.FillRpcParameters(ctx, request, nil)
	resp, err := client.ProcessCommonRequest(request)

	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}
	fmt.Println(resp.GetHttpContentString())
}


func (c *Caller) FillRpcParameters(ctx *cli.Context, request *requests.CommonRequest, api *meta.Api) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		request.QueryParams[f.Name] = f.GetValue()
		if api != nil && !api.HasParameter(f.Name){
			return fmt.Errorf("unknown parameter %s", f.Name)
		}
	}
	if api != nil {
		err := api.CheckRequiredParameters(func(s string) bool {
			switch s {
			case "RegionId":
				return request.RegionId != ""
			case "Action":
				return request.ApiName != ""
			default:
				return ctx.UnknownFlags().IsAssigned(s)
			}
		})
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *Caller) InitClient(ctx *cli.Context, product *meta.Product, isRpc bool) (*sdk.Client, *requests.CommonRequest, error){
	//
	// call OpenApi
	// return: if check failed return error, otherwise return nil
	client, err := c.profile.GetClient()
	if err != nil {
		return nil, nil, fmt.Errorf("bad client %v", err)
	}

	request := requests.NewCommonRequest()
	request.Headers["User-Agent"] = "Aliyun-CLI-V0.32"
	request.RegionId = c.profile.RegionId
	request.Product = product.Name
	request.Version = product.Version

	if v, ok := ctx.Flags().GetValue("region"); ok {
		request.RegionId = v
	}
	if v, ok := ctx.Flags().GetValue("endpoint"); ok {
		request.Domain = v
	}
	if v, ok := ctx.Flags().GetValue("version"); ok {
		request.Version = v
	}

	if v, ok := ctx.Flags().GetValue("content-type"); ok {
		request.SetContentType(v)
	} else if isRpc {
		request.SetContentType("application/json")
	}

	if v, ok := ctx.Flags().GetValue("accept"); ok {
		request.Headers["Accept"] = v
		if strings.Contains(v, "xml") {
			request.AcceptFormat = "XML"
		} else if strings.Contains(v, "json") {
			request.AcceptFormat = "JSON"
		}
	}
	if f := ctx.Flags().Get("header"); f != nil {
		for _, v := range f.GetValues() {
			if k2, v2, ok := cli.SplitWith(v, "="); ok {
				request.Headers[k2] = v2
			} else {
				return nil, nil, fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", v)
			}
		}
	}
	if request.Version == "" {
		return nil, nil, fmt.Errorf("unknown version! Use flag --version 2016-07-09 to assign version")
	}
	if request.Domain == "" {
		request.Domain, err = product.GetEndpoint(request.RegionId, client)
		if err != nil {
			return nil, nil, fmt.Errorf("unknown endpoint! Use flag --endpoint xxx.aliyuncs.com to assign endpoint")
		}
	}

	return client, request, nil
}
