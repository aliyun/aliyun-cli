/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package meta

import (
	"strings"
	"sort"
)

type Library struct {
	Products []Product
	Names []string

	index map[string]Product
	reader Reader
}

func LoadLibrary(reader Reader) *Library {
	var e Products
	err := ReadYamlFrom(reader, "products.yml", &e)
	if err != nil {
		panic(err)
	}

	r := Library {
		index: make(map[string]Product),
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

func (a *Library) GetProduct(code string) (Product, bool) {
	p, ok := a.index[strings.ToLower(code)]
	return p, ok
}

func (a *Library) GetApi(productCode string, version string, apiName string) (Api, bool) {
	var result Api
	product, ok := a.GetProduct(productCode)
	if !ok {
		return result, false
	}

	err := ReadYamlFrom(a.reader, product.Code + "/" + apiName + ".yml", &result)
	if err != nil {
		return result, false
	}
	return result, true
}
