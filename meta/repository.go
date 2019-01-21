/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package meta

import (
	"sort"
	"strings"
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
