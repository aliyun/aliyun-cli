/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"strings"
	"github.com/aliyun/aliyun-cli/meta"
	"text/tabwriter"
	"os"
	"github.com/aliyun/aliyun-cli/cli"
)

var compactList = []string {"Ecs", "Rds", "Vpc", "Slb", "Dm", "Ots", "Ess", "Ocs", "CloudApi"}

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
		fmt.Fprintf(w, "  %s\t%s\t%s\n", product.Name, product.Descriptions["en"] + " ", product.Links["en"] + " ")
	}
	w.Flush()
}

func (a *Helper) PrintProductUsage(productName string) {
	product, ok := a.library.GetProduct(productName)
	if !ok {
		cli.Errorf("unknown product %s", productName)
		return
	}

	if strings.ToLower(product.ApiStyle) == "rpc" || product.ApiStyle == "" {
		fmt.Printf("\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", product.Name)
	} else {
		fmt.Printf("\nUsage:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", product.Name)
	}
	fmt.Printf("\nAvailable Api List: ")

	fmt.Printf("\nProduct: %s (%s)\n", product.Name, product.Descriptions["zh"])
	fmt.Printf("Version: %s \n", product.Version)
	if link, ok := product.Links["zh"]; ok {
		fmt.Printf("Link: %s\n", link)
	}

	// w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	for _, apiName := range product.ApiNames {
		fmt.Printf( "  %s\n", apiName)
	}

	fmt.Printf("\nRun `aliyun help %s <ApiName>` to get more information about api", product.Name)
}

func (a *Helper) PrintApiUsage(productName string, apiName string) {
	product, ok := a.library.GetProduct(productName)
	if !ok {
		cli.Errorf("unknown product %s", productName)
		return
	}
	api, ok := a.library.GetApi(productName, product.Version, apiName)
	if !ok {
		cli.Errorf("unknown api: %s/%s/%s", product.Name, product.Version, apiName)
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
	fmt.Printf("  %s(%s)\t%s\t%s\n", product.Name, product.Version, product.Descriptions["zh"], product.Links["zh"])
}

