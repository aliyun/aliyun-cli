/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/meta"
	"text/tabwriter"
	"os"
	"github.com/aliyun/aliyun-cli/cli"
)

// var compactList = []string {"Ecs", "Rds", "Vpc", "Slb", "Dm", "Ots", "Ess", "Ocs", "CloudApi"}

type Helper struct {
	language string
	library *meta.Library
}

func NewHelper(library *meta.Library) (*Helper) {
	return &Helper {
		library: library,
	}
}

func (a *Helper) PrintProducts() {
	fmt.Printf("\nProducts:\n")
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, product := range a.library.Products {
		fmt.Fprintf(w, "  %s\t%s\t%s\n", product.Code, product.Name["en"] + " ", product.GetDocumentLink("zh") + " ")
	}
	w.Flush()
}

func (a *Helper) PrintProductUsage(productName string) {
	product, ok := a.library.GetProduct(productName)
	if !ok {
		cli.Errorf("unknown product %s", productName)
		return
	}

	if product.ApiStyle == "rpc" {
		fmt.Printf("\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", product.Code)
	} else {
		fmt.Printf("\nUsage:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", product.Code)
	}
	fmt.Printf("\nAvailable Api List: ")

	fmt.Printf("\nProduct: %s (%s)\n", product.Code, product.Name["zh"])
	fmt.Printf("Version: %s \n", product.Version)

	fmt.Printf("Link: %s\n", product.GetDocumentLink("zh"))

	// w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, apiName := range product.ApiNames {
		fmt.Printf( "  %s\n", apiName)
	}

	fmt.Printf("\nRun `aliyun help %s <ApiName>` to get more information about api", product.Code)
}

func (a *Helper) PrintApiUsage(productName string, apiName string) {
	product, ok := a.library.GetProduct(productName)
	if !ok {
		cli.Errorf("unknown product %s", productName)
		return
	}
	api, ok := a.library.GetApi(productName, product.Version, apiName)
	if !ok {
		cli.Errorf("unknown api: %s/%s/%s", product.Code, product.Version, apiName)
		return
	}

	fmt.Printf("\nParameters:\n")
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, param := range api.Parameters {
		if param.Required {
			fmt.Fprintf(w,"  --%s\t%s\t%s\n", param.Name, param.Type, "Required")
		} else {
			fmt.Fprintf(w,"  --%s\t%s\t%s\n", param.Name, param.Type, "Optional")
		}
	}
	w.Flush()
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

