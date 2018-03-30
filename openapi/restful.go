/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"io/ioutil"
	"strings"
	"time"
)

func (c *Caller) InvokeRestful(ctx *cli.Context, product *meta.Product, method string, path string) {
	client, request, err := c.InitClient(ctx, product, false)
	if err != nil {
		ctx.Command().PrintFailed(fmt.Errorf("bad client %v", err), "")
		return
	}

	request.Headers["Date"] = time.Now().Format(time.RFC1123Z)
	request.PathPattern = path
	request.Method = method

	if request.RegionId != "" {
		request.Headers["x-acs-region-id"] = request.RegionId
	}

	if v, ok := BodyFlag.GetValue(); ok {
		request.SetContent([]byte(v))
	}

	if v, ok := BodyFileFlag.GetValue(); ok {
		buf, err := ioutil.ReadFile(v)
		if err != nil {
			fmt.Errorf("failed read file: %s %v", v, err)
		}
		request.SetContent(buf)
	}

	if v, ok := AcceptFlag.GetValue(); ok {
		request.AcceptFormat = v
	}

	if _, ok := request.Headers["Content-Type"]; !ok {
		content := string(request.Content)
		if strings.HasPrefix(content, "{") {
			request.SetContentType("application/json")
		} else if strings.HasPrefix(content, "<") {
			request.SetContentType("application/xml")
		}
	}

	err = c.UpdateRequest(ctx, request)
	if err != nil {
		ctx.Command().PrintFailed(err, "")
		return
	}

	resp, err := client.ProcessCommonRequest(request)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unmarshal") {
			// fmt.Printf("%v\n", err)
		} else {
			ctx.Command().PrintFailed(err, "")
			return
		}
	}

	//err = outputProcessor(ctx, resp.GetHttpContentString())
	//if err != nil {
	//	ctx.Command().PrintFailed(err, "")
	//}
	fmt.Println(resp.GetHttpContentString())
}

func CheckRestfulMethod(ctx *cli.Context, methodOrPath string, pathPattern string) (ok bool, method string, path string, err error) {
	if method, ok = CheckHttpMethod(methodOrPath); ok {
		if strings.HasPrefix(pathPattern, "/") {
			path = pathPattern
			return
		} else {
			err = fmt.Errorf("bad restful path %s", pathPattern)
			return
		}
	//} else if method, ok = ctx.Flags().GetValue("roa"); ok {
	//	if strings.HasPrefix(methodOrPath, "/") && pathPattern == "" {
	//		path = methodOrPath
	//		return
	//	} else {
	//		err = fmt.Errorf("bad restful path %s", methodOrPath)
	//		return
	//	}
	} else {
		ok = false
		return
	}
}

func CheckHttpMethod(s string) (string, bool) {
	m := strings.ToUpper(s)
	if m == "GET" || m == "POST" || m == "PUT" || m == "DELETE" {
		return m, true
	}
	return "", false
}
