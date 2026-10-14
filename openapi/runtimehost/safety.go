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

package runtimehost

import (
	"github.com/aliyun/aliyun-cli/v3/sysconfig/safety"
	"github.com/aliyun/aliyun-openapi-runtime/source"
	"strings"
)

func safetyAPICommand(cmd safety.CommandInfo, args []string) safety.CommandInfo {
	ldr, err := Engine().Loader()
	if err != nil || ldr.EnsureProduct(cmd.Product) != nil {
		return cmd
	}
	provenance := ldr.Provenance(cmd.Product)
	if provenance == nil || provenance.Kind != source.KindBaseline {
		return cmd
	}
	// Match the runtime engine's first --api-version selection, including its
	// environment/default resolution. Never borrow identities from another version.
	requested := ""
	for i, arg := range args {
		if arg == "--api-version" {
			if i+1 < len(args) {
				requested = args[i+1]
			}
			break
		}
		if strings.HasPrefix(arg, "--api-version=") {
			requested = strings.TrimPrefix(arg, "--api-version=")
			break
		}
	}
	version, err := ldr.ResolveVersion(cmd.Product, requested)
	if err != nil {
		return cmd
	}
	index, err := ldr.GetAPIIndex(cmd.Product, version)
	if err != nil || index == nil {
		return cmd
	}
	aliases := make(map[string]string, len(index.Entries))
	for name, entry := range index.Entries {
		aliases[name] = entry.CmdName
	}
	return safety.WithAPIIdentities(cmd, aliases)
}
