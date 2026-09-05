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
	"strings"
	"testing"

	"github.com/aliyun/aliyun-openapi-runtime/argparser"
)

func TestEngineExternalFlags(t *testing.T) {
	specs := engineExternalFlags()
	byName := make(map[string]argparser.ExternalFlagSpec, len(specs))
	for _, spec := range specs {
		byName[spec.Name] = spec
	}

	if got := byName["profile"]; got.Mode != argparser.ExternalFlagOptional || got.Shorthand != 'p' {
		t.Errorf("profile spec = %+v", got)
	}
	if got := byName["read-timeout"]; got.Mode != argparser.ExternalFlagRequired {
		t.Errorf("read-timeout spec = %+v", got)
	}
	if got := byName["skip-secure-verify"]; got.Mode != argparser.ExternalFlagNone {
		t.Errorf("skip-secure-verify spec = %+v", got)
	}
	if _, ok := byName["region"]; ok {
		t.Error("engine-owned --region must not be external")
	}
	if _, ok := byName["endpoint"]; ok {
		t.Error("engine-owned --endpoint must not be external")
	}
	if got := byName["RegionId"]; !strings.Contains(got.RejectMessage, "aliyun <product> <command> --help") {
		t.Errorf("RegionId rejection = %+v", got)
	}
}
