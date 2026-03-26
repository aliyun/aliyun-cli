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
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

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
	dir := t.TempDir()
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs(dir, m)
	want, err := filepath.Abs(filepath.Join(dir, SafetyPolicyFileName))
	assert.NoError(t, err)
	assert.Equal(t, want, m[EnvSafetyPolicyFile])
}

func TestMergeSafetyPolicyPathIntoEnvs_nilOrEmpty(t *testing.T) {
	MergeSafetyPolicyPathIntoEnvs("", map[string]string{"a": "b"}) // no panic
	m := map[string]string{}
	MergeSafetyPolicyPathIntoEnvs("/tmp", nil) // no panic
	assert.Empty(t, m)
}
