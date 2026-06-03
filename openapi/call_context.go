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

	headerSourceIP        = "x-acs-source-ip"
	headerSecureTransport = "x-acs-secure-transport"
	queryKeySourceIP      = "SourceIp"
	queryKeySecureTransport = "SecureTransport"
)


func callContextEnv() (sourceIP string, secureTransport string) {
	sourceIP = strings.TrimSpace(os.Getenv(EnvSourceIP))
	secureTransport = strings.TrimSpace(os.Getenv(EnvSecureTransport))
	return
}

func applyCallContextRPC(queryParams map[string]string) {
	if queryParams == nil {
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

func applyCallContextROA(headers map[string]string) {
	if headers == nil {
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

func applyCallContextTeaHeaders(headers map[string]*string) {
	if headers == nil {
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
