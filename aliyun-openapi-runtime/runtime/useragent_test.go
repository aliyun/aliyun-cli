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

import "testing"

func TestBuildBaseUserAgent(t *testing.T) {
	got := BuildBaseUserAgent("")
	want := "aliyun-openapi-runtime/" + Version
	if got != want {
		t.Fatalf("BuildBaseUserAgent(\"\") = %q, want %q", got, want)
	}
	got = BuildBaseUserAgent("3.0.234")
	want = "Aliyun-CLI/3.0.234 aliyun-openapi-runtime/" + Version
	if got != want {
		t.Fatalf("BuildBaseUserAgent(cli) = %q, want %q", got, want)
	}
}

func TestComposeUserAgent(t *testing.T) {
	got := ComposeUserAgent("3.0.1", "tool/1 AlibabaCloud-AIMode/enabled AlibabaCloud-Agent-Skills")
	want := "Aliyun-CLI/3.0.1 aliyun-openapi-runtime/" + Version + " tool/1 AlibabaCloud-AIMode/enabled AlibabaCloud-Agent-Skills"
	if got != want {
		t.Fatalf("ComposeUserAgent = %q, want %q", got, want)
	}
}

func TestComposeUserAgentWithPlugin(t *testing.T) {
	got := ComposeUserAgentWithPlugin("3.0.1", "aliyun-cli-fc", "0.7.1", "tool/1")
	want := "Aliyun-CLI/3.0.1 aliyun-openapi-runtime/" + Version + " aliyun-cli-fc/0.7.1 tool/1"
	if got != want {
		t.Fatalf("ComposeUserAgentWithPlugin = %q, want %q", got, want)
	}
}

func TestComposeUserAgentWithPluginRequiresCompleteIdentity(t *testing.T) {
	got := ComposeUserAgentWithPlugin("3.0.1", "aliyun-cli-fc", "", "tool/1")
	want := "Aliyun-CLI/3.0.1 aliyun-openapi-runtime/" + Version + " tool/1"
	if got != want {
		t.Fatalf("ComposeUserAgentWithPlugin incomplete identity = %q, want %q", got, want)
	}
}
