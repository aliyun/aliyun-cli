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
	"os"
	"strings"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"
	"github.com/aliyun/aliyun-cli/v3/newmeta"
)

type productHelpPlan struct {
	deliverPluginHelp  bool
	deliverBuiltInHelp bool
	showInstallHint    bool
	abortErr           error
}

func showOriginalProductHelp() bool {
	v := os.Getenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP")
	return v == "true" || v == "1"
}

func pluginArgsFromOS(productCode string, fallback []string) []string {
	cmdIndex := -1
	for i, arg := range os.Args {
		if strings.EqualFold(arg, productCode) {
			cmdIndex = i
			break
		}
	}
	if cmdIndex != -1 && cmdIndex < len(os.Args) {
		return os.Args[cmdIndex:]
	}
	return fallback
}

func (c *Commando) lookupPluginForProduct(productCode string) (pluginName string, found bool) {
	for _, pInfo := range c.pluginIndex.Plugins {
		if strings.EqualFold(pInfo.ProductCode, productCode) {
			return pInfo.Name, true
		}
	}
	if n, _, ok := plugin.FindInstalledPluginInManifest(c.localManifest, productCode); ok {
		return n, true
	}
	return "", false
}

func (c *Commando) isPluginInstalledForProduct(productCode string) bool {
	_, _, ok := plugin.FindInstalledPluginInManifest(c.localManifest, productCode)
	return ok
}

func (c *Commando) planProductLevelHelp(productCode string) (productHelpPlan, meta.Product, string, bool) {
	pluginName, hasPlugin := c.lookupPluginForProduct(productCode)
	isInstalled := hasPlugin && c.isPluginInstalledForProduct(productCode)

	product, hasBuiltIn := c.library.GetProduct(productCode)
	showOriginal := showOriginalProductHelp()

	plan := productHelpPlan{}

	if !hasBuiltIn && !hasPlugin {
		plan.abortErr = &InvalidProductOrPluginError{
			Code: productCode, library: c.library, plugins: c.pluginIndex.Plugins,
		}
		return plan, product, pluginName, false
	}

	if !hasBuiltIn && hasPlugin {
		if isInstalled {
			plan.deliverPluginHelp = true
			return plan, product, pluginName, false
		}
		plan.abortErr = fmt.Errorf(
			"'%s' is not a built-in product and requires an external product plugin.\n  aliyun plugin install --name %s",
			productCode, pluginName)
		return plan, product, pluginName, false
	}

	// Built-in product exists.
	if hasPlugin && isInstalled && !showOriginal {
		plan.deliverPluginHelp = true
		return plan, product, pluginName, true
	}

	if hasPlugin && !isInstalled {
		plan.showInstallHint = true
	}
	plan.deliverBuiltInHelp = true
	return plan, product, pluginName, true
}

func (c *Commando) printPluginInstallSuggestion(ctx *cli.Context, productCode, pluginName string) {
	cli.PrintfWithColor(ctx.Stdout(), cli.Green,
		"\n[Suggestion] A dedicated product plugin '%s' is available for '%s'.\n", pluginName, productCode)
	cli.PrintfWithColor(ctx.Stdout(), cli.Green,
		"Run 'aliyun plugin install --name %s' to install it for enhanced features.\n\n", pluginName)
}

func (c *Commando) executePluginProductHelp(ctx *cli.Context, productCode, pluginName string, pluginArgs []string, hasBuiltIn bool) (bool, error) {
	if !hasBuiltIn {
		cli.Printf(ctx.Stdout(), "Product '%s' is provided by plugin '%s'\n", productCode, pluginName)
	} else {
		cli.Printf(ctx.Stdout(),
			"Note: The help information for product '%s' is provided by the installed plugin '%s'.\n",
			productCode, pluginName)
		cli.Printf(ctx.Stdout(),
			"To view legacy built-in help, set ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true\n")
	}
	c.setLangEnv(ctx)
	ok, err := plugin.ExecutePlugin(productCode, pluginArgs, ctx)
	if err != nil {
		return false, err
	}
	return ok, nil
}

