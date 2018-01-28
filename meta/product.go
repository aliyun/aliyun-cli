package meta

import (
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
)

type Products struct {
	Products []Product 				`yaml:"products"`
}

type Product struct {
	Name                string            `yaml:"name"`
	Version      		string            `yaml:"version"`
	LocationServiceCode string            `yaml:"location_service_code"`
	Descriptions		map[string]string `yaml:"descriptions"`
	Links				map[string]string `yaml:"links"`
	EndpointPatterns    []string          `yaml:"endpoint_patterns"`
	Endpoints           map[string]string `yaml:"endpoints"`
	KnownEndpoints      map[string]string `yaml:"known_endpoints"`
	ApiStyle			string 			  `yaml:"api_style"`
	ApiNames            []string          `yaml:"apis"`
	apis				map[string]Api	  `yaml:"-"`
}

func (a *Product) GetEndpoint(region string, client *sdk.Client) (string, error) {
	ep, ok := a.Endpoints[region]
	if ok {
		return ep, nil
	}
	if a.LocationServiceCode != "" {
		rp := endpoints.ResolveParam{
			Product:          a.Name,
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
	//if strings.Contains(a.Domain, "[RegionId]") {
	//	return "", fmt.Errorf("can't resolve endpoint for %s(%s):%s, use --endpoint xxx.aliyuncs.com parameter instead",
	//		a.Name, a.LocationServiceCode, a.Domain)
	//} else {
	//	return "", nil
	//}
	return "", nil
}


