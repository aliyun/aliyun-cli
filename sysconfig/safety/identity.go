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

import "strings"

// WithAPIIdentities attaches authoritative APIName -> CmdName metadata for
// the command's built-in product and selected version. Callers must not use
// it for executable/metadata plugins or raw REST invocations. Missing names
// are never guessed. The immutable reverse index caches both spellings and
// marks ambiguous spellings unresolved, independent of map iteration order.
func WithAPIIdentities(cmd CommandInfo, aliases map[string]string) CommandInfo {
	cmd.identities = nil
	if cmd.Path != "" || !identityName(cmd.ApiOrMethod) {
		return cmd
	}
	identities := make(map[string]string, 2*len(aliases))
	for api, alias := range aliases {
		if !identityName(api) {
			continue
		}
		for _, name := range []string{api, alias} {
			if !identityName(name) {
				continue
			}
			key := strings.ToLower(name)
			if previous, found := identities[key]; found && previous != api {
				identities[key] = ""
			} else if !found {
				identities[key] = api
			}
		}
	}
	cmd.identities = identities
	return cmd
}

func identityName(name string) bool {
	return name != "" && strings.TrimSpace(name) == name && !strings.ContainsAny(name, "*:/ \t\r\n")
}

// Identity matching has higher specificity than syntactic matching. Equal
// specificity is resolved by declaration order by Policy.Check.
func matchRule(pattern string, cmd CommandInfo) (bool, int) {
	pattern = strings.TrimSpace(pattern)
	if cmd.Path == "" && !strings.Contains(pattern, "*") {
		product, name, ok := strings.Cut(pattern, ":")
		if ok && strings.EqualFold(product, cmd.Product) && identityName(name) {
			ruleID := cmd.identities[strings.ToLower(name)]
			if ruleID != "" {
				return ruleID == cmd.identities[strings.ToLower(cmd.ApiOrMethod)], 1
			}
		}
	}
	return matchPattern(pattern, buildCommandPattern(cmd)), 0
}
