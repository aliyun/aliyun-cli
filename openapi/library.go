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
	"io/fs"
	"sort"
	"strings"
	"text/tabwriter"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/v3/aliyun-openapi-meta"
	"github.com/aliyun/aliyun-cli/v3/canonicalmeta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
)

type Library struct {
	builtinRepo   *meta.Repository
	canonicalRepo canonicalAPIRepository
	writer        io.Writer
}

type canonicalAPIRepository interface {
	GetAPI(productCode, version, apiName string) (*canonicalmeta.API, error)
	GetAPIByPath(productCode, version, method, path string, apiNames []string) (*canonicalmeta.API, error)
}

func NewLibrary(w io.Writer, lang string) *Library {
	lib := &Library{
		builtinRepo: meta.LoadRepository(),
		writer:      w,
	}
	if _, err := fs.ReadDir(aliyunopenapimeta.Metadatas, "canonical"); err == nil {
		lib.canonicalRepo = canonicalmeta.NewRepository(aliyunopenapimeta.Metadatas)
	}
	return lib
}

func (a *Library) GetProduct(productCode string) (meta.Product, bool) {
	return a.builtinRepo.GetProduct(productCode)
}

func (a *Library) GetApi(productCode string, version string, apiName string) (*canonicalmeta.API, bool) {
	if a.canonicalRepo != nil {
		if canonical, err := a.canonicalRepo.GetAPI(productCode, version, apiName); err == nil {
			return canonical, true
		}
	}
	return nil, false
}

// GetCanonicalApi returns the canonical API directly, without bridge conversion.
// Returns nil if canonical data is not available.
func (a *Library) GetCanonicalApi(productCode string, version string, apiName string) *canonicalmeta.API {
	if canonical, ok := a.GetApi(productCode, version, apiName); ok {
		return canonical
	}
	return nil
}

func (a *Library) GetCanonicalApiByPath(productCode string, version string, method string, path string) *canonicalmeta.API {
	if canonical, ok := a.GetApiByPath(productCode, version, method, path); ok {
		return canonical
	}
	return nil
}

func (a *Library) GetApiByPath(productCode string, version string, method string, path string) (*canonicalmeta.API, bool) {
	if a.canonicalRepo != nil {
		product, ok := a.builtinRepo.GetProduct(productCode)
		if !ok {
			return nil, false
		}
		if canonical, err := a.canonicalRepo.GetAPIByPath(product.Code, version, method, path, product.ApiNames); err == nil {
			return canonical, true
		}
	}
	return nil, false
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

	sort.Slice(a.builtinRepo.Products, func(i, j int) bool {
		return strings.ToLower(a.builtinRepo.Products[i].Code) < strings.ToLower(a.builtinRepo.Products[j].Code)
	})

	for _, product := range a.builtinRepo.Products {
		productName := getProductDisplayName(product)
		cli.PrintfWithColor(w, cli.Cyan, "  %-20s\t%s\n", strings.ToLower(product.Code), productName)
	}
	w.Flush()
}

