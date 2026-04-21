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

package safety

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func unsetEnvForTest(t *testing.T, key string) {
	t.Helper()
	prev, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, prev)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	_ = os.Unsetenv(key)
}

func TestPolicy_Check_Disabled(t *testing.T) {
	policy := &Policy{
		Enabled: false,
		Rules: []Rule{
			{Pattern: "*:Delete*", Action: ActionDeny},
		},
	}
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"}
	result := policy.Check(cmd)
	assert.False(t, result.Matched)
	assert.Equal(t, ActionAllow, result.Action)
}

func TestPolicy_Check_Deny(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "*:Delete*", Action: ActionDeny},
		},
	}
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DeleteInstance"}
	result := policy.Check(cmd)
	assert.True(t, result.Matched)
	assert.Equal(t, ActionDeny, result.Action)
	assert.Equal(t, "*:Delete*", result.Rule.Pattern)
}

func TestPolicy_Check_Confirm(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "ecs:Update*", Action: ActionConfirm},
		},
	}
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "UpdateInstance"}
	result := policy.Check(cmd)
	assert.True(t, result.Matched)
	assert.Equal(t, ActionConfirm, result.Action)
}

func TestPolicy_Check_REST_DELETE(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "*:DELETE*", Action: ActionDeny},
		},
	}
	cmd := CommandInfo{Product: "cs", ApiOrMethod: "DELETE", Path: "/clusters"}
	result := policy.Check(cmd)
	assert.True(t, result.Matched)
	assert.Equal(t, ActionDeny, result.Action)
}

func TestPolicy_Check_Allow(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "*:Delete*", Action: ActionDeny},
		},
	}
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "DescribeInstances"}
	result := policy.Check(cmd)
	assert.False(t, result.Matched)
	assert.Equal(t, ActionAllow, result.Action)
}

func TestPolicy_Check_ForbidAlias(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "ecs:Update*", Action: ActionForbid},
		},
	}
	cmd := CommandInfo{Product: "ecs", ApiOrMethod: "UpdateInstance"}
	result := policy.Check(cmd)
	assert.True(t, result.Matched)
	assert.Equal(t, ActionConfirm, result.Action) // Forbid becomes Confirm
}

func TestPolicy_Check_Plugin(t *testing.T) {
	policy := &Policy{
		Enabled: true,
		Rules: []Rule{
			{Pattern: "*:delete*", Action: ActionDeny},
		},
	}
	cmd := CommandInfo{Product: "fc", ApiOrMethod: "delete-function"}
	result := policy.Check(cmd)
	assert.True(t, result.Matched)
	assert.Equal(t, ActionDeny, result.Action)
}

func TestMatchPattern(t *testing.T) {
	tests := []struct {
		pattern string
		cmd    string
		want   bool
	}{
		{"*:Delete*", "ecs:DeleteInstance", true},
		{"*:Delete*", "ecs:deleteinstance", true}, // case insensitive
		{"*:delete*", "fc:delete-function", true}, // plugin command
		{"ecs:Delete*", "ecs:DeleteInstance", true},
		{"ecs:Delete*", "cs:DeleteCluster", false},
		{"*:DELETE*", "cs:DELETE/clusters", true},
		{"*:Update*", "ecs:UpdateInstance", true},
	}
	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.cmd, func(t *testing.T) {
			got := matchPattern(tt.pattern, tt.cmd)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMergeSafetyPolicyPathIntoEnvs(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	unsetEnvForTest(t, EnvSafetyPolicyRules)
	dir := t.TempDir()
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs(dir, m)
	want, err := filepath.Abs(filepath.Join(dir, SafetyPolicyFileName))
	assert.NoError(t, err)
	assert.Equal(t, want, m[EnvSafetyPolicyFile])
	assert.Equal(t, "false", m[EnvSafetyPolicyEnabled])
	assert.Equal(t, "", m[EnvSafetyPolicyRules])
}

func TestMergeSafetyPolicyPathIntoEnvs_EffectiveFromFile(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	unsetEnvForTest(t, EnvSafetyPolicyRules)
	dir := t.TempDir()
	require.NoError(t, SavePolicy(dir, &Policy{
		Enabled: true,
		Rules:   []Rule{{Pattern: "ecs:Delete*", Action: ActionDeny}},
	}))
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs(dir, m)
	assert.Equal(t, "true", m[EnvSafetyPolicyEnabled])
	assert.Equal(t, "ecs:Delete*=deny", m[EnvSafetyPolicyRules])
}

func TestMergeSafetyPolicyPathIntoEnvs_EffectiveEnvOverridesFile(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyRules)
	dir := t.TempDir()
	require.NoError(t, SavePolicy(dir, &Policy{
		Enabled: false,
		Rules:   []Rule{{Pattern: "ecs:Delete*", Action: ActionDeny}},
	}))
	t.Setenv(EnvSafetyPolicyEnabled, "true")
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs(dir, m)
	assert.Equal(t, "true", m[EnvSafetyPolicyEnabled])
	assert.Equal(t, "ecs:Delete*=deny", m[EnvSafetyPolicyRules])
}

func TestMergeSafetyPolicyPathIntoEnvs_nilOrEmpty(t *testing.T) {
	MergeSafetyPolicyPathIntoEnvs("", map[string]string{"a": "b"}) // no panic
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs("/tmp", nil) // no panic
	assert.Empty(t, m)
}

func TestMergePolicyFromEnv_NoEnv(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	unsetEnvForTest(t, EnvSafetyPolicyRules)

	base := &Policy{Enabled: true, Rules: []Rule{{Pattern: "ecs:Delete*", Action: ActionDeny}}}
	got := MergePolicyFromEnv(base)
	assert.True(t, got.Enabled)
	require.Len(t, got.Rules, 1)
	assert.Equal(t, "ecs:Delete*", got.Rules[0].Pattern)
}

func TestMergePolicyFromEnv_EnabledOverride(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyRules)
	base := &Policy{Enabled: false, Rules: []Rule{{Pattern: "*:Delete*", Action: ActionDeny}}}

	t.Setenv(EnvSafetyPolicyEnabled, "true")
	got := MergePolicyFromEnv(base)
	assert.True(t, got.Enabled)
	assert.Len(t, got.Rules, 1)

	t.Setenv(EnvSafetyPolicyEnabled, "0")
	got2 := MergePolicyFromEnv(base)
	assert.False(t, got2.Enabled)
}

