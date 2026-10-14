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
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
)

func NewConfigureSafetyPolicyCommand() *cli.Command {
	cmd := &cli.Command{
		Name: "safety-policy",
		Short: i18n.T(
			"manage safety policy and human-in-the-loop rules",
			"管理安全策略和人工确认规则"),
		Usage: "safety-policy [command] [--config-path <configPath>]",
		Long: i18n.T(
			`Configure safety policy to deny or require confirmation for destructive operations.
For built-in APIs recognized in the selected API version, rules without wildcards match both PascalCase and kebab-case command names.
Rules containing * and commands for plugins or REST requests are matched as entered, without case sensitivity. The same applies when an API name cannot be uniquely resolved.
When multiple rules match, deny takes precedence over confirm, followed by allow. For rules with the same action, an exact match to a recognized API takes precedence; remaining ties use configuration order.
Earlier versions applied the first matching rule. An allow rule can no longer override a matching deny or confirm rule, regardless of order. Commands that match no rule are allowed.`,
			`配置安全策略，用于拒绝或要求确认破坏性操作。
对于所选 API 版本中可识别的内置 API，不含通配符的规则同时匹配 PascalCase 和 kebab-case 两种命令名称。
含 * 的规则、插件命令和 REST 请求按输入的命令名称或路径匹配，不区分大小写。API 名称无法唯一识别时，也采用此方式。
多条规则匹配时，按拒绝（deny）、确认（confirm）、允许（allow）的优先级执行。规则操作相同时，优先选择精确匹配已识别 API 的规则；匹配优先级也相同时，选择配置顺序靠前的规则。
旧版本仅执行首条匹配规则。当前版本中，无论配置顺序如何，allow 规则均不能覆盖匹配的 deny 或 confirm 规则。未匹配任何规则的命令允许执行。`),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadEffectiveSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyShow(ctx, configDir, policy)
		},
	}

	cmd.Flags().Add(&cli.Flag{
		Category:     "safety",
		Name:         "pattern",
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"API or command matching pattern (case-insensitive; supports * wildcards)",
			"API 或命令匹配模式，不区分大小写，支持 * 通配符"),
	})
	cmd.Flags().Add(&cli.Flag{
		Category:     "safety",
		Name:         "action",
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"action for rule: deny, confirm (forbid), or allow",
			"规则操作：deny（拒绝）、confirm（确认，别名为 forbid）或 allow（允许）"),
	})

	AddFlags(cmd.Flags())

	cmd.AddSubCommand(newConfigureSafetyPolicyShowCommand())
	cmd.AddSubCommand(newConfigureSafetyPolicyEnableCommand())
	cmd.AddSubCommand(newConfigureSafetyPolicyDisableCommand())
	cmd.AddSubCommand(newConfigureSafetyPolicyAddCommand())
	cmd.AddSubCommand(newConfigureSafetyPolicyRemoveCommand())
	cmd.AddSubCommand(newConfigureSafetyPolicyListCommand())
	return cmd
}

func loadSafetyPolicy(ctx *cli.Context) (configDir string, policy *safety.Policy, err error) {
	configDir = GetConfigDir(ctx)
	policy, err = safety.LoadPolicy(configDir)
	if err != nil {
		return "", nil, fmt.Errorf("load safety policy failed: %w", err)
	}
	return configDir, policy, nil
}

func loadEffectiveSafetyPolicy(ctx *cli.Context) (configDir string, policy *safety.Policy, err error) {
	configDir = GetConfigDir(ctx)
	policy, err = safety.LoadEffectivePolicy(configDir)
	if err != nil {
		return "", nil, fmt.Errorf("load effective safety policy failed: %w", err)
	}
	return configDir, policy, nil
}

func newConfigureSafetyPolicyShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show [--config-path <configPath>]",
		Short: i18n.T("display current safety policy", "显示当前安全策略"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadEffectiveSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyShow(ctx, configDir, policy)
		},
	}
}

func newConfigureSafetyPolicyEnableCommand() *cli.Command {
	return &cli.Command{
		Name:  "enable",
		Usage: "enable [--config-path <configPath>]",
		Short: i18n.T("enable safety policy", "启用安全策略"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyEnable(ctx, configDir, policy)
		},
	}
}

func newConfigureSafetyPolicyDisableCommand() *cli.Command {
	return &cli.Command{
		Name:  "disable",
		Usage: "disable [--config-path <configPath>]",
		Short: i18n.T("disable safety policy", "禁用安全策略"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyDisable(ctx, configDir, policy)
		},
	}
}

func newConfigureSafetyPolicyAddCommand() *cli.Command {
	return &cli.Command{
		Name:  "add",
		Usage: "add --pattern <pattern> --action <deny|confirm|forbid> [--config-path <configPath>]",
		Short: i18n.T("add or update a safety rule", "添加或更新安全规则"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyAdd(ctx, configDir, policy)
		},
	}
}

func newConfigureSafetyPolicyRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:  "remove",
		Usage: "remove --pattern <pattern> [--config-path <configPath>]",
		Short: i18n.T("remove a safety rule by pattern", "按模式删除安全规则"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyRemove(ctx, configDir, policy)
		},
	}
}

func newConfigureSafetyPolicyListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list [--config-path <configPath>]",
		Short: i18n.T("list all safety rules", "列出所有安全规则"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadEffectiveSafetyPolicy(ctx)
			if err != nil {
				return err
			}
			return doSafetyPolicyList(ctx, configDir, policy)
		},
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
