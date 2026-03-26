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
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func enterSafetyPolicySub(t *testing.T, ctx *cli.Context, name string) *cli.Command {
	t.Helper()
	root := NewConfigureSafetyPolicyCommand()
	ctx.EnterCommand(root)
	sub := root.GetSubCommand(name)
	require.NotNil(t, sub, "subcommand %q", name)
	ctx.EnterCommand(sub)
	return sub
}

func TestConfigureSafetyPolicy_Show_Default(t *testing.T) {
	dir := t.TempDir()
	ctx, w := testAiModeContext(t, dir)
	sub := enterSafetyPolicySub(t, ctx, "show")
	require.NoError(t, sub.Run(ctx, []string{}))
	var p safety.Policy
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(w.String())), &p))
	assert.False(t, p.Enabled)
	assert.Empty(t, p.Rules)
}

func TestConfigureSafetyPolicy_ParentRun_DefaultShow(t *testing.T) {
	dir := t.TempDir()
	ctx, w := testAiModeContext(t, dir)
	root := NewConfigureSafetyPolicyCommand()
	ctx.EnterCommand(root)
	require.NoError(t, root.Run(ctx, []string{}))
	var p safety.Policy
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(w.String())), &p))
	assert.False(t, p.Enabled)
}

func TestConfigureSafetyPolicy_EnableDisable(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	enable := enterSafetyPolicySub(t, ctx, "enable")
	require.NoError(t, enable.Run(ctx, []string{}))
	p, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	assert.True(t, p.Enabled)

	ctx2, _ := testAiModeContext(t, dir)
	disable := enterSafetyPolicySub(t, ctx2, "disable")
	require.NoError(t, disable.Run(ctx2, []string{}))
	p2, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	assert.False(t, p2.Enabled)
}

func TestConfigureSafetyPolicy_AddRemoveList(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	pat := ctx.Flags().Get("pattern")
	act := ctx.Flags().Get("action")
	require.NotNil(t, pat)
	require.NotNil(t, act)
	pat.SetAssigned(true)
	pat.SetValue("ecs:Delete*")
	act.SetAssigned(true)
	act.SetValue(string(safety.ActionDeny))
	require.NoError(t, add.Run(ctx, []string{}))

	p, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	require.Len(t, p.Rules, 1)
	assert.Equal(t, "ecs:Delete*", p.Rules[0].Pattern)
	assert.Equal(t, safety.ActionDeny, p.Rules[0].Action)

	ctx2, w2 := testAiModeContext(t, dir)
	list := enterSafetyPolicySub(t, ctx2, "list")
	require.NoError(t, list.Run(ctx2, []string{}))
	out := w2.String()
	assert.Contains(t, out, "Safety policy:")
	assert.Contains(t, out, "Config file:")
	assert.Contains(t, out, "ecs:Delete*")
	assert.Contains(t, out, string(safety.ActionDeny))

	ctx3, _ := testAiModeContext(t, dir)
	rm := enterSafetyPolicySub(t, ctx3, "remove")
	pat3 := ctx3.Flags().Get("pattern")
	pat3.SetAssigned(true)
	pat3.SetValue("ecs:Delete*")
	require.NoError(t, rm.Run(ctx3, []string{}))
	p3, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	assert.Empty(t, p3.Rules)
}

func TestConfigureSafetyPolicy_Add_UpdatesExistingPattern(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, safety.SavePolicy(dir, &safety.Policy{
		Enabled: true,
		Rules: []safety.Rule{{Pattern: "x:Y", Action: safety.ActionDeny}},
	}))

	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	ctx.Flags().Get("pattern").SetAssigned(true)
	ctx.Flags().Get("pattern").SetValue("x:Y")
	ctx.Flags().Get("action").SetAssigned(true)
	ctx.Flags().Get("action").SetValue(string(safety.ActionConfirm))
	require.NoError(t, add.Run(ctx, []string{}))

	p, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	require.Len(t, p.Rules, 1)
	assert.Equal(t, safety.ActionConfirm, p.Rules[0].Action)
}

func TestConfigureSafetyPolicy_Add_ForbidAction(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	ctx.Flags().Get("pattern").SetAssigned(true)
	ctx.Flags().Get("pattern").SetValue("*:Delete*")
	ctx.Flags().Get("action").SetAssigned(true)
	ctx.Flags().Get("action").SetValue(string(safety.ActionForbid))
	require.NoError(t, add.Run(ctx, []string{}))
	p, err := safety.LoadPolicy(dir)
	require.NoError(t, err)
	require.Len(t, p.Rules, 1)
	assert.Equal(t, safety.ActionForbid, p.Rules[0].Action)
}

func TestConfigureSafetyPolicy_Add_MissingPattern(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	ctx.Flags().Get("action").SetAssigned(true)
	ctx.Flags().Get("action").SetValue(string(safety.ActionDeny))
	err := add.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--pattern is required")
}

func TestConfigureSafetyPolicy_Add_MissingAction(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	ctx.Flags().Get("pattern").SetAssigned(true)
	ctx.Flags().Get("pattern").SetValue("a:b")
	err := add.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--action is required")
}

func TestConfigureSafetyPolicy_Add_InvalidAction(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	add := enterSafetyPolicySub(t, ctx, "add")
	ctx.Flags().Get("pattern").SetAssigned(true)
	ctx.Flags().Get("pattern").SetValue("a:b")
	ctx.Flags().Get("action").SetAssigned(true)
	ctx.Flags().Get("action").SetValue("nope")
	err := add.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "action must be deny, confirm, or forbid")
}

func TestConfigureSafetyPolicy_Remove_MissingPattern(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	rm := enterSafetyPolicySub(t, ctx, "remove")
	err := rm.Run(ctx, []string{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "--pattern is required")
}

func TestConfigureSafetyPolicy_Subcommand_ExtraArgs(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	sub := enterSafetyPolicySub(t, ctx, "show")
	err := sub.Run(ctx, []string{"extra"})
	require.Error(t, err)
}

func TestConfigureSafetyPolicy_ParentRun_ExtraArgs(t *testing.T) {
	dir := t.TempDir()
	ctx, _ := testAiModeContext(t, dir)
	root := NewConfigureSafetyPolicyCommand()
	ctx.EnterCommand(root)
	err := root.Run(ctx, []string{"nope"})
	require.Error(t, err)
}

func TestConfigureSafetyPolicy_List_NoRulesMessage(t *testing.T) {
	dir := t.TempDir()
	ctx, w := testAiModeContext(t, dir)
	list := enterSafetyPolicySub(t, ctx, "list")
	require.NoError(t, list.Run(ctx, []string{}))
	assert.Contains(t, w.String(), "Safety policy:")
	// Empty rules: either English or Chinese "no rules" message
	assert.True(t,
		strings.Contains(w.String(), "No rules") || strings.Contains(w.String(), "未配置"),
		"output: %s", w.String())
}
