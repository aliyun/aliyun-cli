package meta

import (
	"strings"
)

type Profile struct {
	Products []Product 				`yaml:"products"`
	products map[string]*Product	`yaml:"-"`
}

func LoadProfile(reader FileReader) *Profile {
	var profile Profile
	err := reader.ReadYaml("profile.yml", &profile)
	if err != nil {
		panic(err)
	}

	products := make(map[string]*Product)
	for _, product := range profile.Products {
		name := strings.ToLower(product.Name)
		_, ok := products[name]
		if !ok {
			products[name] = &product
		} else {
			panic("Duplicated Name:" + product.Name)
		}
	}
	profile.products = products
	return &profile
}
