/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"io"
	"os"
	"strings"
	"text/tabwriter"
)

// var compactList = []string {"Ecs", "Rds", "Vpc", "Slb", "Dm", "Ots", "Ess", "Ocs", "CloudApi"}

type Helper struct {
	language string
	library  *meta.Library
}

func NewHelper(library *meta.Library) *Helper {
	return &Helper{
		library: library,
	}
}

func (a *Helper) PrintProducts() {
	fmt.Printf("\nProducts:\n")
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, product := range a.library.Products {
		fmt.Fprintf(w, "  %s\t%s\n", strings.ToLower(product.Code), product.Name[i18n.GetLanguage()])
	}
	w.Flush()
}

func (a *Helper) PrintProductUsage(productCode string, withApi bool) error {
	product, ok := a.library.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a.library}
	}

	if product.ApiStyle == "rpc" {
		fmt.Printf("\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", product.Code)
	} else {
		withApi = false
		fmt.Printf("\nUsage:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", product.Code)
	}

	fmt.Printf("\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	fmt.Printf("Version: %s \n", product.Version)
	fmt.Printf("Link: %s\n", product.GetDocumentLink(i18n.GetLanguage()))

	if withApi {
		fmt.Printf("\nAvailable Api List: \n")
		for _, apiName := range product.ApiNames {
			fmt.Printf("  %s\n", apiName)
		}
		// TODO some ApiName is too long, two column not seems good
		//w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
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

	fmt.Printf("\nRun `aliyun help %s <ApiName>` to get more information about api", product.GetLowerCode())
	return nil
}

func (a *Helper) PrintApiUsage(productCode string, apiName string) error {
	product, ok := a.library.GetProduct(productCode)
	if !ok {
		return &InvalidProductError{Code: productCode, library: a.library}
	}
	api, ok := a.library.GetApi(productCode, product.Version, apiName)
	if !ok {
		return &InvalidApiError{Name: apiName, product: &product}
	}

	fmt.Printf("\nProduct: %s (%s)\n", product.Code, product.Name[i18n.GetLanguage()])
	// fmt.Printf("Api: %s %s\n", api.Name, api.Description[i18n.GetLanguage()])
	fmt.Printf("Link:    %s\n", api.GetDocumentLink())
	fmt.Printf("\nParameters:\n")

	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
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
			fmt.Fprintf(w, "  --%s%s.n\t%s\t%s\n", prefix, param.Name, param.Type, required(param.Required))
		} else {
			fmt.Fprintf(w, "  --%s%s\t%s\t%s\n", prefix, param.Name, param.Type, required(param.Required))
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

//
//func (a *Helper) printCompactList() {
//	for _, s := range compactList {
//		product, _ := c.products.GetProduct(s)
//		c.PrintProduct(product)
//	}
//	fmt.Printf("  ... ")
//}

func (a *Helper) printProduct(product meta.Product) {
	fmt.Printf("  %s(%s)\t%s\t%s\n", product.Code, product.Version, product.Name["zh"],
		product.GetDocumentLink("zh"))
}
