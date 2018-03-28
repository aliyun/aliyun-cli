/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/aliyun-cli/meta"
	"fmt"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/aliyun-cli/cli"
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
		return err
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

	if flag := ctx.Flags().Get("all-pages"); flag != nil {
		if flag.IsAssigned() {
			pager := NewPager(flag.GetValue())
			r, err := c.InvokeRpcWithPager(client, request, pager)
			if err != nil {
				ctx.Command().PrintFailed(err, "")
			} else {
				err = outputProcessor(ctx,r)
				if err != nil {
					ctx.Command().PrintFailed(err, "")
				}
			}
			return nil
		}
	}

	waiter := NewWaiterWithCTX(ctx, client, request)
	//resp, err := client.ProcessCommonRequest(request)

	resp, err := waiter.Wait()

	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}

	err = outputProcessor(ctx, resp.GetHttpContentString())
	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}

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
	if err != nil {
		ctx.Command().PrintFailed(err, "")
		return
	}

	resp, err := client.ProcessCommonRequest(request)
	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}

	err = outputProcessor(ctx, resp.GetHttpContentString())
	if err != nil {
		ctx.Command().PrintFailed(err, "")
	}
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

func (c *Caller) InvokeRpcWithPager(client *sdk.Client, request *requests.CommonRequest, pager *Pager) (string, error) {
	for {
		resp, err := client.ProcessCommonRequest(request)
		if err != nil {
			return "", fmt.Errorf("call failed %s", err)
		}
		//
		// jmespath.Parser{}
		err = pager.FeedResponse(resp.GetHttpContentBytes())
		if err != nil {
			return "", fmt.Errorf("call failed %s", err)
		}

		if !pager.HasMore() {
			break
		}
		pager.MoveNextPage(request)
	}

	return pager.GetResponseCollection(), nil
}