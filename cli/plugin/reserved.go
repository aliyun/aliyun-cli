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

package plugin

import (
	"sort"
	"strings"
)

// 内置顶层命令注册表。产品方 alias 与 plugin command 都不允许覆盖内置命令，
// 默认包含 CLI 自带且不会随build 变化的核心命令。
// 对于按需注册的 cliext 命令，`main` 在构建 rootCmd 之后会调用 RegisterReservedTopLevelCommands 追加进来，避免维护列表漂移。
// 并发契约：约定 RegisterReservedTopLevelCommands 只在进程启动阶段（rootCmd 构建期）调用；之后所有访问都视为只读。
// 如果未来有并发写入需求（例如懒加载 cliext / 异步插件发现），需要引入 Mutex
var reservedTopLevelCommands = map[string]struct{}{
	"aliyun":          {},
	"configure":       {},
	"version":         {},
	"auto-completion": {},
	"help":            {},
	"plugin":          {},
	"upgrade":         {},
	"oss":             {},
	"ossutil":         {},
	"mcp":             {},
	"mock":            {},
}

func RegisterReservedTopLevelCommands(names []string) {
	for _, n := range names {
		key := normalizeCommandName(n)
		if key == "" {
			continue
		}
		reservedTopLevelCommands[key] = struct{}{}
	}
}

func IsReservedTopLevelCommand(name string) bool {
	key := normalizeCommandName(name)
	if key == "" {
		return false
	}
	_, ok := reservedTopLevelCommands[key]
	return ok
}

func ReservedTopLevelCommands() []string {
	names := make([]string, 0, len(reservedTopLevelCommands))
	for n := range reservedTopLevelCommands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

func normalizeCommandName(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}
