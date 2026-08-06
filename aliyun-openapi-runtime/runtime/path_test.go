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

package runtime

import (
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/meta"
)

func TestEncodePathGoPluginParity(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "plain", path: "/things/abc-123", want: "/things/abc-123"},
		{name: "space", path: "/things/a b", want: "/things/a%20b"},
		{name: "plus", path: "/things/a+b", want: "/things/a%2Bb"},
		{name: "percent", path: "/things/a%b", want: "/things/a%25b"},
		{name: "unicode", path: "/things/中文", want: "/things/%E4%B8%AD%E6%96%87"},
		{name: "fragment marker", path: "/things/a#b", want: "/things/a%23b"},
		{name: "colon", path: "/things/a:b", want: "/things/a%3Ab"},
		{name: "asterisk and tilde", path: "/things/a*b~c", want: "/things/a%2Ab~c"},
		{name: "slash remains a separator", path: "/things/a/b", want: "/things/a/b"},
		{name: "query suffix remains unchanged", path: "/things/a b?next=x y", want: "/things/a%20b?next=x y"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := encodePath(tt.path); got != tt.want {
				t.Fatalf("encodePath(%q) = %q, want %q", tt.path, got, tt.want)
			}
		})
	}
}

func TestAssembleEncodesPathAfterSubstitution(t *testing.T) {
	api := &meta.API{
		Name:    "GetThing",
		Version: "2024-01-01",
		Method:  "GET",
		Style:   meta.StyleROA,
		URL:     "/things/{thingId}:inspect",
		Parameters: []meta.Parameter{
			{Name: "thing_id", RawName: "thingId", Type: meta.TypeString, Position: meta.PosPath, Required: true},
		},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"thingId": "thing-123"}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if want := "/things/thing-123%3Ainspect"; req.Pathname != want {
		t.Fatalf("pathname = %q, want %q", req.Pathname, want)
	}
}

func TestAssemblePreservesInlineQuerySuffix(t *testing.T) {
	api := &meta.API{
		Name:    "GetAgentStatus",
		Version: "2024-06-26",
		Method:  "GET",
		Style:   meta.StyleROA,
		URL:     "/agent/{agentName}?status",
		Parameters: []meta.Parameter{
			{Name: "agent_name", RawName: "agentName", Type: meta.TypeString, Position: meta.PosPath, Required: true},
		},
	}
	req, err := Assemble(&ExecContext{API: api, Args: map[string]any{"agentName": "test_agent_name"}})
	if err != nil {
		t.Fatalf("Assemble: %v", err)
	}
	if want := "/agent/test_agent_name?status"; req.Pathname != want {
		t.Fatalf("pathname = %q, want %q", req.Pathname, want)
	}
}
