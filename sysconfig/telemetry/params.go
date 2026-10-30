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

import (
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
)

// These commands accept paths after their command or operation name.
var fixedCommandDepthByToken = map[string]int{
	"oss":        2,
	"ossutil":    2,
	"go-migrate": 1,
}

func ExtractParamNames(args []string) []string {
	seen := make(map[string]struct{})
	var names []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break // Plugins may use a conventional end-of-options delimiter.
		}
		name, inline := telemetryFlagName(arg)
		if name == "" {
			continue
		}
		if _, ok := seen[name]; !ok {
			seen[name] = struct{}{}
			names = append(names, name)
		}
		if !inline && i+1 < len(args) {
			// Without the command's flag definitions, a short dash-leading
			// token can be a value. Prefer omitting it over exposing a value.
			nextName, _ := telemetryFlagName(args[i+1])
			if args[i+1] != "--" && !strings.HasPrefix(nextName, "--") {
				i++
			}
		}
	}
	return names
}

func telemetryFlagName(arg string) (string, bool) {
	name := arg
	inline := false
	if separator := strings.IndexAny(name, "=:"); separator >= 0 {
		// The CLI keeps waiting for a value after an empty "=" or ":".
		inline = separator+1 < len(name)
		name = name[:separator]
	}
	if strings.HasPrefix(name, "---") {
		return "", false // The CLI parser treats certificate-like tokens as values.
	}
	if strings.HasPrefix(name, "--") {
		if safeTelemetryIdentifier(name[2:], true) {
			return name, inline
		}
	} else if len(name) == 2 && name[0] == '-' && asciiLetter(name[1]) {
		return name, inline
	}
	return "", false
}

func positionalParts(args []string) []string {
	// Plugin/extension protocol uses non-flag tokens for command hierarchy and
	// passes business inputs as flag values. Both inline separators match CLI.
	var parts []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			break
		}
		if strings.HasPrefix(arg, "-") {
			if separator := strings.IndexAny(arg, "=:"); separator >= 0 && separator+1 < len(arg) {
				continue
			}
			// Empty inline assignments still consume the next independent
			// value. A following long flag starts another parameter instead.
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
				i++
			}
			continue
		}
		parts = append(parts, arg)
	}
	return parts
}

func commandParts(args []string, scope Scope) []string {
	command := scope.CommandToken
	if command == "" && len(args) > 0 {
		command = args[0]
	}
	if !safeTelemetryIdentifier(command, false) {
		return nil
	}
	parts := []string{command}
	if depth, ok := fixedCommandDepthByToken[strings.ToLower(command)]; ok {
		parts = positionalParts(args)
		if len(parts) > depth {
			parts = parts[:depth]
		}
		return parts
	}
	if scope.PluginName != "" || scope.PluginType != "" || isCLIExtension(command) {
		return positionalParts(args)
	}
	if len(args) == 0 || !strings.EqualFold(args[0], command) {
		return parts
	}
	command = strings.ToLower(command)
	// These core commands accept positional input after their subcommand:
	// plugin search takes a search term, configure get takes field names,
	// and utils go-migrate takes a source directory.
	coreParts := positionalParts(args)
	if len(coreParts) > 2 {
		switch command + " " + strings.ToLower(coreParts[1]) {
		case "plugin search", "configure get", "utils go-migrate":
			return coreParts[:2]
		}
	}
	if command == "help" {
		if len(args) > 1 && plugin.IsReservedTopLevelCommand(args[1]) {
			return append(parts, commandParts(args[1:], Scope{CommandToken: args[1]})...)
		}
		return parts
	}
	if plugin.IsReservedTopLevelCommand(command) {
		return coreParts
	}
	// OpenAPI retains the product and API name or HTTP method. ROA requests
	// can include a positional request path, which must not be collected.
	if len(coreParts) > 1 && safeTelemetryIdentifier(coreParts[1], false) {
		parts = append(parts, coreParts[1])
	}
	return parts
}

func safeTelemetryIdentifier(value string, allowDot bool) bool {
	if len(value) == 0 || len(value) > 128 || !asciiLetter(value[0]) {
		return false
	}
	for _, char := range value[1:] {
		if char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' || char == '-' || char == '_' || allowDot && char == '.' {
			continue
		}
		return false
	}
	return true
}

func asciiLetter(char byte) bool {
	return char >= 'a' && char <= 'z' || char >= 'A' && char <= 'Z'
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
