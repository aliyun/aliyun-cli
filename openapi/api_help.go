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
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/newmeta"
)

type apiHelpPlan struct {
	deliverBuiltInApiHelp bool
	deliverPluginHelp     bool
	abortErr              error
}

func getPluginArgsForApiHelp(productCode, apiName string) []string {
	return ensurePluginHelpArgs(productCode, []string{productCode, apiName})
}

func (c *Commando) planApiLevelHelp(productCode, apiName string) (apiHelpPlan, meta.Product, meta.Api, string, bool) {
	pluginName, hasPlugin := c.lookupPluginForProduct(productCode)
	isInstalled := hasPlugin && c.isPluginInstalledForProduct(productCode)

	product, inLibrary := c.library.GetProduct(productCode)
	hasBuiltInApis := inLibrary && productHasBuiltinApis(product)
	plan := apiHelpPlan{}

	if !inLibrary && !hasPlugin {
		plan.abortErr = &InvalidProductOrPluginError{
			Code: productCode, library: c.library, plugins: c.pluginIndex.Plugins,
		}
		return plan, product, meta.Api{}, pluginName, false
	}
	// shouldn't happen
	// if inLibrary && !hasPlugin {}

	// Product listed in built-in catalog without APIs, or plugin-only product → plugin path.
	if !hasBuiltInApis && hasPlugin {
		if !isInstalled {
			plan.abortErr = fmt.Errorf(
				"Command help for '%s %s' requires the product plugin '%s'.\n  aliyun plugin install --name %s\n  After installation, run `aliyun %s %s --help` to view command help.",
				productCode, apiName, pluginName, pluginName, strings.ToLower(productCode), apiName)
			return plan, product, meta.Api{}, pluginName, false
		}
		plan.deliverPluginHelp = true
		return plan, product, meta.Api{}, pluginName, false
	}

	api, apiFound := c.library.GetApi(productCode, product.Version, apiName)
	if apiFound {
		plan.deliverBuiltInApiHelp = true
		return plan, product, api, pluginName, hasBuiltInApis
	}

	localPlugin := c.getInstalledLocalPlugin(productCode)
	if isInstalled && pluginCmdMatches(apiName, localPlugin) {
		plan.deliverPluginHelp = true
		return plan, product, meta.Api{}, pluginName, hasBuiltInApis
	}

	plan.abortErr = newApiOrCmdNotFoundError(&product, apiName, localPlugin, pluginName)
	return plan, product, meta.Api{}, pluginName, hasBuiltInApis
}

func (c *Commando) executePluginApiHelp(ctx *cli.Context, productCode string, pluginArgs []string) (bool, error) {
	c.setLangEnv(ctx)
	ok, err := plugin.ExecutePlugin(productCode, pluginArgs, ctx)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (c *Commando) printBuiltInApiUsage(ctx *cli.Context, product meta.Product, api meta.Api, apiName string) error {
	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), product.Code)

	if product.ApiStyle == "restful" {
		cli.Printf(ctx.Stdout(), "\nProduct:     %s (%s)\n", product.Code, productName)
		cli.Printf(ctx.Stdout(), "Method:      %s\n", api.Method)
		cli.Printf(ctx.Stdout(), "PathPattern: %s\n", api.PathPattern)
	} else {
		cli.Printf(ctx.Stdout(), "\nProduct: %s (%s)\n", product.Code, productName)
	}

	cli.Printf(ctx.Stdout(), "\nParameters:\n")

	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	detail, _ := newmeta.GetAPIDetail(i18n.GetLanguage(), product.Code, apiName)
	printParameters(w, api.Parameters, "", detail)
	w.Flush()

	return nil
}

func (c *Commando) renderApiLevelHelp(ctx *cli.Context, productCode, apiName string, pluginArgs []string) error {
	c.loadPlugins()
	c.printHelpContextHints(ctx)

	plan, product, api, _, hasBuiltIn := c.planApiLevelHelp(productCode, apiName)
	if plan.abortErr != nil {
		return plan.abortErr
	}

	pluginArgs = ensurePluginHelpArgs(productCode, pluginArgs)

	if plan.deliverPluginHelp {
		executed, err := c.executePluginApiHelp(ctx, productCode, pluginArgs)
		if err != nil {
			if !hasBuiltIn {
				return nil
			}
		} else if executed {
			return nil
		} else if !hasBuiltIn {
			return nil
		}
	}

	if plan.deliverBuiltInApiHelp {
		return c.printBuiltInApiUsage(ctx, product, api, apiName)
	}
	return nil
}
