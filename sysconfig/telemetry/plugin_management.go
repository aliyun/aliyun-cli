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
	"regexp"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
)

var safePluginNamePattern = regexp.MustCompile(`^[A-Za-z0-9._-]{1,128}$`)

type PluginManagementEvent struct {
	Operation   string   `json:"operation"`
	PluginNames []string `json:"plugin_names,omitempty"`
	FromVersion string   `json:"from_version,omitempty"`
	ToVersion   string   `json:"to_version,omitempty"`
	PluginType  string   `json:"plugin_type,omitempty"`
	Inner       *bool    `json:"inner,omitempty"`
	SourceType  string   `json:"source_type,omitempty"`
}

func DetectPluginManagement(args []string) *PluginManagementEvent {
	if len(args) < 2 || strings.ToLower(args[0]) != "plugin" {
		return nil
	}
	operation := strings.ToLower(strings.TrimSpace(args[1]))
	switch operation {
	case "install", "uninstall", "update":
	case "install-all", "install-custom":
		operation = strings.ReplaceAll(operation, "-", "_")
	default:
		return nil
	}

	event := &PluginManagementEvent{Operation: operation}
	event.PluginNames = pluginFlagValues(args[2:], "name", "names")
	if value := firstFlagValue(args[2:], "version"); value != "" {
		event.ToVersion = sanitizeVersion(value)
	}
	if hasFlag(args[2:], "package") {
		event.SourceType = "package"
	} else if operation != "uninstall" {
		event.SourceType = "catalog"
	}
	if len(event.PluginNames) == 1 {
		if local, err := plugin.LookupLocalPlugin(event.PluginNames[0]); err == nil && local != nil {
			event.PluginNames[0] = local.Name
			event.FromVersion = sanitizeVersion(local.Version)
			event.PluginType = plugin.NormalizePluginType(local.Type)
			inner := local.Inner
			event.Inner = &inner
		}
	}
	return event
}

func CompletePluginManagement(event *PluginManagementEvent) {
	if event == nil || len(event.PluginNames) != 1 || event.Operation == "uninstall" {
		return
	}
	if local, err := plugin.LookupLocalPlugin(event.PluginNames[0]); err == nil && local != nil {
		event.PluginNames[0] = local.Name
		event.ToVersion = sanitizeVersion(local.Version)
		event.PluginType = plugin.NormalizePluginType(local.Type)
		inner := local.Inner
		event.Inner = &inner
	}
}

func pluginFlagValues(args []string, names ...string) []string {
	allowed := make(map[string]struct{}, len(names))
	for _, name := range names {
		allowed[name] = struct{}{}
	}
	var values []string
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		nameValue := strings.TrimLeft(arg, "-")
		name, inline, hasInline := strings.Cut(nameValue, "=")
		if _, ok := allowed[name]; !ok {
			continue
		}
		if hasInline {
			values = appendSafePluginNames(values, inline)
			continue
		}
		for i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			i++
			values = appendSafePluginNames(values, args[i])
			if name == "name" {
				break
			}
		}
	}
	return uniqueStrings(values)
}

func appendSafePluginNames(values []string, raw string) []string {
	for _, value := range strings.Split(raw, ",") {
		value = strings.TrimSpace(value)
		if safePluginNamePattern.MatchString(value) {
			values = append(values, value)
		}
	}
	return values
}

func firstFlagValue(args []string, target string) string {
	for i := 0; i < len(args); i++ {
		arg := strings.TrimSpace(args[i])
		if !strings.HasPrefix(arg, "-") {
			continue
		}
		name, inline, hasInline := strings.Cut(strings.TrimLeft(arg, "-"), "=")
		if name != target {
			continue
		}
		if hasInline {
			return strings.TrimSpace(inline)
		}
		if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
			return strings.TrimSpace(args[i+1])
		}
	}
	return ""
}

func hasFlag(args []string, target string) bool {
	for _, arg := range args {
		name := strings.TrimLeft(strings.TrimSpace(arg), "-")
		if name == target || strings.HasPrefix(name, target+"=") {
			return true
		}
	}
	return false
}

func sanitizeVersion(value string) string {
	value = strings.TrimSpace(value)
	if len(value) > 64 {
		value = value[:64]
	}
	for _, char := range value {
		if !(char == '.' || char == '-' || char == '_' ||
			char >= '0' && char <= '9' ||
			char >= 'A' && char <= 'Z' ||
			char >= 'a' && char <= 'z') {
			return ""
		}
	}
	return value
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
