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

package telemetry

import "strings"

// Commands in this map accept positional business values. Limit their
// telemetry command to the declared depth so paths and resource names are
// never included in Command.
var fixedCommandDepthByToken = map[string]int{
	"oss":     2,
	"ossutil": 2,
}

func ExtractParamNames(args []string) []string {
	seen := make(map[string]struct{})
	var names []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		// Keep the original flag spelling, including leading "-" / "--".
		name := arg
		if eq := strings.Index(name, "="); eq >= 0 {
			name = name[:eq]
		}
		name = strings.TrimSpace(name)
		if strings.TrimLeft(name, "-") == "" {
			continue
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
		if !strings.Contains(arg, "=") && i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
		}
	}
	return names
}

func positionalParts(args []string) []string {
	// Plugin commands treat every non-flag token as a command segment.
	// Plugin business inputs must be passed as flag values, not positional values.
	var parts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			name := strings.TrimLeft(arg, "-")
			if eq := strings.Index(name, "="); eq >= 0 {
				continue
			}
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				i++
			}
			continue
		}
		parts = append(parts, arg)
	}
	return parts
}

func commandParts(args []string, commandToken string) []string {
	parts := positionalParts(args)
	depth, ok := fixedCommandDepthByToken[strings.ToLower(strings.TrimSpace(commandToken))]
	if !ok || len(parts) <= depth {
		return parts
	}
	return parts[:depth]
}

func normalizeCommand(parts []string) string {
	if len(parts) == 0 {
		return ""
	}
	// Join argv tokens with spaces so product codes and subcommands that
	// already contain "-" stay reversible, e.g. "ecs describe-instances".
	var tokens []string
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p == "" {
			continue
		}
		tokens = append(tokens, p)
	}
	return strings.Join(tokens, " ")
}
