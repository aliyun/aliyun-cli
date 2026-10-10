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

import "github.com/aliyun/aliyun-cli/v3/cli/plugin"

const (
	PipelinePublic = "public"
	PipelineInner  = "inner"
)

type Scope struct {
	Active       bool
	Pipeline     string
	Trigger      string
	PluginName   string
	PluginVer    string
	CommandToken string
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
	return Scope{
		Active:       true,
		Pipeline:     PipelineInner,
		Trigger:      "plugin_manifest",
		PluginName:   lp.Name,
		PluginVer:    lp.Version,
		CommandToken: commandToken,
	}
}

func SelectScope(commandToken string, p ProfileSnapshot, cfg *Config) (bool, Scope) {
	if commandToken == "" {
		return false, Scope{}
	}
	inner := DetectInnerScope(commandToken)
	if inner.Active {
		if InnerPipelineDisabled(p, cfg) {
			return false, Scope{}
		}
		return true, inner
	}
	if PublicPipelineDisabled(p, cfg) {
		return false, Scope{}
	}
	scope := Scope{
		Active:       true,
		Pipeline:     PipelinePublic,
		Trigger:      "public_default",
		CommandToken: commandToken,
	}
	if !plugin.IsReservedTopLevelCommand(commandToken) {
		if lp, err := plugin.LookupLocalPlugin(commandToken); err == nil && lp != nil {
			scope.PluginName = lp.Name
			scope.PluginVer = lp.Version
		}
	}
	return true, scope
}

func ShouldCollectInner(commandToken string, p ProfileSnapshot, cfg *Config) (bool, Scope) {
	if InnerPipelineDisabled(p, cfg) {
		return false, Scope{}
	}
	scope := DetectInnerScope(commandToken)
	return scope.Active, scope
}
