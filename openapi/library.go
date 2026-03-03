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
package openapi

import (
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/newmeta"
)

type Library struct {
	builtinRepo *meta.Repository
	writer      io.Writer
}

func NewLibrary(w io.Writer, lang string) *Library {
	return &Library{
		builtinRepo: meta.LoadRepository(),
		writer:      w,
	}
}

func (a *Library) GetProduct(productCode string) (meta.Product, bool) {
	return a.builtinRepo.GetProduct(productCode)
}

func (a *Library) GetApi(productCode string, version string, apiName string) (meta.Api, bool) {
	return a.builtinRepo.GetApi(productCode, version, apiName)
}

func (a *Library) GetApiByPath(productCode string, version string, method string, path string) (meta.Api, bool) {
	return a.builtinRepo.GetApiByPath(productCode, version, method, path)

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

	// sort products by code
	sort.Slice(a.builtinRepo.Products, func(i, j int) bool {
		return strings.ToLower(a.builtinRepo.Products[i].Code) < strings.ToLower(a.builtinRepo.Products[j].Code)
	})

	for _, product := range a.builtinRepo.Products {
		var productName, _ = newmeta.GetProductName(i18n.GetLanguage(), product.Code)
		cli.PrintfWithColor(w, cli.Cyan, "  %-20s\t%s\n", strings.ToLower(product.Code), productName)
	}
	w.Flush()
}

func (a *Library) PrintProductUsage(productCode string, withApi bool) error {
	product, ok := a.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a}
	}

	if product.ApiStyle == "rpc" {
		cli.Printf(a.writer, "\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", strings.ToLower(product.Code))
	} else {
		cli.Printf(a.writer, "\nUsage 1:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", strings.ToLower(product.Code))
		cli.Printf(a.writer, "\nUsage 2 (For API with NO PARAMS in PathPattern only.):\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ... --body \"...\"\n", strings.ToLower(product.Code))
	}
	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), product.Code)
	cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, productName)
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
				api, _ := newmeta.GetAPI(i18n.GetLanguage(), productCode, apiName)
				if api != nil {
					apiDetail, _ := newmeta.GetAPIDetail(i18n.GetLanguage(), productCode, apiName)
					// use new api metadata
					if api.Deprecated {
						fmt := fmt.Sprintf("  %%-%ds [Deprecated]%%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmt, apiName, api.Summary)
					} else if apiDetail.IsAnonymousAPI() {
						fmt := fmt.Sprintf("  %%-%ds [Anonymous]%%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmt, apiName, api.Summary)
					} else {
						fmt := fmt.Sprintf("  %%-%ds %%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmt, apiName, api.Summary)
					}
				} else {
					cli.PrintfWithColor(a.writer, cli.Green, "  %s\n", apiName)
				}
			}
		}
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

	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), productCode)

	if product.ApiStyle == "restful" {
		cli.Printf(a.writer, "\nProduct:     %s (%s)\n", product.Code, productName)
		cli.Printf(a.writer, "Method:      %s\n", api.Method)
		cli.Printf(a.writer, "PathPattern: %s\n", api.PathPattern)
	} else {
		cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, productName)
	}

	cli.Printf(a.writer, "\nParameters:\n")

	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	detail, _ := newmeta.GetAPIDetail(i18n.GetLanguage(), productCode, apiName)
	printParameters(w, api.Parameters, "", detail)
	w.Flush()

	return nil
}

func printParameters(w io.Writer, params []meta.Parameter, prefix string, detail *newmeta.APIDetail) {

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
				printParameters(w, param.SubParameters, prefix+param.Name+".n.", detail)
			} else {
				fmt.Fprintf(w, "  --%s%s.n\t%s\t%s\n\n", cli.Colorized(cli.BBlack, prefix), cli.Colorized(cli.BBlack, param.Name), param.Type, required(param.Required))
				displayDescription(w, getDescription(detail, param.Name))
			}
		} else {
			fmt.Fprintf(w, "  --%s%s\t%s\t%s\n\n", cli.Colorized(cli.BBlack, prefix), cli.Colorized(cli.BBlack, param.Name), param.Type, required(param.Required))
			displayDescription(w, getDescription(detail, param.Name))
		}
	}
}

func displayDescription(w io.Writer, desc string) {
	lines := strings.Split(desc, "\n")
	for _, v := range lines {
		fmt.Fprintf(w, "  %s\n", v)
	}
	fmt.Fprintf(w, "\n")
}

func required(r bool) string {
	if r {
		return "Required"
	} else {
		return "Optional"
	}
}

func getDescription(detail *newmeta.APIDetail, name string) string {
	if detail == nil {
		return ""
	}
	for _, p := range detail.Parameters {
		if name == p.Name {
			return strings.TrimSpace(p.Description)
		}
	}

	return ""
}
