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
	"encoding/json"
	"regexp"
	"sort"
	"strings"

	_ "embed"
)

type Repository struct {
	Products []Product
	Names    []string

	index map[string]Product
}

func LoadRepository() *Repository {
	var e ProductSet
	err := ReadJsonFrom("products.json", &e)
	if err != nil {
		panic(err)
	}

	r := Repository{
		index: make(map[string]Product),
	}
	for _, product := range e.Products {
		name := strings.ToLower(product.Code)
		_, ok := r.index[name]
		if !ok {
			r.Names = append(r.Names, product.Code)
			r.index[name] = product
		} else {
			panic("Duplicated Name:" + product.Code)
		}
		sort.Strings(product.ApiNames)
		r.Products = append(r.Products, product)
	}
	sort.Strings(r.Names)
	return &r
}

func (a *Repository) GetProduct(code string) (Product, bool) {
	p, ok := a.index[strings.ToLower(code)]
	return p, ok
}

func (a *Repository) GetApi(productCode string, version string, apiName string) (Api, bool) {
	var result Api
	product, ok := a.GetProduct(productCode)
	if !ok {
		return result, false
	}

	err := ReadJsonFrom(strings.ToLower(product.Code)+"/"+apiName+".json", &result)
	if err != nil {
		return result, false
	}
	result.Product = &product
	return result, true
}

func (a *Repository) GetApiByPath(productCode string, version string, method string, path string) (Api, bool) {
	var result Api
	product, ok := a.GetProduct(productCode)
	if !ok {
		return result, false
	}
	// list all apis
	for _, apiName := range product.ApiNames {
		err := ReadJsonFrom(strings.ToLower(product.Code)+"/"+apiName+".json", &result)
		if err != nil {
			return result, false
		}
		// method not allowed
		if !strings.Contains(result.Method, method) {
			continue
		}
		// replace all [something] to *
		// example /permissions/users/[uid]/update to /permissions/users/*/update
		var pattern = ReplacePathPattern(result.PathPattern)
		// match path
		re := regexp.MustCompile("^" + pattern + "$")
		if re.MatchString(path) {
			result.Product = &product
			return result, true
		}
	}
	result = Api{}
	return result, false
}

//go:embed versions.json
var versions []byte

// [
// 	{
// 	"code": "aegis",
// 	"styles": [
// 		{
// 			"style": "RPC",
// 			"version": "2016-11-11"
// 		}
// 	]
// }
// ]

type ProductStyle struct {
	Code   string  `json:"code"`
	Styles []Style `json:"styles"`
}

type Style struct {
	Style   string `json:"style"`
	Version string `json:"version"`
}

func (a *Repository) GetStyle(productCode, version string) (string, bool) {
	productStyles := new([]ProductStyle)
	err := json.Unmarshal(versions, &productStyles)
	if err != nil {
		return "", false
	}

	for _, p := range *productStyles {
		if p.Code == productCode {
			for _, s := range p.Styles {
				if s.Version == version {
					return s.Style, true
				}
			}
		}
	}

	return "", false
}

func ReplacePathPattern(pattern string) string {
	re := regexp.MustCompile(`\[[^\]]+\]`)
	return re.ReplaceAllString(pattern, "[0-9a-zA-Z_\\-\\.{}]+")
}
