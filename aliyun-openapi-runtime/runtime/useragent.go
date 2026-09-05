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

import "strings"

// BuildBaseUserAgent returns the engine's fixed UA prefix:
//
//	[Aliyun-CLI/{cliVersion}] aliyun-openapi-runtime/{Version}
//
// CLI version is stamped first. cliVersion is the embedding CLI version (e.g. cli.GetVersion()); empty omits the Aliyun-CLI segment.
func BuildBaseUserAgent(cliVersion string) string {
	runtimeSeg := "aliyun-openapi-runtime/" + Version
	if v := strings.TrimSpace(cliVersion); v != "" {
		return "Aliyun-CLI/" + v + " " + runtimeSeg
	}
	return runtimeSeg
}

// ComposeUserAgent joins the base prefix with host-supplied suffixes (--user-agent, AI-mode segments). Empty suffix yields the base alone.
func ComposeUserAgent(cliVersion, suffix string) string {
	base := BuildBaseUserAgent(cliVersion)
	if s := strings.TrimSpace(suffix); s != "" {
		return strings.TrimSpace(base + " " + s)
	}
	return base
}

// ComposeUserAgentWithPlugin inserts a metadata plugin package segment between
// the fixed base and host-supplied suffixes. Both name and version must be set.
func ComposeUserAgentWithPlugin(cliVersion, pluginName, pluginVersion, suffix string) string {
	base := BuildBaseUserAgent(cliVersion)
	name := sanitizeUserAgentToken(pluginName)
	version := sanitizeUserAgentToken(pluginVersion)
	if name != "" && version != "" {
		base += " " + name + "/" + version
	}
	if s := strings.TrimSpace(suffix); s != "" {
		base += " " + s
	}
	return strings.TrimSpace(base)
}

func sanitizeUserAgentToken(value string) string {
	return strings.Map(func(r rune) rune {
		if r <= ' ' || r == 0x7f {
			return -1
		}
		return r
	}, strings.TrimSpace(value))
}
