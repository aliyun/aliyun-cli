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

// Package headers 负责把父 CLI 这一侧"自动注入"的 HTTP header 汇总成单一
// 透传字段（ALIBABA_CLOUD_HEADERS, JSON 编码），交给插件运行时（aliyun-cli-runtime）
// 在发起 OpenAPI 请求前 apply。
//
// 设计要点：
//   - 单一透传字段：避免每加一个 header 来源就新增一组 env 变量；
//   - JSON 编码：baggage 之类的值天然包含 ',' / '='，自定义分隔符必踩坑；
//   - 仅父进程 → 子进程私有协议：用户面向接口仍然是 ALIBABA_CLOUD_OTEL_* / --header
//     等，请勿手动设置本变量；
//   - 优先级：runtime 端 apply 时晚于 --header，会覆盖用户 --header（与父 CLI
//     直发请求时 otel.InjectHeaders 调用顺序保持一致）。
package headers

import (
	"encoding/json"

	"github.com/aliyun/aliyun-cli/v3/sysconfig/otel"
)

// EnvPluginHeaders 是父 CLI 透传给插件进程的 env key。值为 JSON 编码的 map[string]string。
const EnvPluginHeaders = "ALIBABA_CLOUD_HEADERS"

// Collect 汇总父 CLI 这一侧所有"自动注入"的 header。
// 后续要接入新的 header provider（如自定义统计、AB test 等）时，在这里 merge 即可，
func Collect() map[string]string {
	out := map[string]string{}
	for k, v := range otel.GetHeaders() {
		out[k] = v
	}
	return out
}

// 当无任何 header 需要透传时不写入；序列化失败时静默跳过（不阻断插件执行）。
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
	envs[EnvPluginHeaders] = string(b)
}
