/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

type RpcInvoker struct {
	*BasicInvoker
	api *meta.Api
}

func (a *RpcInvoker) Prepare(ctx *cli.Context) error {
	// tidy names
	api := a.api
	request := a.request

	// assign api name, scheme method
	request.ApiName = api.Name
	request.Scheme = api.GetProtocol()
	request.Method = api.GetMethod()

	// assign parameters
	for _, f := range ctx.UnknownFlags().Flags() {
		param := api.FindParameter(f.Name)
		if param == nil {
			return &InvalidParameterError{Name: f.Name, api: api, flags: ctx.Flags()}
		}

		if param.Position == "Query" {
			request.QueryParams[f.Name], _ = f.GetValue()
		} else if param.Position == "Body" {
			request.FormParams[f.Name], _ = f.GetValue()
		} else {
			return fmt.Errorf("unknown parameter position; %s is %s", param.Name, param.Position)
		}
	}

	err := a.api.CheckRequiredParameters(func(s string) bool {
		switch s {
		case "RegionId":
			return request.RegionId != ""
		case "Action":
			return request.ApiName != ""
		default:
			f := ctx.UnknownFlags().Get(s)
			return f != nil && f.IsAssigned()
		}
	})

	if err != nil {
		return cli.NewErrorWithTip(err,
			"See %s or use `aliyun %s %s --help` to get more information",
			api.GetDocumentLink(), api.Product.GetLowerCode(), api.Name)
	}
	return nil
}

func (a *RpcInvoker) Call() (*responses.CommonResponse, error) {
	resp, err := a.client.ProcessCommonRequest(a.request)
	// fmt.Printf("Resp: %s", resp.String())
	return resp, err
}
//
//func (c *Caller) InvokeRpc(ctx *cli.Context, product *meta.Product, apiName string) error {
//	api, ok := c.library.GetApi(product.Code, product.Version, apiName)
//	if !ok {
//		return &InvalidApiError{Name: apiName, product: product}
//	}
//
//	//
//	// call OpenApi
//	// return: if check failed return error, otherwise return nil
//	client, request, err := c.InitClient(ctx, product, true)
//	if err != nil {
//		return err
//	}
//
//	request.ApiName = apiName
//	err = c.FillRpcParameters(ctx, request, &api)
//	if err != nil {
//		return err
//	}
//
//	request.Scheme = api.GetProtocol()
//	request.Method = api.GetMethod()
//
//	err = c.UpdateRequest(ctx, request)
//	if err != nil {
//		return err
//	}
//
//	if v, ok := PagerFlag.GetValue(); ok {
//		pager := GetPager(v)
//		r, err := c.InvokeRpcWithPager(client, request, pager)
//		if err != nil {
//			ctx.Command().PrintFailed(err, "")
//		} else {
//			filter, err := GetOutputFilter()
//			if err != nil {
//				ctx.Command().PrintFailed(err,"init output filter failed")
//				return nil
//			}
//
//			if filter != nil {
//				r, err = filter.FilterOutput(r)
//				if err != nil {
//					ctx.Command().PrintFailed(err,"output filter process failed")
//					return nil
//				}
//			}
//			fmt.Println(r)
//			if err != nil {
//				ctx.Command().PrintFailed(err, "")
//			}
//			return nil
//		}
//	}
//
//		// waiter := NewWaiterWithCTX(ctx, client, request)
//
//
//	filter, err := GetOutputFilter()
//	if err != nil {
//		ctx.Command().PrintFailed(err,"init output filter failed")
//		return nil
//	}
//
//	if filter != nil {
//		out, err = filter.FilterOutput(out)
//		if err != nil {
//			ctx.Command().PrintFailed(err,"output filter process failed")
//			return nil
//		}
//	}
//	fmt.Println(out)
//
//	//
//	return nil
//}
//
//func (c *Caller) InvokeRpcForce(ctx *cli.Context, product *meta.Product, apiName string) {
//	//
//	// call OpenApi
//	// return: if check failed return error, otherwise return nil
//	client, request, err := c.InitClient(ctx, product, true)
//
//	if err != nil {
//		ctx.Command().PrintFailed(fmt.Errorf("init client failed: %v", err), "")
//		return
//	}
//
//
//	err = c.UpdateRequest(ctx, request)
//	if err != nil {
//		ctx.Command().PrintFailed(err, "")
//		return
//	}
//
//	resp, err := client.ProcessCommonRequest(request)
//	if err != nil {
//		ctx.Command().PrintFailed(err, "")
//	}
//
//	// err = outputProcessor(ctx, resp.GetHttpContentString())
//	//if err != nil {
//	//	ctx.Command().PrintFailed(err, "")
//	//}
//
//	out := resp.GetHttpContentString()
//
//	filter, err := GetOutputFilter()
//
//	if err != nil {
//		ctx.Command().PrintFailed(err,"init output filter failed")
//		return
//	}
//
//	if filter != nil {
//		out, err = filter.FilterOutput(out)
//		if err != nil {
//			ctx.Command().PrintFailed(err,"output filter process failed")
//			return
//		}
//	}
//	fmt.Println(out)
//}
//
//func (c *Caller) FillRpcParameters(ctx *cli.Context, request *requests.CommonRequest, api *meta.Api) error {
//
//}
//
//func (c *Caller) InvokeRpcWithPager(client *sdk.Client, request *requests.CommonRequest, pager *Pager) (string, error) {
//	for {
//		resp, err := client.ProcessCommonRequest(request)
//		if err != nil {
//			return "", fmt.Errorf("call failed %s", err)
//		}
//		//
//		// jmespath.Parser{}
//		err = pager.FeedResponse(resp.GetHttpContentBytes())
//		if err != nil {
//			return "", fmt.Errorf("call failed %s", err)
//		}
//
//		if !pager.HasMore() {
//			break
//		}
//		pager.MoveNextPage(request)
//	}
//
//	return pager.GetResponseCollection(), nil
//}
