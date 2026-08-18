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
package meta

import (
	"strings"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
)

type ProductSet struct {
	Products []Product `json:"products"`
}

type Product struct {
	Code                    string            `json:"code"`
	Version                 string            `json:"version"`
	Catalog1                map[string]string `json:"catalog1"`
	Catalog2                map[string]string `json:"catalog2"`
	Name                    map[string]string `json:"name"`
	LocationServiceCode     string            `json:"location_service_code"`
	RegionalEndpoints       map[string]string `json:"regional_endpoints"`
	RegionalVpcEndpoints    map[string]string `json:"regional_vpc_endpoints"`
	GlobalEndpoint          string            `json:"global_endpoint"`
	RegionalEndpointPattern string            `json:"regional_endpoint_patterns"`
	ApiStyle                string            `json:"api_style"`
	ApiNames                []string          `json:"apis"`
	PluginDefaultVersion    string            `json:"plugin_default_version"`
	Versions                []string          `json:"versions"`
	Distribution            string            `json:"distribution,omitempty"`
}

func (a *Product) GetLowerCode() string {
	return strings.ToLower(a.Code)
}

func (a *Product) GetEndpoint(region string, client *sdk.Client) (endpoint string, err error) {
	return a.GetEndpointWithType(region, client, "")
}

func (a *Product) GetEndpointWithType(region string, client *sdk.Client, endpointType string) (endpoint string, err error) {
	// VPC endpoints remain independent from public endpoints generated from portal metadata.
	if strings.ToLower(endpointType) == "vpc" && a.RegionalVpcEndpoints != nil && len(a.RegionalVpcEndpoints) > 0 {
		ep, ok := a.RegionalVpcEndpoints[region]
		if ok {
			return ep, nil
		}
	}

	ep, ok := a.RegionalEndpoints[region]
	if ok {
		return ep, nil
	}

	if a.GlobalEndpoint != "" {
		return a.GlobalEndpoint, nil
	}

	return "", &InvalidEndpointError{nil, region, a}
}
