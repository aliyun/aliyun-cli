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

const (
	CommandTypeCore           = "core"
	CommandTypeCLIExt         = "cliext"
	CommandTypeOpenAPILegacy  = "openapi-legacy"
	CommandTypeOpenAPIRuntime = "openapi-runtime"
	CommandTypePluginGo       = "plugin-go"
	CommandTypePluginMeta     = "plugin-meta"
)

const (
	PipelinePublic = "public"
	PipelineInner  = "inner"
)

type Scope struct {
	Active       bool
	Pipeline     string
	PluginName   string
	PluginVer    string
	PluginType   string
	CommandToken string
}

var cliExtensions = map[string]struct{}{}

// RegisterCLIExtensions records the root commands implemented by cliext.
func RegisterCLIExtensions(extensions map[string]bool) {
	defer recoverCollectionPanic()
	registered := make(map[string]struct{}, len(extensions))
	for name, enabled := range extensions {
		if enabled {
			registered[strings.ToLower(strings.TrimSpace(name))] = struct{}{}
		}
	}
	cliExtensions = registered
}

func isCLIExtension(command string) bool {
	_, ok := cliExtensions[strings.ToLower(command)]
	return ok
}

// DetectCommandType classifies the command implementation without collecting
// argument values. Installed plugin metadata and the registered root command
// set are authoritative; OpenAPI command casing selects the remaining route.
func DetectCommandType(args []string) string {
	parts := positionalParts(args)
	if len(parts) == 0 {
		return ""
	}
	commandToken := strings.ToLower(strings.TrimSpace(parts[0]))
	if isCLIExtension(commandToken) {
		return CommandTypeCLIExt
	}
	if local, err := plugin.LookupLocalPlugin(commandToken); err == nil && local != nil {
		return classifyCommandType(parts, plugin.NormalizePluginType(local.Type))
	}
	return classifyCommandType(parts, "")
}

// detectCommandTypeForScope reuses plugin metadata resolved while selecting
// the telemetry pipeline, so event construction does not read the manifest again.
func detectCommandTypeForScope(args []string, scope Scope) string {
	return classifyCommandType(positionalParts(args), scope.PluginType)
}

func classifyCommandType(parts []string, pluginType string) string {
	if len(parts) == 0 {
		return ""
	}
	commandToken := strings.ToLower(strings.TrimSpace(parts[0]))
	if isCLIExtension(commandToken) {
		return CommandTypeCLIExt
	}
	if pluginType != "" {
		if plugin.NormalizePluginType(pluginType) == plugin.PluginTypeMeta {
			return CommandTypePluginMeta
		}
		return CommandTypePluginGo
	}
	if plugin.IsReservedTopLevelCommand(commandToken) {
		return CommandTypeCore
	}
	if len(parts) > 1 && strings.ToLower(parts[1]) == parts[1] {
		return CommandTypeOpenAPIRuntime
	}
	return CommandTypeOpenAPILegacy
}

func scopeFromLocalPlugin(commandToken string, lp *plugin.LocalPlugin) Scope {
	scope := Scope{
		Active:       true,
		Pipeline:     PipelinePublic,
		CommandToken: commandToken,
	}
	if lp == nil {
		return scope
	}
	scope.PluginName = lp.Name
	scope.PluginVer = lp.Version
	scope.PluginType = plugin.NormalizePluginType(lp.Type)
	if lp.Inner {
		scope.Pipeline = PipelineInner
	}
	return scope
}

func DetectInnerScope(commandToken string) Scope {
	if commandToken == "" {
		return Scope{}
	}
	if plugin.IsReservedTopLevelCommand(commandToken) {
		return Scope{}
	}
	lp, err := plugin.LookupLocalPlugin(commandToken)
	if err != nil || lp == nil || !lp.Inner {
		return Scope{}
	}
	return scopeFromLocalPlugin(commandToken, lp)
}

func SelectScope(commandToken string, cfg *Config) (bool, Scope) {
	if commandToken == "" || Disabled(cfg) {
		return false, Scope{}
	}
	if plugin.IsReservedTopLevelCommand(commandToken) {
		return true, scopeFromLocalPlugin(commandToken, nil)
	}
	lp, err := plugin.LookupLocalPlugin(commandToken)
	if err != nil {
		return true, scopeFromLocalPlugin(commandToken, nil)
	}
	return true, scopeFromLocalPlugin(commandToken, lp)
}

func ShouldCollectInner(commandToken string, cfg *Config) (bool, Scope) {
	if Disabled(cfg) {
		return false, Scope{}
	}
	scope := DetectInnerScope(commandToken)
	return scope.Active, scope
}
