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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	sysmock "github.com/aliyun/aliyun-cli/v3/sysconfig/mock"
)

// Read definitions without assigning flags or changing the command parser's
// state. Unknown flag forms leave the command boundary unconfirmed.
func stripLeadingTelemetryFlags(args []string, flags *cli.FlagSet) []string {
	if flags == nil {
		return sysmock.StripLeadingGlobalFlags(args)
	}
	for i := 0; i < len(args); {
		token := args[i]
		if !strings.HasPrefix(token, "-") || strings.HasPrefix(token, "---") {
			return args[i:]
		}
		flag := telemetryFlagDefinition(token, flags)
		if flag == nil {
			return nil
		}
		i++
		_, value, inline := cli.SplitStringWithPrefix(token, "=:")
		if flag.AssignedMode == cli.AssignedNone {
			if inline && value != "" {
				return nil
			}
			continue
		}
		if inline && value != "" {
			continue
		}
		for i < len(args) {
			next := args[i]
			if strings.HasPrefix(next, "--") && !strings.HasPrefix(next, "---") || telemetryFlagDefinition(next, flags) != nil {
				break
			}
			i++
			if flag.AssignedMode != cli.AssignedRepeatable {
				break
			}
		}
	}
	return nil
}

func telemetryFlagDefinition(token string, flags *cli.FlagSet) *cli.Flag {
	name, _, _ := cli.SplitStringWithPrefix(token, "=:")
	if strings.HasPrefix(name, "--") {
		return flags.Get(name[2:])
	}
	if len(name) == 2 && name[0] == '-' {
		return flags.GetByShorthand(rune(name[1]))
	}
	return nil
}

func executionParameterNames(execution *activeCommandExecution) []string {
	names := execution.parameterNames
	commandType := detectCommandTypeForScope(execution.rootArgs, execution.scope)
	// An installed Go plugin is only a candidate until dispatch selects it.
	// Native OpenAPI calls can fail parsing before that decision is reached.
	inferToolNames := !execution.runtimeParametersSet &&
		(execution.scope.CommandType == CommandTypePluginGo || commandType == CommandTypeCLIExt)
	if inferToolNames {
		names = inferToolParameterNames(execution.rootArgs)
	}
	if execution.context != nil {
		limit := -1
		if execution.runtimeParametersSet || inferToolNames {
			// The runtime snapshot or tool inference covers the product's tail.
			// Keep only already parsed host flags preceding it to avoid duplicates.
			limit = execution.commandArgOffset
		}
		names = append(execution.context.ParsedFlagNames(limit), names...)
	}
	return names
}

// inferToolParameterNames is best-effort observation for Go plugins and CLI
// extensions whose parameter definitions are unavailable to the host. It cannot
// distinguish every dash-leading business value from an option. Never use it
// as a fallback for commands with an authoritative parser or API definition.
func inferToolParameterNames(args []string) []string {
	var names []string
	for i := 0; i < len(args); i++ {
		if args[i] == "--" {
			break
		}
		name, inline := inferredToolFlagName(args[i])
		if name == "" {
			continue
		}
		names = append(names, name)
		if !inline && i+1 < len(args) {
			// Short dash-leading tokens can also be values. Prefer omitting
			// an ambiguous token immediately following a value-taking option.
			next, _ := inferredToolFlagName(args[i+1])
			if args[i+1] != "--" && !strings.HasPrefix(next, "--") {
				i++
			}
		}
	}
	return names
}

func inferredToolFlagName(token string) (string, bool) {
	name, value, separated := cli.SplitStringWithPrefix(token, "=:")
	inline := separated && value != ""
	if strings.HasPrefix(name, "---") {
		return "", false
	}
	if strings.HasPrefix(name, "--") && safeTelemetryIdentifier(name[2:], true) {
		return name, inline
	}
	if len(name) == 2 && name[0] == '-' && asciiLetter(name[1]) {
		return name, inline
	}
	return "", false
}

func positionalParts(args []string) []string {
	// Installed plugins pass business inputs as flag values. Extensions may
	// accept positional business inputs, so callers must bound their hierarchy.
	// Both inline separators match CLI.
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

func commandParts(args []string, scope Scope, commandPath []string) []string {
	command := scope.CommandToken
	if command == "" && len(args) > 0 {
		command = args[0]
	}
	if !safeTelemetryIdentifier(command, false) {
		return nil
	}
	parts := []string{command}
	switch strings.ToLower(command) {
	case "go-migrate":
		return parts
	case "oss", "ossutil":
		return extensionCommandParts(args, command)
	}
	// Extension argv can contain positional business inputs. Only retain paths
	// confirmed from tool help; no confirmed prefix means root-only fallback.
	if isCLIExtension(command) {
		return extensionCommandParts(args, command)
	}
	if scope.PluginName != "" || scope.PluginType != "" {
		if plugin.NormalizePluginType(scope.PluginType) == plugin.PluginTypeGo {
			return confirmedPluginCommandParts(args, command, scope.PluginCommandPaths)
		}
		if safeTelemetryIdentifier(scope.ResolvedCommand, false) {
			return append(parts, scope.ResolvedCommand)
		}
		return parts
	}
	if len(args) == 0 || !strings.EqualFold(args[0], command) {
		return parts
	}
	if plugin.IsReservedTopLevelCommand(command) {
		// A Run callback may accept or ignore extra positional input. Only
		// names actually selected from the registered tree are command levels.
		if strings.EqualFold(command, "help") {
			if len(commandPath) > 0 && plugin.IsReservedTopLevelCommand(commandPath[0]) {
				return append(parts, commandPath...)
			}
		} else if len(commandPath) > 0 && commandPath[0] == command {
			return commandPath
		}
		return parts
	}
	// OpenAPI retains the product and API name or HTTP method. ROA requests
	// can include a positional request path, which must not be collected.
	coreParts := positionalParts(args)
	if len(coreParts) > 1 && safeTelemetryIdentifier(coreParts[1], false) {
		parts = append(parts, coreParts[1])
	}
	return parts
}

func confirmedPluginCommandParts(args []string, root string, paths []string) []string {
	parts := []string{root}
	if len(args) == 0 || !strings.EqualFold(args[0], root) {
		return parts
	}
	// Go plugins accept business inputs through flags, but malformed invocations
	// can still contain arbitrary tokens before them. Only retain names confirmed
	// by the selected package's manifest, including groups implied by full paths.
	matched := 0
	for _, path := range paths {
		depth := 0
		for i, name := range strings.Fields(path) {
			if i+1 >= len(args) || strings.HasPrefix(args[i+1], "-") || !strings.EqualFold(args[i+1], name) {
				break
			}
			depth++
		}
		if depth > matched {
			matched = depth
		}
	}
	return append(parts, args[1:matched+1]...)
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
	// Join command tokens without changing their case or hyphens, preserving
	// the distinction between e.g. "ecs DescribeRegions" and "ecs describe-regions".
	var tokens []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		tokens = append(tokens, p)
	}
	return strings.Join(tokens, " ")
}
