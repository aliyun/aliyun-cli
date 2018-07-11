/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/resource"
	"io"
	"strings"
	"text/tabwriter"
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

func (a *Library) GetProducts() []meta.Product {
	return a.builtinRepo.Products
}

func (a *Library) PrintProducts() {
	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	cli.PrintfWithColor(w, cli.ColorOff,"\nProducts:\n")
	for _, product := range a.builtinRepo.Products {
		cli.PrintfWithColor(w, cli.Cyan,"  %s\t%s\n", strings.ToLower(product.Code), product.Name[i18n.GetLanguage()])
	}
	w.Flush()
}

func (a *Library) printProduct(product meta.Product) {
	cli.Printf(a.writer, "  %s(%s)\t%s\t%s\n", product.Code, product.Version, product.Name["zh"],
		product.GetDocumentLink("zh"))
}

func (a *Library) PrintProductUsage(productCode string, withApi bool) error {
	product, ok := a.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a}
	}

	if product.ApiStyle == "rpc" {
		cli.Printf(a.writer, "\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", product.Code)
	} else {
		withApi = false
		cli.Printf(a.writer, "\nUsage:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", product.Code)
	}

	cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	cli.Printf(a.writer, "Version: %s \n", product.Version)
	cli.Printf(a.writer, "Link: %s\n", product.GetDocumentLink(i18n.GetLanguage()))

	if withApi {
		cli.PrintfWithColor(a.writer, cli.ColorOff,"\nAvailable Api List: \n")
		for _, apiName := range product.ApiNames {
			cli.PrintfWithColor(a.writer, cli.Green,"  %s\n", apiName)
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

	cli.Printf(a.writer, "\nRun `aliyun %s <ApiName> --help` to get more information about api\n", product.GetLowerCode())
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

	cli.Printf(a.writer, "\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	// cli.Printf("Api: %s %s\n", api.Name, api.Description[i18n.GetLanguage()])
	cli.Printf(a.writer, "Link:    %s\n", api.GetDocumentLink())
	cli.Printf(a.writer, "\nParameters:\n")

	w := tabwriter.NewWriter(a.writer, 8, 0, 1, ' ', 0)
	printParameters(w, api.Parameters, "")
	w.Flush()

	return nil
}

func printParameters(w io.Writer, params []meta.Parameter, prefix string) {
	for _, param := range params {
		if param.Hidden {
			continue
		}
		if len(param.SubParameters) > 0 {
			printParameters(w, param.SubParameters, param.Name+".n.")
			//for _, sp := range param.SubParameters {
			//	fmt.Fprintf(w,"  --%s.n.%s\t%s\t%s\n", param.Name, sp.Name, sp.Type, required(sp.Required))
			//}
		} else if param.Type == "RepeatList" {
			fmt.Fprintf(w, "  --%s%s.n\t%s\t%s\t%s\n", prefix, param.Name, param.Type, required(param.Required), getDescription(param.Description))
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