func getProductDisplayName(product meta.Product) string {
	if product.Name != nil {
		lang := i18n.GetLanguage()
		if name, ok := product.Name[lang]; ok && name != "" {
			return name
		}
		for _, name := range product.Name {
			if name != "" {
				return name
			}
		}
	}
	return ""
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
	productName := getProductDisplayName(product)
	cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, productName)
	cli.Printf(a.writer, "Version: %s \n", product.Version)

	if withApi {
		if len(product.ApiNames) > 0 {
			cli.PrintfWithColor(a.writer, cli.ColorOff, "\nAvailable Api List: \n")
		}
		maxNameLen := 0

		for _, apiName := range product.ApiNames {
			if len(apiName) > maxNameLen {
				maxNameLen = len(apiName)
			}
		}

		for _, apiName := range product.ApiNames {
			if product.ApiStyle == "restful" {
				api := a.GetCanonicalApi(productCode, product.Version, apiName)
				if api == nil {
					continue
				}
				summary := api.Description(i18n.GetLanguage())
				if summary != "" {
					if api.Deprecated {
						summary = "[Deprecated]" + summary
					}
					ptn := fmt.Sprintf("  %%-%ds : %%s %%s  %%s\n", maxNameLen+1)
					cli.PrintfWithColor(a.writer, cli.Green, ptn, apiName, api.Method, api.PathPattern, summary)
				} else {
					ptn := fmt.Sprintf("  %%-%ds : %%s %%s\n", maxNameLen+1)
					cli.PrintfWithColor(a.writer, cli.Green, ptn, apiName, api.Method, api.PathPattern)
				}
			} else {
				summary := ""
				deprecated := false
				anonymous := false

				if a.canonicalRepo != nil {
					if canonical, err := a.canonicalRepo.GetAPI(productCode, product.Version, apiName); err == nil {
						summary = canonical.Description(i18n.GetLanguage())
						deprecated = canonical.Deprecated
						anonymous = canonical.IsAnonymous()
					}
				}

				if summary != "" {
					if deprecated {
						fmtStr := fmt.Sprintf("  %%-%ds [Deprecated]%%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmtStr, apiName, summary)
					} else if anonymous {
						fmtStr := fmt.Sprintf("  %%-%ds [Anonymous]%%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmtStr, apiName, summary)
					} else {
						fmtStr := fmt.Sprintf("  %%-%ds %%s\n", maxNameLen+1)
						cli.PrintfWithColor(a.writer, cli.Green, fmtStr, apiName, summary)
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

	api := a.GetCanonicalApi(productCode, product.Version, apiName)
	if api == nil {
		return &InvalidApiError{Name: apiName, product: &product}
	}

	productName := getProductDisplayName(product)

	if product.ApiStyle == "restful" {
		cli.Printf(a.writer, "\nProduct:     %s (%s)\n", product.Code, productName)
		cli.Printf(a.writer, "Method:      %s\n", api.Method)
		cli.Printf(a.writer, "PathPattern: %s\n", api.PathPattern)
	} else {
		cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, productName)
	}

	cli.Printf(a.writer, "\nParameters:\n")

	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	printCanonicalAPI(w, api, "")
	w.Flush()

	printCanonicalExamples(a.writer, api, product.ApiStyle)

	return nil
}

func printCanonicalExamples(w io.Writer, api *canonicalmeta.API, apiStyle string) {
	if api == nil || (api.KebabExample == "" && api.CamelExample == "") {
		return
	}
	cli.Printf(w, "\nExample:\n")
	if api.KebabExample != "" {
		cli.Printf(w, "  (Recommended) Command Style:\n  %s\n", api.KebabExample)
	}
	if api.CamelExample != "" {
		if apiStyle == "restful" {
			cli.Printf(w, "  RESTful Style:\n  %s\n", api.CamelExample)
		} else {
			cli.Printf(w, "  PascalCase Style:\n  %s\n", api.CamelExample)
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

func printCanonicalAPI(w io.Writer, api *canonicalmeta.API, prefix string) {
	var bodyFields []*canonicalmeta.Parameter
	if api.V1BodyParameters != nil {
		bodyFields = api.LegacyBodyFields()
	}
	printLegacyViews(w, api.LegacyTopLevelParameters(), prefix, bodyFields)
}

func printLegacyViews(w io.Writer, views []*canonicalmeta.LegacyParameterView, prefix string, bodyFields []*canonicalmeta.Parameter) {
	lang := i18n.GetLanguage()

	// Sort on a separate slice copy: required first, then by name.
	sorted := make([]*canonicalmeta.LegacyParameterView, len(views))
	copy(sorted, views)
	sort.SliceStable(sorted, func(i, j int) bool {
		ri, rj := sorted[i].LegacyRequired(), sorted[j].LegacyRequired()
		if ri && rj {
			return sorted[i].LegacyName() < sorted[j].LegacyName()
		}
		if ri {
			return true
		}
		if rj {
			return false
		}
		return sorted[i].LegacyName() < sorted[j].LegacyName()
	})

	for _, v := range sorted {
		pos := v.LegacyPosition()
		if pos == "Domain" || pos == "Header" {
			continue
		}

		name := v.LegacyName()
		displayType := v.LegacyType()

		if v.LegacyHasChildren() {
			children := v.LegacyChildren()
			printLegacyViews(w, children, prefix+name+".n.", nil)
		} else if v.IsLegacyRepeatList() {
			fmt.Fprintf(w, "  --%s%s.n\t%s\t%s\n\n", cli.Colorized(cli.BBlack, prefix), cli.Colorized(cli.BBlack, name), displayType, required(v.LegacyRequired()))
			displayDescription(w, v.LegacyDescription(lang))
		} else {
			fmt.Fprintf(w, "  --%s%s\t%s\t%s\n\n", cli.Colorized(cli.BBlack, prefix), cli.Colorized(cli.BBlack, name), displayType, required(v.LegacyRequired()))
			displayDescription(w, v.LegacyDescription(lang))
			if v.IsTopLevelBody() && len(bodyFields) > 0 {
				printBodyFields(w, bodyFields, lang)
			}
		}
	}
}

// printBodyFields prints the fields carried inside a top-level --body parameter,
// indented one extra level with a "|-" marker to show hierarchy. The marker
// intentionally omits the "--" prefix: these are JSON keys inside --body, not flags.
// Only one level is shown; nested fields are not expanded.
func printBodyFields(w io.Writer, fields []*canonicalmeta.Parameter, lang string) {
	sorted := make([]*canonicalmeta.Parameter, len(fields))
	copy(sorted, fields)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Required != sorted[j].Required {
			return sorted[i].Required
		}
		return sorted[i].RawName < sorted[j].RawName
	})

	for _, p := range sorted {
		fmt.Fprintf(w, "    |- %s\t%s\t%s\n\n", cli.Colorized(cli.BBlack, p.RawName), p.Type, required(p.Required))
		desc := p.DescriptionEn
		if lang == "zh" {
			desc = p.DescriptionZh
		}
		if desc != "" {
			for _, line := range strings.Split(desc, "\n") {
				fmt.Fprintf(w, "      %s\n", line)
			}
			fmt.Fprintf(w, "\n")
		}
	}
}
