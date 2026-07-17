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
			`Configure safety policy to deny or require confirmation for destructive operations.`,
			`配置安全策略，用于拒绝或要求确认破坏性操作。`),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
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
			"command pattern for rule (e.g. *:Delete* or ecs:UpdateInstance)",
			"规则的命令模式 (如 *:Delete* 或 ecs:UpdateInstance)"),
	})
	cmd.Flags().Add(&cli.Flag{
		Category:     "safety",
		Name:         "action",
		AssignedMode: cli.AssignedOnce,
		Persistent:   true,
		Short: i18n.T(
			"action for rule: deny or confirm",
			"规则动作: deny 或 confirm"),
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

func newConfigureSafetyPolicyShowCommand() *cli.Command {
	return &cli.Command{
		Name:  "show",
		Usage: "show [--config-path <configPath>]",
		Short: i18n.T("display current safety policy", "显示当前安全策略"),
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			configDir, policy, err := loadSafetyPolicy(ctx)
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
			configDir, policy, err := loadSafetyPolicy(ctx)
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
