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
	"fmt"
	"strings"
)

// 对 manifest 声明的 alias 列表做规范化和去重：
//   - 每个 alias 单独 trim + 小写
//   - 空 alias、与主 command 相同、与已保留过的 alias 相同的都会被剔除
func sanitizeCommandAliases(command string, raw []string) []string {
	if len(raw) == 0 {
		return nil
	}
	cmdKey := normalizeCommandName(command)
	seen := make(map[string]struct{}, len(raw))
	out := make([]string, 0, len(raw))
	for _, a := range raw {
		key := normalizeCommandName(a)
		if key == "" || key == cmdKey {
			continue
		}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

//  1. Command 不能为空（新插件必须显式声明；已安装的历史插件不走此路径）
//  2. Command 不能是内置顶层命令
//  3. Command 不能出现在自身 alias 列表里（sanitize 已剔除，此处再校验一次防绕过）
//  4. 每个 alias 不能是内置命令
//  5. 该插件的任何 command/alias 不得与本地已装的“其他”插件的 command/alias 冲突
//  6. 与其他插件的 plugin name / short name 冲突时同样拒绝，避免歧义（如新插件 command="fc" 与已装的 aliyun-cli-fc 冲突）
//
// 允许同名插件覆盖自己（升级场景）：本地 plugin key == actualPluginName 时视为同一插件，跳过跨插件冲突。
func validatePluginCommandAndAliases(local *LocalManifest, actualPluginName, incomingCommand string, incomingAliases []string) error {
	cmdKey := normalizeCommandName(incomingCommand)
	if cmdKey == "" {
		return fmt.Errorf("plugin manifest: command is empty (a top-level CLI subcommand name is required)")
	}
	if IsReservedTopLevelCommand(cmdKey) {
		return fmt.Errorf("plugin manifest: command %q conflicts with a built-in top-level command", cmdKey)
	}

	// 逐个 alias 做规则 3/4 校验。sanitize 保证 alias 已小写且非空，再挡一道，防止绕过 sanitize 直接调用。
	seen := map[string]struct{}{cmdKey: {}}
	for _, a := range incomingAliases {
		key := normalizeCommandName(a)
		if key == "" {
			return fmt.Errorf("plugin manifest: alias is empty")
		}
		if key == cmdKey {
			return fmt.Errorf("plugin manifest: alias %q duplicates its command", key)
		}
		if _, dup := seen[key]; dup {
			return fmt.Errorf("plugin manifest: alias %q is duplicated", key)
		}
		if IsReservedTopLevelCommand(key) {
			return fmt.Errorf("plugin manifest: alias %q conflicts with a built-in top-level command", key)
		}
		seen[key] = struct{}{}
	}

	if local == nil || len(local.Plugins) == 0 {
		return nil
	}

	// 遍历本地其他插件（同名视为自身升级，跳过）；
	// 把它们的 command / aliases / 短名都汇入一张查询表进行冲突校验。
	actualPluginNameKey := strings.ToLower(strings.TrimSpace(actualPluginName))
	for name, p := range local.Plugins {
		if strings.ToLower(strings.TrimSpace(name)) == actualPluginNameKey {
			continue
		}
		otherKeys := collectPluginRoutingKeys(name, p)
		for key := range seen {
			if src, dup := otherKeys[key]; dup {
				return fmt.Errorf("plugin manifest: command/alias %q conflicts with installed plugin %q (%s)", key, name, src)
			}
		}
	}

	return nil
}

// command、每个 alias、plugin name 本身以及去掉 "aliyun-cli-" 前缀后的短名。
// value 是可读的来源描述，用于冲突错误信息。
func collectPluginRoutingKeys(name string, lp LocalPlugin) map[string]string {
	keys := make(map[string]string, 4)
	if cmd := normalizeCommandName(lp.Command); cmd != "" {
		keys[cmd] = "command"
	}
	for _, a := range lp.CommandAliases {
		if key := normalizeCommandName(a); key != "" {
			if _, exists := keys[key]; !exists {
				keys[key] = "alias"
			}
		}
	}
	if n := normalizeCommandName(name); n != "" {
		if _, exists := keys[n]; !exists {
			keys[n] = "plugin name"
		}
	}
	if short := normalizeCommandName(strings.TrimPrefix(strings.ToLower(strings.TrimSpace(name)), "aliyun-cli-")); short != "" && short != normalizeCommandName(name) {
		if _, exists := keys[short]; !exists {
			keys[short] = "plugin short name"
		}
	}
	return keys
}
