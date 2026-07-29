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
	"fmt"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

// engineExternalFlags describes root-parsed flags that remain in the raw argv
// tail passed to the engine. The engine consumes these tokens syntactically;
// their values continue to be owned by ctx/profileHost.
func engineExternalFlags() []argparser.ExternalFlagSpec {
	ref := cli.NewFlagSet()
	config.AddFlags(ref)

	specs := make([]argparser.ExternalFlagSpec, 0, len(ref.Flags())+3)
	for _, flag := range ref.Flags() {
		switch flag.Name {
		case config.RegionFlagName, config.EndpointFlagName:
			// Engine-reserved globals.
			continue
		case config.RegionIdFlagName:
			specs = append(specs, argparser.ExternalFlagSpec{
				Name: flag.Name,
				Mode: argparser.ExternalFlagRequired,
				RejectMessage: "--RegionId is only supported by legacy PascalCase commands; " +
					"for kebab-case commands, use --region for the endpoint/signing region, " +
					"or check 'aliyun <product> <command> --help' for the API's RegionId parameter (for example --biz-region-id)",
			})
		default:
			specs = append(specs, argparser.ExternalFlagSpec{
				Name:      flag.Name,
				Shorthand: flag.Shorthand,
				Mode:      externalFlagMode(flag),
			})
		}
	}

	specs = append(specs,
		argparser.ExternalFlagSpec{Name: "user-agent", Mode: argparser.ExternalFlagRequired},
		argparser.ExternalFlagSpec{Name: "cli-ai-mode", Mode: argparser.ExternalFlagNone},
		argparser.ExternalFlagSpec{Name: "no-cli-ai-mode", Mode: argparser.ExternalFlagNone},
	)
	return specs
}

func externalFlagMode(flag *cli.Flag) argparser.ExternalFlagMode {
	switch flag.AssignedMode {
	case cli.AssignedNone:
		return argparser.ExternalFlagNone
	case cli.AssignedDefault:
		return argparser.ExternalFlagOptional
	case cli.AssignedOnce:
		return argparser.ExternalFlagRequired
	case cli.AssignedRepeatable:
		panic(fmt.Sprintf("unsupported repeatable host flag --%s", flag.Name))
	default:
		panic(fmt.Sprintf("host flag --%s has unknown assigned mode %d", flag.Name, flag.AssignedMode))
	}
}
