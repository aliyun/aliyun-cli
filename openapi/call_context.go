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
	"os"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
)

// 调用方上下文相关参数（source-ip / secure-transport）的注入键。
//   - ROA：通过 header 注入 x-acs-source-ip / x-acs-secure-transport
//   - RPC：通过 query 注入 SourceIp / SecureTransport
const (
	// EnvSourceIP 调用方源 IP。仅在为非空字符串时注入。
	EnvSourceIP = "ALIBABA_CLOUD_SOURCE_IP"
	// EnvSecureTransport 调用方是否走安全传输。仅在为非空字符串时注入；值原样透传，由网关自行解析。
	EnvSecureTransport = "ALIBABA_CLOUD_SECURE_TRANSPORT"
	// EnvCallContextSkipProducts 自建网关产品扩展跳过列表（逗号分隔，大小写不敏感，附加到默认列表）。
	EnvCallContextSkipProducts = "ALIBABA_CLOUD_CALL_CONTEXT_SKIP_PRODUCTS"

	headerSourceIP          = "x-acs-source-ip"
	headerSecureTransport   = "x-acs-secure-transport"
	queryKeySourceIP        = "SourceIp"
	queryKeySecureTransport = "SecureTransport"
)

// 已知使用自建网关 / 专属签名规则的产品
var selfBuiltGatewayProducts = map[string]struct{}{
	"sls": {},
	"pds": {},
}

func shouldSkipCallContext(productCode string) bool {
	code := strings.ToLower(strings.TrimSpace(productCode))
	if code == "" {
		return false
	}
	if _, ok := selfBuiltGatewayProducts[code]; ok {
		return true
	}
	if extra := strings.TrimSpace(os.Getenv(EnvCallContextSkipProducts)); extra != "" {
		for _, p := range strings.Split(extra, ",") {
			if strings.ToLower(strings.TrimSpace(p)) == code {
				return true
			}
		}
	}
	return false
}

func callContextEnv() (sourceIP string, secureTransport string) {
	sourceIP = strings.TrimSpace(os.Getenv(EnvSourceIP))
	secureTransport = strings.TrimSpace(os.Getenv(EnvSecureTransport))
	return
}

func applyCallContextRPC(productCode string, queryParams map[string]string) {
	if queryParams == nil || shouldSkipCallContext(productCode) {
		return
	}
	sourceIP, secureTransport := callContextEnv()
	if sourceIP != "" {
		if _, exists := queryParams[queryKeySourceIP]; !exists {
			queryParams[queryKeySourceIP] = sourceIP
		}
	}
	if secureTransport != "" {
		if _, exists := queryParams[queryKeySecureTransport]; !exists {
			queryParams[queryKeySecureTransport] = secureTransport
		}
	}
}

func applyCallContextROA(productCode string, headers map[string]string) {
	if headers == nil || shouldSkipCallContext(productCode) {
		return
	}
	sourceIP, secureTransport := callContextEnv()
	if sourceIP != "" {
		if _, exists := headers[headerSourceIP]; !exists {
			headers[headerSourceIP] = sourceIP
		}
	}
	if secureTransport != "" {
		if _, exists := headers[headerSecureTransport]; !exists {
			headers[headerSecureTransport] = secureTransport
		}
	}
}

// 用于 darabonba-openapi 风格请求（OpenapiContext / HttpContext）。
// 该路径下 headers 为 map[string]*string，单独提供一个版本以避免重复装箱逻辑泄露给调用方。
// 注意：当前主仓走该路径的产品（sls）本身就在跳过列表里。这里做产品判断只是为了让 helper 语义保持一致，便于后续接入新的 darabonba 产品时不踩坑。
func applyCallContextTeaHeaders(productCode string, headers map[string]*string) {
	if headers == nil || shouldSkipCallContext(productCode) {
		return
	}
	sourceIP, secureTransport := callContextEnv()
	if sourceIP != "" {
		if _, exists := headers[headerSourceIP]; !exists {
			headers[headerSourceIP] = tea.String(sourceIP)
		}
	}
	if secureTransport != "" {
		if _, exists := headers[headerSecureTransport]; !exists {
			headers[headerSecureTransport] = tea.String(secureTransport)
		}
	}
}
