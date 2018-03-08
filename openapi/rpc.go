/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/meta"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
)

func (c *Caller) InvokeRpc(ctx *cli.Context, product *meta.Product, apiName string) error {
	api, ok := c.library.GetApi(product.Code, product.Version, apiName)
	if !ok {
		return &InvalidApiError{Name: apiName, product: product}
	}

	//
	// call OpenApi
	// return: if check failed return error, otherwise return nil
	client, request, err := c.InitClient(ctx, product, true)
	if err != nil {
		return nil
	}

	request.ApiName = apiName
	err = c.FillRpcParameters(ctx, request, &api)
	if err != nil {
		return err
	}

	request.Scheme = api.GetProtocol()
	request.Method = api.GetMethod()

	err = c.UpdateRequest(ctx, request)
	if err != nil {
		return err
	}

	resp, err := client.ProcessCommonRequest(request)

	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}
	fmt.Println(resp.GetHttpContentString())
	return nil
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
	err = c.FillRpcParameters(ctx, request, nil)
	if err != nil {
		ctx.Command().PrintFailed(err, "")
		return
	}

	err = c.UpdateRequest(ctx, request)
	if err == nil {
		ctx.Command().PrintFailed(err, "")
		return
	}

	resp, err := client.ProcessCommonRequest(request)
	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}

	fmt.Println(resp.GetHttpContentString())
}

func (c *Caller) FillRpcParameters(ctx *cli.Context, request *requests.CommonRequest, api *meta.Api) error {
	for _, f := range ctx.UnknownFlags().Flags() {
		if api != nil {
			param := api.FindParameter(f.Name)
			if param == nil {
				return &InvalidParameterError{Name: f.Name, api: api, flags: ctx.Flags()}
			}
			if param.Position == "Query" {
				request.QueryParams[f.Name] = f.GetValue()
			} else if param.Position == "Body" {
				request.FormParams[f.Name] = f.GetValue()
			} else {
				return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
			}
			//return fmt.Errorf("unknown parameter %s", f.Name)
		} else {
			request.QueryParams[f.Name] = f.GetValue()
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
			return fmt.Errorf("%s \n\n use `aliyun %s %s --help` to get more information",
				err.Error(), api.Product.GetLowerCode(), api.Name)
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
	request.Headers["User-Agent"] = "Aliyun-CLI-V0.60"
	request.RegionId = c.profile.RegionId
	request.Product = product.Code
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

func (c *Caller) UpdateRequest(ctx *cli.Context, request *requests.CommonRequest) error {
	if _, ok := ctx.Flags().GetValue("secure"); ok {
		request.Scheme = "https"
	}

	if f := ctx.Flags().Get("header"); f != nil {
		for _, v := range f.GetValues() {
			if k2, v2, ok := cli.SplitWith(v, "="); ok {
				request.Headers[k2] = v2
			} else {
				return fmt.Errorf("invaild flag --header `%s` use `--header HeaderName=Value`", v)
			}
		}
	}
	return nil
}