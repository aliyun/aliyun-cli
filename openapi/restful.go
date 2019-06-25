// Copyright 1999-2019 Alibaba Group Holding Limited
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package openapi

import (
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/cli"
)

type RestfulInvoker struct {
	*BasicInvoker
	method string
	path   string
	force  bool
}

func (a *RestfulInvoker) Prepare(ctx *cli.Context) error {
	a.request.Headers["Date"] = time.Now().Format(time.RFC1123Z)
	a.request.PathPattern = a.path
	a.request.Method = a.method

	if a.request.RegionId != "" {
		a.request.Headers["x-acs-region-id"] = a.request.RegionId
	}

	if v, ok := BodyFlag(ctx.Flags()).GetValue(); ok {
		a.request.SetContent([]byte(v))
	}

	if v, ok := BodyFileFlag(ctx.Flags()).GetValue(); ok {
		buf, err := ioutil.ReadFile(v)
		if err != nil {
			// fmt.Errorf("failed read file: %s %v", v, err)
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

	// assign parameters
	for _, f := range ctx.UnknownFlags().Flags() {
		a.request.QueryParams[f.Name], _ = f.GetValue()
	}

	if _, ok := SecureFlag(ctx.Flags()).GetValue(); ok {
		a.request.Scheme = "https"
	}

	return nil
}

func (a *RestfulInvoker) Call() (*responses.CommonResponse, error) {
	resp, err := a.client.ProcessCommonRequest(a.request)
	return resp, err
}

func checkRestfulMethod(ctx *cli.Context, methodOrPath string, pathPattern string) (ok bool, method string, path string, err error) {
	if pathPattern == "" {
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
	} else if method, ok = RoaFlag(ctx.Flags()).GetValue(); ok {
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