func (c *Commando) printBuiltInProductUsage(ctx *cli.Context, product meta.Product) error {
	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), product.Code)
	cli.Printf(ctx.Stdout(), "\nProduct: %s (%s)\n", product.Code, productName)
	cli.Printf(ctx.Stdout(), "Version: %s \n", product.Version)
	cli.Printf(ctx.Stdout(), "\n")

	if len(product.ApiNames) == 0 {
		return nil
	}
	if product.ApiStyle == "rpc" {
		cli.Printf(ctx.Stdout(), "\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", strings.ToLower(product.Code))
	} else {
		cli.Printf(ctx.Stdout(), "\nUsage 1:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ... --body \"...\"\n", strings.ToLower(product.Code))
		cli.Printf(ctx.Stdout(), "\nUsage 2:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", strings.ToLower(product.Code))
	}

	cli.PrintfWithColor(ctx.Stdout(), cli.ColorOff, "\nAvailable Api List: \n")

	maxNameLen := 0
	for _, apiName := range product.ApiNames {
		if len(apiName) > maxNameLen {
			maxNameLen = len(apiName)
		}
	}

	for _, apiName := range product.ApiNames {
		if product.ApiStyle == "restful" {
			api, _ := c.library.GetApi(product.Code, product.Version, apiName)
			ptn := fmt.Sprintf("  %%-%ds : %%s %%s\n", maxNameLen+1)
			cli.PrintfWithColor(ctx.Stdout(), cli.Green, ptn, apiName, api.Method, api.PathPattern)
		} else {
			api, _ := newmeta.GetAPI(i18n.GetLanguage(), product.Code, apiName)
			if api != nil {
				apiDetail, _ := newmeta.GetAPIDetail(i18n.GetLanguage(), product.Code, apiName)
				if api.Deprecated {
					fmtStr := fmt.Sprintf("  %%-%ds [Deprecated]%%s\n", maxNameLen+1)
					cli.PrintfWithColor(ctx.Stdout(), cli.Green, fmtStr, apiName, api.Summary)
				} else if apiDetail.IsAnonymousAPI() {
					fmtStr := fmt.Sprintf("  %%-%ds [Anonymous]%%s\n", maxNameLen+1)
					cli.PrintfWithColor(ctx.Stdout(), cli.Green, fmtStr, apiName, api.Summary)
				} else {
					fmtStr := fmt.Sprintf("  %%-%ds %%s\n", maxNameLen+1)
					cli.PrintfWithColor(ctx.Stdout(), cli.Green, fmtStr, apiName, api.Summary)
				}
			} else {
				cli.PrintfWithColor(ctx.Stdout(), cli.Green, "  %s\n", apiName)
			}
		}
	}

	cli.Printf(ctx.Stdout(), "\nRun `aliyun %s <ApiName> --help` to get more information about this API\n", product.GetLowerCode())

	return nil
}

func (c *Commando) renderProductLevelHelp(ctx *cli.Context, productCode string, pluginArgs []string) error {
	c.loadPlugins()
	c.printHelpContextHints(ctx)

	plan, product, pluginName, hasBuiltIn := c.planProductLevelHelp(productCode)
	if plan.abortErr != nil {
		return plan.abortErr
	}

	pluginArgs = ensurePluginHelpArgs(productCode, pluginArgs)

	if plan.deliverPluginHelp {
		executed, err := c.executePluginProductHelp(ctx, productCode, pluginName, pluginArgs, hasBuiltIn)
		if err != nil {
			if !hasBuiltIn {
				return nil
			}
			// Installed plugin delegate failed; fall through to legacy built-in help.
		} else if executed {
			return nil
		} else if !hasBuiltIn {
			return nil
		}
		// Installed plugin delegate failed; fall through to legacy built-in help.
	} else if plan.showInstallHint {
		c.printPluginInstallSuggestion(ctx, productCode, pluginName)
	}

	if plan.deliverBuiltInHelp {
		return c.printBuiltInProductUsage(ctx, product)
	}
	return nil
}

func ensurePluginHelpArgs(productCode string, args []string) []string {
	hasHelp := false
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			hasHelp = true
			break
		}
	}
	if !hasHelp {
		args = append(append([]string{}, args...), "--help")
	}
	return args
}
