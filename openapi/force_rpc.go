/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
)

type ForceRpcInvoker struct {
	*BasicInvoker
	method string
}

func (a *ForceRpcInvoker) Prepare(ctx *cli.Context) error {
	// assign api name
	a.request.ApiName = a.method

	// assign parameters
	for _, f := range ctx.UnknownFlags().Flags() {
		a.request.QueryParams[f.Name], _ = f.GetValue()
	}

	// --secure use https
	if _, ok := SecureFlag.GetValue(); ok {
		a.request.Scheme = "https"
	}
	return nil
}

func (a *ForceRpcInvoker) Call() (*responses.CommonResponse, error) {
	resp, err := a.client.ProcessCommonRequest(a.request)
	return resp, err
}
