// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
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
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
)

type ProductSet struct {
	Products []Product `json:"products"`
}

type SiteType string

const (
	ChinaSite         = SiteType("china")
	InternationalSite = SiteType("international")
	JapanSite         = SiteType("japan")
)

type Product struct {
	Code     string            `json:"code"`
	Version  string            `json:"version"`
	Catalog1 map[string]string `json:"catalog1"`
	Catalog2 map[string]string `json:"catalog2"`
	Name     map[string]string `json:"name"`

	LocationServiceCode     string            `json:"location_service_code"`
	RegionalEndpoints       map[string]string `json:"regional_endpoints"`
	GlobalEndpoint          string            `json:"global_endpoint"`
	RegionalEndpointPattern string            `json:"regional_endpoint_patterns"`

	ApiStyle string         `json:"api_style"`
	ApiNames []string       `json:"apis"`
	apis     map[string]Api `yaml:"-"`
}

func (a *Product) GetLowerCode() string {
	return strings.ToLower(a.Code)
}

func (a *Product) GetEndpoint(region string, client *sdk.Client) (string, error) {
	var le error
	if a.LocationServiceCode != "" {
		rp := endpoints.ResolveParam{
			Product:              a.Code,
			RegionId:             region,
			LocationProduct:      a.LocationServiceCode,
			LocationEndpointType: "openAPI",
			CommonApi:            client.ProcessCommonRequest,
		}
		ep, err := endpoints.Resolve(&rp)
		if err == nil {
			return ep, nil
		} else {
			le = err
		}
	}

	ep, ok := a.RegionalEndpoints[region]
	if ok {
		return ep, nil
	}

	if a.GlobalEndpoint != "" {
		return a.GlobalEndpoint, nil
	}

	return "", &InvalidEndpointError{le, region, a}
}

func (a *Product) TryGetEndpoints(region string, client *sdk.Client) (endpoint string, lcEndpoint string) {
	endpoint, _ = a.RegionalEndpoints[region]

	if a.LocationServiceCode != "" {
		rp := endpoints.ResolveParam{
			Product:              a.Code,
			RegionId:             region,
			LocationProduct:      a.LocationServiceCode,
			LocationEndpointType: "openAPI",
			CommonApi:            client.ProcessCommonRequest,
		}
		ep, err := endpoints.Resolve(&rp)
		if err != nil {
			lcEndpoint = ""
		} else {
			lcEndpoint = ep
		}
	}

	//if strings.Contains(a.Domain, "[RegionId]") {
	//	return "", fmt.Errorf("can't resolve endpoint for %s(%s):%s, use --endpoint xxx.aliyuncs.com parameter instead",
	//		a.Name, a.LocationServiceCode, a.Domain)
	//} else {
	//	return "", nil
	//}
	return
}
