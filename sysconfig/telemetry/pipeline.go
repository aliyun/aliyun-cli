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

type InnerScope struct {
	Active       bool
	Trigger      string
	PluginName   string
	PluginVer    string
	CommandToken string
}

func DetectInnerScope(commandToken string) InnerScope {
	if commandToken == "" {
		return InnerScope{}
	}
	if plugin.IsReservedTopLevelCommand(commandToken) {
		return InnerScope{}
	}
	lp, err := plugin.LookupLocalPlugin(commandToken)
	if err != nil || lp == nil || !lp.Inner {
		return InnerScope{}
	}
	return InnerScope{
		Active:       true,
		Trigger:      "plugin_manifest",
		PluginName:   lp.Name,
		PluginVer:    lp.Version,
		CommandToken: commandToken,
	}
}

func ShouldCollectInner(commandToken string, p ProfileSnapshot, cfg *Config) (bool, InnerScope) {
	if InnerPipelineDisabled(p, cfg) {
		return false, InnerScope{}
	}
	scope := DetectInnerScope(commandToken)
	return scope.Active, scope
}
