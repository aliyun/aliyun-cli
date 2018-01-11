package meta

import (
	"encoding/json"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"strings"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/endpoints"
	"fmt"
)

type Profile struct {
	Products []Product 			`json:"products"`
}

type Product struct {
	Name string						`json:"name"`
	MajorVersion string				`json:"major_version"`
	Domain string					`json:"domain"`
	LocationServiceCode string		`json:"location_service_code"`
	Endpoints map[string]string		`json:"endpoints"`
}

func (a *Product) GetEndpoint(region string, client *sdk.Client) (string, error) {
	ep, ok := a.Endpoints[region]
	if ok {
		return ep, nil
	}
	if a.LocationServiceCode != "" {
		rp := endpoints.ResolveParam {
			Product: a.Name,
			RegionId: region,
			LocationProduct: a.LocationServiceCode,
			LocationEndpoint: "openAPI",
			CommonApi: client.ProcessCommonRequest,
		}
		ep, err := endpoints.Resolve(&rp)
		if err == nil {
			return ep, nil
		}
	}
	if strings.Contains(a.Domain, "[RegionId]") {
		return "", fmt.Errorf("can't resolve endpoint for %s(%s):%s, use --endpoint xxx.aliyuncs.com parameter instead",
			a.Name, a.LocationServiceCode, a.Domain)
	} else {
		return a.Domain, nil
	}
}


func LoadProfile() (map[string]Product) {
	var profile Profile
	err := json.Unmarshal(profileJson, &profile)
	if err != nil {
		panic(err)
	}

	products := make(map[string]Product)
	for _, product := range profile.Products {
		products[product.Name] = product
	}
	return products
}