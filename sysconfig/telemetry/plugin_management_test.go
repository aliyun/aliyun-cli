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
	"testing"
)

func TestDetectPluginManagement(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		operation  string
		sourceType string
		names      []string
		toVersion  string
	}{
		{
			name:       "catalog install",
			args:       []string{"plugin", "install", "--name=aliyun-cli-demo", "--version", "1.2.3"},
			operation:  "install",
			sourceType: "catalog",
			names:      []string{"aliyun-cli-demo"},
			toVersion:  "1.2.3",
		},
		{
			name:       "package install omits source",
			args:       []string{"plugin", "install-custom", "--package", "/secret/path/plugin.zip"},
			operation:  "install_custom",
			sourceType: "package",
		},
		{
			name:       "update",
			args:       []string{"plugin", "update", "--name", "demo"},
			operation:  "update",
			sourceType: "catalog",
			names:      []string{"demo"},
		},
		{
			name:      "uninstall",
			args:      []string{"plugin", "uninstall", "--name", "demo"},
			operation: "uninstall",
			names:     []string{"demo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DetectPluginManagement(tt.args)
			if got == nil ||
				got.Operation != tt.operation ||
				got.SourceType != tt.sourceType ||
				got.ToVersion != tt.toVersion ||
				!equalStrings(got.PluginNames, tt.names) {
				t.Fatalf("unexpected plugin event: %+v", got)
			}
			encoded, err := json.Marshal(got)
			if err != nil {
				t.Fatal(err)
			}
			if contains(string(encoded), "/secret/") {
				t.Fatalf("package path leaked into event: %s", encoded)
			}
		})
	}
}

func TestDetectPluginManagementIgnoresReadOnlyCommands(t *testing.T) {
	for _, args := range [][]string{
		{"plugin", "list"},
		{"plugin", "show", "--name", "demo"},
		{"ecs", "DescribeInstances"},
	} {
		if got := DetectPluginManagement(args); got != nil {
			t.Fatalf("%v unexpectedly produced %+v", args, got)
		}
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
