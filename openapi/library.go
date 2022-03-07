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
package openapi

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/resource"
)

type Library struct {
	lang        string
	builtinRepo *meta.Repository
	extraRepo   *meta.Repository
	writer      io.Writer
}

func NewLibrary(w io.Writer, lang string) *Library {
	return &Library{
		builtinRepo: meta.LoadRepository(resource.NewReader()),
		extraRepo:   nil,
		lang:        lang,
		writer:      w,
	}
}

func (a *Library) GetProduct(productCode string) (meta.Product, bool) {
	return a.builtinRepo.GetProduct(productCode)
}

func (a *Library) GetApi(productCode string, version string, apiName string) (meta.Api, bool) {
	return a.builtinRepo.GetApi(productCode, version, apiName)
}

func (a *Library) GetStyle(productCode string, version string) (string, bool) {
	return a.builtinRepo.GetStyle(productCode, version)
}

func (a *Library) GetProducts() []meta.Product {
	return a.builtinRepo.Products
}

func (a *Library) PrintProducts() {
	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	cli.PrintfWithColor(w, cli.ColorOff, "\nProducts:\n")
	for _, product := range a.builtinRepo.Products {
		cli.PrintfWithColor(w, cli.Cyan, "  %s\t%s\n", strings.ToLower(product.Code), product.Name[i18n.GetLanguage()])
	}
	w.Flush()
}

func (a *Library) printProduct(product meta.Product) {
	cli.Printf(a.writer, "  %s(%s)\t%s\n", product.Code, product.Version, product.Name["zh"])
}

func (a *Library) PrintProductUsage(productCode string, withApi bool) error {
	product, ok := a.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a}
	}

	if product.ApiStyle == "rpc" {
		cli.Printf(a.writer, "\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", product.Code)
	} else {
		cli.Printf(a.writer, "\nUsage 1:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", product.Code)
		cli.Printf(a.writer, "\nUsage 2 (For API with NO PARAMS in PathPattern only.):\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ... --body \"...\"\n", product.Code)
	}

	cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	cli.Printf(a.writer, "Version: %s \n", product.Version)

	if withApi {
		cli.PrintfWithColor(a.writer, cli.ColorOff, "\nAvailable Api List: \n")
		maxNameLen := 0

		for _, apiName := range product.ApiNames {
			if len(apiName) > maxNameLen {
				maxNameLen = len(apiName)
			}
		}

		for _, apiName := range product.ApiNames {
			if product.ApiStyle == "restful" {
				api, _ := a.GetApi(productCode, product.Version, apiName)
				ptn := fmt.Sprintf("  %%-%ds : %%s %%s\n", maxNameLen+1)
				cli.PrintfWithColor(a.writer, cli.Green, ptn, apiName, api.Method, api.PathPattern)
			} else {
				cli.PrintfWithColor(a.writer, cli.Green, "  %s\n", apiName)
			}

		}
		// TODO some ApiName is too long, two column not seems good
		//w := tabwriter.NewWriter(cli.GetOutputWriter(), 8, 0, 1, ' ', 0)
		//for i := 0; i < len(product.ApiNames); i += 2 {
		//	name1 := product.ApiNames[i]
		//	name2 := ""
		//	if i + 1 < len(product.ApiNames) {
		//		name2 = product.ApiNames[i + 1]
		//	}
		//	fmt.Fprintf(w, "  %s\t%s\n", name1, name2)
		//}
		//w.Flush()
	}

	cli.Printf(a.writer, "\nRun `aliyun %s <ApiName> --help` to get more information about this API\n", product.GetLowerCode())
	return nil
}

func (a *Library) PrintApiUsage(productCode string, apiName string) error {
	product, ok := a.builtinRepo.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a}
	}
	api, ok := a.builtinRepo.GetApi(productCode, product.Version, apiName)
	if !ok {
		return &InvalidApiError{Name: apiName, product: &product}
	}

	if product.ApiStyle == "restful" {
		cli.Printf(a.writer, "\nProduct:     %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
		cli.Printf(a.writer, "Method:      %s\n", api.Method)
		cli.Printf(a.writer, "PathPattern: %s\n", api.PathPattern)
	} else {
		cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	}

	cli.Printf(a.writer, "\nParameters:\n")

	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	printParameters(w, api.Parameters, "")
	w.Flush()

	return nil
}

func printParameters(w io.Writer, params []meta.Parameter, prefix string) {

	sort.Sort(meta.ParameterSlice(params))

	for _, param := range params {
		if param.Hidden {
			continue
		}

		if param.Position == "Domain" {
			continue
		}

		if param.Position == "Header" {
			continue
		}

		if param.Type == "RepeatList" {
			if len(param.SubParameters) > 0 {
				printParameters(w, param.SubParameters, prefix+param.Name+".n.")
			} else {
				fmt.Fprintf(w, "  --%s%s.n\t%s\t%s\t%s\n", prefix, param.Name, param.Type, required(param.Required), getDescription(param.Description))
			}
		} else {
			fmt.Fprintf(w, "  --%s%s\t%s\t%s\t%s\n", prefix, param.Name, param.Type, required(param.Required), getDescription(param.Description))
		}
	}
}

func required(r bool) string {
	if r {
		return "Required"
	} else {
		return "Optional"
	}
}

func getDescription(d map[string]string) string {
	return ""
	// TODO: description too long, need optimize for display
	//if d == nil {
	//	return ""
	//}
	//if v, ok := d[i18n.GetLanguage()]; ok {
	//	return v
	//} else {
	//	return ""
	//}
}

//
//func (a *Helper) printCompactList() {
//	for _, s := range compactList {
//		product, _ := c.products.GetProduct(s)
//		c.PrintProduct(product)
//	}
//	cli.Printf("  ... ")
//}