func TestMergePolicyFromEnv_RulesOverride(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	base := &Policy{Enabled: true, Rules: []Rule{{Pattern: "old:*", Action: ActionDeny}}}
	t.Setenv(EnvSafetyPolicyRules, "ecs:Update*=confirm")
	got := MergePolicyFromEnv(base)
	assert.True(t, got.Enabled)
	require.Len(t, got.Rules, 1)
	assert.Equal(t, "ecs:Update*", got.Rules[0].Pattern)
	assert.Equal(t, ActionConfirm, got.Rules[0].Action)
}

func TestMergePolicyFromEnv_RulesOverride_MultipleComma(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	base := &Policy{Enabled: true, Rules: []Rule{{Pattern: "old:*", Action: ActionDeny}}}
	t.Setenv(EnvSafetyPolicyRules, " *:Delete*=deny , ecs:Update* = confirm ")
	got := MergePolicyFromEnv(base)
	require.Len(t, got.Rules, 2)
	assert.Equal(t, "*:Delete*", got.Rules[0].Pattern)
	assert.Equal(t, ActionDeny, got.Rules[0].Action)
	assert.Equal(t, "ecs:Update*", got.Rules[1].Pattern)
	assert.Equal(t, ActionConfirm, got.Rules[1].Action)
}

func TestMergePolicyFromEnv_RulesEmptyStringClears(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	base := &Policy{Enabled: true, Rules: []Rule{{Pattern: "x", Action: ActionDeny}}}
	t.Setenv(EnvSafetyPolicyRules, "   ")
	got := MergePolicyFromEnv(base)
	assert.Empty(t, got.Rules)
}

func TestMergePolicyFromEnv_InvalidRulesEnvKeepsFile(t *testing.T) {
	unsetEnvForTest(t, EnvSafetyPolicyEnabled)
	base := &Policy{Enabled: true, Rules: []Rule{{Pattern: "keep:me", Action: ActionDeny}}}
	t.Setenv(EnvSafetyPolicyRules, "not-json,{badaction},noequals")
	got := MergePolicyFromEnv(base)
	require.Len(t, got.Rules, 1)
	assert.Equal(t, "keep:me", got.Rules[0].Pattern)
}

func TestLoadEffectivePolicy_AppliesEnv(t *testing.T) {
	dir := t.TempDir()
	path := GetPolicyFilePath(dir)
	filePolicy := Policy{Enabled: false, Rules: []Rule{{Pattern: "ecs:Describe*", Action: ActionDeny}}}
	data, err := json.MarshalIndent(filePolicy, "", "\t")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0600))

	t.Setenv(EnvSafetyPolicyEnabled, "true")
	t.Setenv(EnvSafetyPolicyRules, "ecs:Delete*=deny")

	got, err := LoadEffectivePolicy(dir)
	require.NoError(t, err)
	assert.True(t, got.Enabled)
	require.Len(t, got.Rules, 1)
	assert.Equal(t, "ecs:Delete*", got.Rules[0].Pattern)
}
