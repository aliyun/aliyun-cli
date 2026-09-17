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
	"os"
	"regexp"
	"strings"
)

var financeRegions = map[string]struct{}{
	"cn-shanghai-finance-1": {},
	"cn-shenzhen-finance-1": {},
}

var privateCloudEndpointPattern = regexp.MustCompile(`(?i)(apsara|private[-_]?cloud|专有云)`)

func envTruthy(key string) bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	return v == "1" || v == "true" || v == "yes"
}

func GlobalOptOut() bool {
	return envTruthy("ALIYUN_CLI_TELEMETRY_OPTOUT")
}

func InnerOptOut() bool {
	return envTruthy("ALIYUN_CLI_TELEMETRY_INNER_OPTOUT")
}

type ProfileSnapshot struct {
	RegionID string
	Endpoint string
}

func L1HardDisabled(p ProfileSnapshot) bool {
	if p.RegionID != "" {
		if _, ok := financeRegions[strings.ToLower(p.RegionID)]; ok {
			return true
		}
	}
	if p.Endpoint != "" && privateCloudEndpointPattern.MatchString(p.Endpoint) {
		return true
	}
	return false
}

func InnerPipelineDisabled(p ProfileSnapshot, cfg *Config) bool {
	if L1HardDisabled(p) {
		return true
	}
	if GlobalOptOut() || InnerOptOut() {
		return true
	}
	if !InnerAutoOnEnabled(cfg) {
		return true
	}
	return false
}
