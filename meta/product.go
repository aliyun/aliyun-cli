/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package meta

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
)

type Products struct {
	Products []Product 				`yaml:"products"`
}

type Product struct {
	Code                string            `yaml:"code"`
	Version             string            `yaml:"version"`
	LocationServiceCode string            `yaml:"location_service_code"`
	Name                map[string]string `yaml:"name"`
	DocumentId			string 			  `yaml:"document_id"`
	SiteUrl				string			  `yaml:"site_url"`
	Sites				[]string 		  `yaml:"sites"`
	EndpointPatterns    []string          `yaml:"endpoint_patterns"`
	Endpoints           map[string]string `yaml:"endpoints"`
	KnownEndpoints      map[string]string `yaml:"known_endpoints"`
	ApiStyle            string            `yaml:"api_style"`
	ApiNames            []string          `yaml:"apis"`
	apis                map[string]Api    `yaml:"-"`
}

func (a *Product) GetEndpoint(region string, client *sdk.Client) (string, error) {
	ep, ok := a.Endpoints[region]
	if ok {
		return ep, nil
	}
	if a.LocationServiceCode != "" {
		rp := endpoints.ResolveParam{
			Product:          a.Code,
			RegionId:         region,
			LocationProduct:  a.LocationServiceCode,
			LocationEndpointType: "openAPI",
			CommonApi:        client.ProcessCommonRequest,
		}
		ep, err := endpoints.Resolve(&rp)
		if err == nil {
			return ep, nil
		}
	}

	ep, ok = a.KnownEndpoints[region]
	if ok {
		return ep, nil
	}
	return "", nil
}

func (a *Product) TryGetEndpoints(region string, client *sdk.Client) (endpoint string, lcEndpoint string) {
	endpoint, _ = a.Endpoints[region]

	rp := endpoints.ResolveParam{
		Product:          a.Code,
		RegionId:         region,
		LocationProduct:  a.LocationServiceCode,
		LocationEndpointType: "openAPI",
		CommonApi:        client.ProcessCommonRequest,
	}
	lcEndpoint, err := endpoints.Resolve(&rp)
	if err != nil {
		lcEndpoint = ""
	}

	//if strings.Contains(a.Domain, "[RegionId]") {
	//	return "", fmt.Errorf("can't resolve endpoint for %s(%s):%s, use --endpoint xxx.aliyuncs.com parameter instead",
	//		a.Name, a.LocationServiceCode, a.Domain)
	//} else {
	//	return "", nil
	//}
	return
}

func (a *Product) GetDocumentLink(lang string) string {
	switch lang {
	case "zh":
		return "https://help.aliyun.com/document_detail/" + a.DocumentId + ".html"
	case "en":
		return "https://www.alibabacloud.com/help/doc-detail/" + a.DocumentId + ".html"
	case "jp":
		return "https://jp.alibabacloud.com/help/doc-detail/" + a.DocumentId + ".html"
	default:
		return "https://www.alibabacloud.com/help/doc-detail/" + a.DocumentId + ".html"
	}
}


