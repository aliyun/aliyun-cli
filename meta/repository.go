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
	"encoding/json"
	"sort"
	"strings"

	_ "embed"

	jmespath "github.com/jmespath/go-jmespath"
)

type Repository struct {
	Products []Product
	Names    []string

	index  map[string]Product
	reader Reader
}

func LoadRepository(reader Reader) *Repository {
	var e ProductSet
	err := ReadJsonFrom(reader, "products.json", &e)
	if err != nil {
		panic(err)
	}

	r := Repository{
		index:  make(map[string]Product),
		reader: reader,
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

	err := ReadJsonFrom(a.reader, strings.ToLower(product.Code)+"/"+apiName+".json", &result)
	if err != nil {
		return result, false
	}
	result.Product = &product
	return result, true
}

//go:embed versions.json
var versions []byte

func (a *Repository) GetStyle(productCode, version string) (string, bool) {
	v := new(interface{})
	err := json.Unmarshal(versions, v)
	if err != nil {
		return "", false
	}
	styles, _ := jmespath.Search("[?code=='"+productCode+"'].styles[]", *v)
	style, _ := jmespath.Search("[?version=='"+version+"'].style", styles)
	if len(style.([]interface{})) == 0 {
		return "", false
	}
	return style.([]interface{})[0].(string), true
}
