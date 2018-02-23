/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package openapi

import (
	"fmt"
	"strings"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/cli"
)

type Caller struct {
	profile *config.Profile
	library *meta.Library
	helper *Helper

	force bool
	verbose bool		// TODO: next version
}

func NewCaller(profile *config.Profile, library *meta.Library) (*Caller) {
	return &Caller {
		profile: profile,
		library: library,
		helper: NewHelper(library),
	}
}

func (c *Caller) Validate() error {
	return c.profile.Validate()
}

//
// entrance call from main
func (c *Caller) Run(ctx *cli.Context, productName string, apiOrMethod string, path string) {
	c.force = ctx.Flags().IsAssigned("force")

	//
	// get product info
	product, ok := c.library.GetProduct(productName)
	if !ok {
		if !c.force {
			ctx.Command().PrintFailed(fmt.Errorf("unknown product: %s", productName),
				"Use\n  `aliyun help`  to view product list\n  or add --force flag to skip name check")
			return
		} else {
			product = meta.Product {
				Code: productName,
			}
		}
	}

	if strings.ToLower(product.ApiStyle) == "rpc" || product.ApiStyle == "" {
		//
		// Rpc call
		if path != "" {
			ctx.Command().PrintFailed(fmt.Errorf("invalid arguments"), "")
			return
		}
		if c.force {
			c.InvokeRpcForce(ctx, &product, apiOrMethod)
		} else {
			c.InvokeRpc(ctx, &product, apiOrMethod)
		}
	} else if product.Version != "" {
		//
		// Restful Call
		// aliyun cs GET /clusters
		// aliyun cs /clusters --roa GET
		ok, method, path, err := CheckRestfulMethod(ctx, apiOrMethod, path)
		if !ok {
			if err != nil {
				ctx.Command().PrintFailed(err, "")
			} else {
				ctx.Command().PrintFailed(fmt.Errorf("product %s need restful call", product.Code), "")
			}
			return
		}
		c.InvokeRestful(ctx, &product, method, path)
		if err != nil {
			ctx.Command().PrintFailed(fmt.Errorf("call restful %s%s.%s faild %v", product.Code, path, method, err), "")
			return
		}
	} else {
		ok, method, path, err := CheckRestfulMethod(ctx, apiOrMethod, path)
		if ok {
			if err != nil {
				ctx.Command().PrintFailed(err, "")
			} else {
				c.InvokeRestful(ctx, &product, method, path)
			}
		} else {
			c.InvokeRpcForce(ctx, &product, apiOrMethod)
		}
	}
}


