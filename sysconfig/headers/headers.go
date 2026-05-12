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

// Package headers 负责把父 CLI 这一侧"自动注入"的 HTTP header 汇总成单一编码格式：base64( json.Marshal( map[string]string ) )
package headers

import (
	"encoding/base64"
	"encoding/json"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/otel"
)

// EnvPluginHeaders 是父 CLI 透传给插件进程的 env key。值为 base64( json.Marshal( map[string]string ) )。
const EnvPluginHeaders = "ALIBABA_CLOUD_HEADERS"

// Collect 汇总父 CLI 这一侧所有"自动注入"的 header。
func Collect() map[string]string {
	out := map[string]string{}
	for k, v := range otel.GetHeaders() {
		out[k] = v
	}
	return out
}

func MergeIntoPluginEnvs(envs map[string]string) {
	if envs == nil {
		return
	}
	h := Collect()
	if len(h) == 0 {
		return
	}
	b, err := json.Marshal(h)
	if err != nil {
		return
	}
	envs[EnvPluginHeaders] = base64.StdEncoding.EncodeToString(b)
}
