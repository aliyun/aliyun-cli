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
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Action defines the policy action for a matched command
type Action string

const (
	// ActionAllow permits the operation (default when no rule matches)
	ActionAllow Action = "allow"
	// ActionDeny blocks the operation completely
	ActionDeny Action = "deny"
	// ActionConfirm requires human confirmation before proceeding (human-in-the-loop)
	// "forbid" is treated as alias for confirm - forbids automatic execution without approval
	ActionConfirm Action = "confirm"
	// ActionForbid is alias for ActionConfirm
	ActionForbid Action = "forbid"
)

// Rule defines a safety policy rule with command pattern and action
type Rule struct {
	// Pattern: "product:ApiName" or "product:METHOD" or "product:METHOD/path"
	// Supports wildcard: * matches any. Examples:
	//   "*:Delete*"   - deny all delete operations across products
	//   "ecs:Delete*" - deny delete on ECS
	//   "ecs:Update*" - require confirm for ECS update operations
	//   "*:DELETE"    - REST: deny all DELETE HTTP method
	Pattern string `json:"pattern"`
	// Action: allow, deny, confirm (or forbid)
	Action Action `json:"action"`
}

// Policy holds the safety policy configuration
type Policy struct {
	Enabled              bool   `json:"enabled"`
	Rules                []Rule `json:"rules"`
	PluginSpecialOSSUTIL any    `json:"ossutil,omitempty"`
}

// DefaultPolicy returns a policy with safety disabled (no restrictions)
func DefaultPolicy() *Policy {
	return &Policy{
		Enabled: false,
		Rules:   []Rule{},
	}
}

// CheckResult is the result of policy check
type CheckResult struct {
	Action  Action
	Matched bool
	Rule    *Rule
}

// CommandInfo describes the command being executed for policy matching
type CommandInfo struct {
	Product string // e.g., "ecs", "cs"
	// For RPC: ApiName like "DeleteInstance", "UpdateInstance"
	// For REST: HTTP method like "DELETE", "PUT", "POST"
	ApiOrMethod string
	// For REST only: path like "/clusters"
	Path string
}

// Check evaluates the policy against the given command and returns the applicable action
func (p *Policy) Check(cmd CommandInfo) CheckResult {
	if !p.Enabled || len(p.Rules) == 0 {
		return CheckResult{Action: ActionAllow, Matched: false}
	}

	// Build command identifier for matching
	// RPC: product:ApiName (e.g., ecs:DeleteInstance)
	// REST: product:METHOD or product:METHOD/path (e.g., cs:DELETE, cs:DELETE/clusters)
	cmdPattern := buildCommandPattern(cmd)

	// Rules are evaluated in order; first match wins
	for i := range p.Rules {
		rule := &p.Rules[i]
		if matchPattern(rule.Pattern, cmdPattern) {
			action := rule.Action
			if action == ActionForbid {
				action = ActionConfirm
			}
			if action == "" {
				action = ActionAllow
			}
			return CheckResult{
				Action:  action,
				Matched: true,
				Rule:    rule,
			}
		}
	}

	return CheckResult{Action: ActionAllow, Matched: false}
}

func buildCommandPattern(cmd CommandInfo) string {
	product := strings.ToLower(cmd.Product)
	if cmd.Path != "" {
		return fmt.Sprintf("%s:%s%s", product, strings.ToUpper(cmd.ApiOrMethod), cmd.Path)
	}
	// RPC: product:ApiName (preserve case for API names like DeleteInstance)
	return fmt.Sprintf("%s:%s", product, cmd.ApiOrMethod)
}

// matchPattern checks if the command matches the rule pattern
// Pattern supports * as wildcard. Pattern format: "product:apiOrMethod" or "product:apiOrMethod/path"
func matchPattern(pattern, cmd string) bool {
	if pattern == "" {
		return false
	}
	pattern = strings.TrimSpace(pattern)
	cmd = strings.TrimSpace(cmd)

	// Convert * to regex: * matches any sequence
	// Escape other regex special chars
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, "\\*", ".*")
	regexPattern = "(?i)^" + regexPattern + "$" // (?i) = case-insensitive

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}
	return re.MatchString(cmd)
}

// InferOperationFromApiName returns delete, update, create or "" from API name
func InferOperationFromApiName(apiName string) string {
	apiLower := strings.ToLower(apiName)
	if strings.HasPrefix(apiLower, "delete") {
		return "delete"
	}
	if strings.HasPrefix(apiLower, "update") || strings.HasPrefix(apiLower, "modify") {
		return "update"
	}
	if strings.HasPrefix(apiLower, "create") || strings.HasPrefix(apiLower, "add") {
		return "create"
	}
	return ""
}

// InferOperationFromHttpMethod maps HTTP method to operation type
func InferOperationFromHttpMethod(method string) string {
	switch strings.ToUpper(method) {
	case "DELETE":
		return "delete"
	case "PUT", "PATCH":
		return "update"
	case "POST":
		return "create" // or update, POST is ambiguous
	default:
		return ""
	}
}

// OssutilConfigPolicyOssutilKey is the JSON key under OSSUTIL_CONFIG_VALUE for Policy.PluginSpecialOSSUTIL
// (JSON field "ossutil" in safety-policy.json). Separate from profile "ossutil"; both may be present.
const OssutilConfigPolicyOssutilKey = "policy-ossutil"

// SafetyPolicyFileName is the name of the standalone safety policy file
const SafetyPolicyFileName = "safety-policy.json"

// GetPolicyFilePath returns the path to safety policy file in the given config directory.
// Policy is stored in a separate file (e.g. ~/.aliyun/safety-policy.json) as a global config.
func GetPolicyFilePath(configDir string) string {
	return filepath.Join(configDir, SafetyPolicyFileName)
}

// LoadPolicy loads the safety policy from the config directory.
// Returns DefaultPolicy() if file does not exist or is invalid.
func LoadPolicy(configDir string) (*Policy, error) {
	path := GetPolicyFilePath(configDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultPolicy(), nil
		}
		return nil, err
	}
	var p Policy
	if err := json.Unmarshal(data, &p); err != nil {
		return DefaultPolicy(), nil
	}
	if p.Rules == nil {
		p.Rules = []Rule{}
	}
	return &p, nil
}

// EnvSafetyPolicyFile names the environment variable set for subprocesses (plugins, ossutil)
// to the absolute path of safety-policy.json (same file as configure safety-policy).
const EnvSafetyPolicyFile = "ALIBABA_CLOUD_CLI_SAFETY_POLICY_FILE"

// MergeSafetyPolicyPathIntoEnvs sets EnvSafetyPolicyFile to the absolute path of the global
// safety policy file under configDir. Subprocesses may load and enforce policy themselves;
// the main CLI still enforces policy before plugin API execution when applicable.
func MergeSafetyPolicyPathIntoEnvs(configDir string, envs map[string]string) {
	if envs == nil || configDir == "" {
		return
	}
	p := GetPolicyFilePath(configDir)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	envs[EnvSafetyPolicyFile] = p
}

// SavePolicy writes the safety policy to the config directory.
func SavePolicy(configDir string, policy *Policy) error {
	if policy == nil {
		policy = DefaultPolicy()
	}
	path := GetPolicyFilePath(configDir)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(policy, "", "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0600)
}
