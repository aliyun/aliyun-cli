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
		name := strings.ToLower(product.Name)
		_, ok := r.index[name]
		if !ok {
			r.Names = append(r.Names, product.Name)
			r.index[name] = product
		} else {
			panic("Duplicated Name:" + product.Name)
		}
		r.Products = append(r.Products, product)
	}
	sort.Strings(r.Names)
	return &r
}

func (a *Library) GetProduct(name string) (Product, bool) {
	p, ok := a.index[strings.ToLower(name)]
	return p, ok
}

func (a *Library) GetApi(name string, version string, apiName string) (Api, bool) {
	var result Api
	product, ok := a.GetProduct(name)
	if !ok {
		return result, false
	}

	err := ReadYamlFrom(a.reader, product.Name + "/" + apiName + ".yml", &result)
	if err != nil {
		return result, false
	}
	return result, true
}
