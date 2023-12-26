package newmeta

import (
	"encoding/json"
	"strings"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/aliyun-openapi-meta"
)

// {
// 	"products": [
// 	  {
// 		"code": "ARMS",
// 		"name": "Application Real-Time Monitoring Service",
// 		"version": "2019-08-08",
// 		"endpointType": "regional",
// 		"endpoints": {
// 		  "us-west-1": {
// 			"regionId": "us-west-1",
// 			"regionName": "US (Silicon Valley)",
// 			"areaId": "europeAmerica",
// 			"areaName": "Europe & Americas",
// 			"public": "arms.us-west-1.aliyuncs.com",
// 			"vpc": "arms-vpc.us-east-1.aliyuncs.com"
// 		  }
// 		}
// 	  }
// 	]
// }

type ProductSet struct {
	Products []Product `json:"products"`
}

type Product struct {
	Code         string              `json:"code"`
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	EndpointType string              `json:"endpointType"`
	Endpoints    map[string]Endpoint `json:"endpoints"`
}

type Endpoint struct {
	RegionId     string `json:"regionId"`
	Name         string `json:"regionName"`
	Version      string `json:"areaId"`
	EndpointType string `json:"areaName"`
	Public       string `json:"public"`
	VPC          string `json:"vpc"`
}

func GetProductName(language, code string) (name string, err error) {
	content, err := GetMetadata(language, "/products.json")
	if err != nil {
		return
	}

	products := new(ProductSet)
	err = json.Unmarshal(content, &products)

	for _, p := range products.Products {
		if strings.EqualFold(p.Code, code) {
			name = strings.TrimSpace(p.Name)
			break
		}
	}

	return
}

func GetMetadataPrefix(language string) string {
	if language == "en" {
		return "en-US"
	}
	return "zh-CN"
}

func GetMetadata(language string, path string) (content []byte, err error) {
	content, err = aliyunopenapimeta.Metadatas.ReadFile(GetMetadataPrefix(language) + path)
	return
}
