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
	"strconv"
	"strings"
)

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

type Policy struct {
	Enabled bool   `json:"enabled"`
	Rules   []Rule `json:"rules"`
}

func DefaultPolicy() *Policy {
	return &Policy{
		Enabled: false,
		Rules:   []Rule{},
	}
}

type CheckResult struct {
	Action  Action
	Matched bool
	Rule    *Rule
}

type CommandInfo struct {
	Product string // e.g., "ecs", "cs"
	// For RPC: ApiName like "DeleteInstance", "UpdateInstance"
	// For REST: HTTP method like "DELETE", "PUT", "POST"
	ApiOrMethod string
	// For REST only: path like "/clusters"
	Path string
}

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

const SafetyPolicyFileName = "safety-policy.json"

func GetPolicyFilePath(configDir string) string {
	return filepath.Join(configDir, SafetyPolicyFileName)
}

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

const EnvSafetyPolicyEnabled = "ALIBABA_CLOUD_SAFETY_POLICY_ENABLED"

// Comma-separated entries, each entry is pattern=action (first '=' separates pattern and action).
// Example: *:Delete*=deny,ecs:Update*=confirm
const EnvSafetyPolicyRules = "ALIBABA_CLOUD_SAFETY_POLICY_RULES"

func actionFromEnvToken(s string) (Action, bool) {
	a := Action(strings.ToLower(strings.TrimSpace(s)))
	switch a {
	case ActionAllow, ActionDeny, ActionConfirm, ActionForbid:
		return a, true
	default:
		return "", false
	}
}

func parseEnvRulesList(raw string) ([]Rule, bool) {
	var out []Rule
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		pattern, actionStr, found := strings.Cut(part, "=")
		if !found {
			continue
		}
		pattern = strings.TrimSpace(pattern)
		actionStr = strings.TrimSpace(actionStr)
		if pattern == "" || actionStr == "" {
			continue
		}
		act, ok := actionFromEnvToken(actionStr)
		if !ok {
			continue
		}
		out = append(out, Rule{Pattern: pattern, Action: act})
	}
	if len(out) == 0 {
		return nil, false
	}
	return out, true
}

func copyPolicyRules(rules []Rule) []Rule {
	if len(rules) == 0 {
		return []Rule{}
	}
	return append([]Rule(nil), rules...)
}

func serializeRulesForEnv(rules []Rule) string {
	if len(rules) == 0 {
		return ""
	}
	parts := make([]string, 0, len(rules))
	for _, r := range rules {
		pat := strings.TrimSpace(r.Pattern)
		act := strings.ToLower(strings.TrimSpace(string(r.Action)))
		if pat == "" || act == "" {
			continue
		}
		parts = append(parts, pat+"="+act)
	}
	return strings.Join(parts, ",")
}

func MergePolicyFromEnv(base *Policy) *Policy {
	if base == nil {
		base = DefaultPolicy()
	}

	enabled := base.Enabled
	if v, ok := os.LookupEnv(EnvSafetyPolicyEnabled); ok {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			enabled = parsed
		}
	}

	var rules []Rule
	if raw0, ok := os.LookupEnv(EnvSafetyPolicyRules); ok {
		raw := strings.TrimSpace(raw0)
		if raw == "" {
			rules = []Rule{}
		} else if r, parsed := parseEnvRulesList(raw); parsed {
			rules = r
		} else {
			rules = copyPolicyRules(base.Rules)
		}
	} else {
		rules = copyPolicyRules(base.Rules)
	}

	return &Policy{Enabled: enabled, Rules: rules}
}

func LoadEffectivePolicy(configDir string) (*Policy, error) {
	p, err := LoadPolicy(configDir)
	if err != nil {
		return nil, err
	}
	return MergePolicyFromEnv(p), nil
}

const EnvSafetyPolicyFile = "ALIBABA_CLOUD_CLI_SAFETY_POLICY_FILE"

func MergeSafetyPolicyPathIntoEnvs(configDir string, envs map[string]string) {
	if envs == nil || configDir == "" {
		return
	}
	p := GetPolicyFilePath(configDir)
	if abs, err := filepath.Abs(p); err == nil {
		p = abs
	}
	envs[EnvSafetyPolicyFile] = p

	ef, err := LoadEffectivePolicy(configDir)
	if err != nil {
		return
	}
	envs[EnvSafetyPolicyEnabled] = strconv.FormatBool(ef.Enabled)
	envs[EnvSafetyPolicyRules] = serializeRulesForEnv(ef.Rules)
}

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
