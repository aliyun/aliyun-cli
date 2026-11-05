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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
)

func TestDetectCommandType(t *testing.T) {
	const (
		extension = "telemetry-cliext-type-test"
		core      = "telemetry-core-type-test"
	)
	RegisterCLIExtensions(map[string]bool{extension: true})
	plugin.RegisterReservedTopLevelCommands([]string{extension, core})

	pluginsDir := t.TempDir()
	t.Setenv(plugin.EnvPluginsDir, pluginsDir)
	manifest := plugin.LocalManifest{Plugins: map[string]plugin.LocalPlugin{
		"aliyun-cli-go-type-test": {
			Name: "aliyun-cli-go-type-test", Command: "go-type-test", Type: plugin.PluginTypeGo,
		},
		"aliyun-cli-meta-type-test": {
			Name: "aliyun-cli-meta-type-test", Command: "meta-type-test", Type: plugin.PluginTypeMeta,
		},
	}}
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pluginsDir, "manifest.json"), data, 0600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "cliext", args: []string{extension, "list"}, want: CommandTypeCLIExt},
		{name: "core", args: []string{core, "show"}, want: CommandTypeCore},
		{name: "legacy openapi", args: []string{"ecs", "DescribeInstances"}, want: CommandTypeOpenAPILegacy},
		{name: "runtime openapi", args: []string{"ecs", "describe-instances"}, want: CommandTypeOpenAPIRuntime},
		{name: "go plugin", args: []string{"go-type-test", "list"}, want: CommandTypePluginGo},
		{name: "meta plugin", args: []string{"meta-type-test", "list"}, want: CommandTypePluginMeta},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DetectCommandType(tt.args); got != tt.want {
				t.Fatalf("DetectCommandType(%v) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}

func TestDetectCommandTypeForScopeReusesResolvedPluginType(t *testing.T) {
	t.Setenv(plugin.EnvPluginsDir, t.TempDir())

	tests := []struct {
		name       string
		pluginType string
		want       string
	}{
		{name: "go plugin", pluginType: plugin.PluginTypeGo, want: CommandTypePluginGo},
		{name: "meta plugin", pluginType: plugin.PluginTypeMeta, want: CommandTypePluginMeta},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scope := Scope{PluginType: tt.pluginType}
			if got := detectCommandTypeForScope([]string{"not-installed", "list"}, scope); got != tt.want {
				t.Fatalf("detectCommandTypeForScope() = %q, want %q", got, tt.want)
			}
		})
	}
}
