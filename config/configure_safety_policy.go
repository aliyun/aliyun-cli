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

package config

import (
	"encoding/json"
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/safety"
)

func NewConfigureSafetyPolicyCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "safety-policy",
		Short: i18n.T(
			"manage safety policy and human-in-the-loop rules",
			"管理安全策略和人工确认规则"),
		Usage: "safety-policy [show|enable|disable|add|remove|list]",
		Long: i18n.T(
			`Configure safety policy to deny or require confirmation for destructive operations.

Commands:
  show      - Display current safety policy (default)
  enable    - Enable safety policy
  disable   - Disable safety policy
  add       - Add a rule: --pattern "product:ApiName" --action [deny|confirm]
  remove    - Remove a rule by pattern: --pattern "product:ApiName"
  list      - List all rules

Pattern format (supports * wildcard):
  *:Delete*       - Match all delete operations (built-in API + plugins)
  ecs:Delete*     - Match delete operations on ECS
  fc:delete-*     - Match plugin fc delete commands (e.g. fc delete-function)
  *:DELETE        - Match REST DELETE HTTP method

Actions:
  deny    - Block the operation completely
  confirm - Require human confirmation (forbid auto execution)`,
			`配置安全策略，用于拒绝或要求确认破坏性操作。

命令:
  show      - 显示当前安全策略（默认）
  enable    - 启用安全策略
  disable   - 禁用安全策略
  add       - 添加规则: --pattern "product:ApiName" --action [deny|confirm]
  remove    - 按模式删除规则: --pattern "product:ApiName"
  list      - 列出所有规则

模式格式（支持 * 通配符）:
  *:Delete*       - 匹配所有删除操作（内置 API + 插件）
  ecs:Delete*     - 匹配 ECS 的删除操作
  fc:delete-*     - 匹配插件 fc 的 delete 命令
  *:DELETE        - 匹配 REST DELETE HTTP 方法

动作:
  deny    - 完全阻止操作
  confirm - 需要人工确认（禁止自动执行）`),
		Run: func(ctx *cli.Context, args []string) error {
			return doConfigureSafetyPolicy(ctx, args)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Category:     "safety",
		Name:         "pattern",
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"command pattern for rule (e.g. *:Delete* or ecs:UpdateInstance)",
			"规则的命令模式 (如 *:Delete* 或 ecs:UpdateInstance)"),
	})
	cmd.Flags().Add(&cli.Flag{
		Category:     "safety",
		Name:         "action",
		AssignedMode: cli.AssignedOnce,
		Short: i18n.T(
			"action for rule: deny or confirm",
			"规则动作: deny 或 confirm"),
	})

	AddFlags(cmd.Flags())
	return cmd
}

func doConfigureSafetyPolicy(ctx *cli.Context, args []string) error {
	configDir := GetConfigDir(ctx)
	policy, err := safety.LoadPolicy(configDir)
	if err != nil {
		return fmt.Errorf("load safety policy failed: %w", err)
	}

	subcmd := "show"
	if len(args) > 0 {
		subcmd = args[0]
	}

	switch subcmd {
	case "show":
		return doSafetyPolicyShow(ctx, configDir, policy)
	case "enable":
		return doSafetyPolicyEnable(ctx, configDir, policy)
	case "disable":
		return doSafetyPolicyDisable(ctx, configDir, policy)
	case "add":
		return doSafetyPolicyAdd(ctx, configDir, policy)
	case "remove":
		return doSafetyPolicyRemove(ctx, configDir, policy)
	case "list":
		return doSafetyPolicyList(ctx, configDir, policy)
	default:
		return fmt.Errorf("unknown subcommand: %s. Use show, enable, disable, add, remove, or list", subcmd)
	}
}

func doSafetyPolicyShow(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	data, err := json.MarshalIndent(policy, "", "  ")
	if err != nil {
		return err
	}
	cli.Println(ctx.Stdout(), string(data))
	return nil
}

func doSafetyPolicyEnable(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	policy.Enabled = true
	return safety.SavePolicy(configDir, policy)
}

func doSafetyPolicyDisable(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	policy.Enabled = false
	return safety.SavePolicy(configDir, policy)
}

func doSafetyPolicyAdd(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	pattern, ok := ctx.Flags().Get("pattern").GetValue()
	if !ok || pattern == "" {
		return fmt.Errorf("--pattern is required for add")
	}
	actionStr, ok := ctx.Flags().Get("action").GetValue()
	if !ok || actionStr == "" {
		return fmt.Errorf("--action is required for add (deny or confirm)")
	}

	action := safety.Action(actionStr)
	if action != safety.ActionDeny && action != safety.ActionConfirm && action != safety.ActionForbid {
		return fmt.Errorf("action must be deny, confirm, or forbid")
	}

	// Check if rule already exists, update it
	for i := range policy.Rules {
		if policy.Rules[i].Pattern == pattern {
			policy.Rules[i].Action = action
			return safety.SavePolicy(configDir, policy)
		}
	}

	policy.Rules = append(policy.Rules, safety.Rule{
		Pattern: pattern,
		Action:  action,
	})
	return safety.SavePolicy(configDir, policy)
}

func doSafetyPolicyRemove(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	pattern, ok := ctx.Flags().Get("pattern").GetValue()
	if !ok || pattern == "" {
		return fmt.Errorf("--pattern is required for remove")
	}

	newRules := make([]safety.Rule, 0)
	for _, r := range policy.Rules {
		if r.Pattern != pattern {
			newRules = append(newRules, r)
		}
	}
	policy.Rules = newRules
	return safety.SavePolicy(configDir, policy)
}

func doSafetyPolicyList(ctx *cli.Context, configDir string, policy *safety.Policy) error {
	w := ctx.Stdout()
	cli.Printf(w, "Safety policy: %s\n", enabledStatus(policy.Enabled))
	cli.Printf(w, "Config file: %s\n", safety.GetPolicyFilePath(configDir))
	cli.Println(w, "")
	if len(policy.Rules) == 0 {
		cli.Println(w, i18n.T("No rules configured.", "未配置规则。").GetMessage())
		return nil
	}
	cli.Println(w, "Rules:")
	for i, r := range policy.Rules {
		cli.Printf(w, "  %d. %s -> %s\n", i+1, r.Pattern, r.Action)
	}
	return nil
}

func enabledStatus(enabled bool) string {
	if enabled {
		return i18n.T("enabled", "已启用").GetMessage()
	}
	return i18n.T("disabled", "已禁用").GetMessage()
}
