/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"io/ioutil"
	"strings"
	"time"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
)

type RestfulInvoker struct {
	*BasicInvoker
	method string
	path string
	force bool
}

func (a *RestfulInvoker) Prepare(ctx *cli.Context) error {
	a.request.Headers["Date"] = time.Now().Format(time.RFC1123Z)
	a.request.PathPattern = a.path
	a.request.Method = a.method

	if a.request.RegionId != "" {
		a.request.Headers["x-acs-region-id"] = a.request.RegionId
	}

	if v, ok := BodyFlag.GetValue(); ok {
		a.request.SetContent([]byte(v))
	}

	if v, ok := BodyFileFlag.GetValue(); ok {
		buf, err := ioutil.ReadFile(v)
		if err != nil {
			fmt.Errorf("failed read file: %s %v", v, err)
		}
		a.request.SetContent(buf)
	}

	if _, ok := a.request.Headers["Content-Type"]; !ok {
		content := string(a.request.Content)
		if strings.HasPrefix(content, "{") {
			a.request.SetContentType("application/json")
		} else if strings.HasPrefix(content, "<") {
			a.request.SetContentType("application/xml")
		}
	}

	if _, ok := SecureFlag.GetValue(); ok {
		a.request.Scheme = "https"
	}

	return nil
}

func (a *RestfulInvoker) Call() (*responses.CommonResponse, error) {
	resp, err := a.client.ProcessCommonRequest(a.request)
	return resp, err
}

func checkRestfulMethod(methodOrPath string, pathPattern string) (ok bool, method string, path string, err error) {
	if path == "" {
		ok = false
		return
	}
	if method, ok = checkHttpMethod(methodOrPath); ok {
		if strings.HasPrefix(pathPattern, "/") {
			path = pathPattern
			return
		} else {
			err = fmt.Errorf("bad restful path %s", pathPattern)
			return
		}
	} else if method, ok = RoaFlag.GetValue(); ok {
		if strings.HasPrefix(methodOrPath, "/") && pathPattern == "" {
			path = methodOrPath
			return
		} else {
			err = fmt.Errorf("bad restful path %s", methodOrPath)
			return
		}
	} else {
		ok = false
		return
	}
}

func checkHttpMethod(s string) (string, bool) {
	m := strings.ToUpper(s)
	if m == "GET" || m == "POST" || m == "PUT" || m == "DELETE" {
		return m, true
	}
	return "", false
}
