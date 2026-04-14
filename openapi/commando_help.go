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
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/newmeta"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/aimode"
)

func (c *Commando) printPluginIndexLoadHint(ctx *cli.Context) {
	if c.pluginIndexErr == nil {
		return
	}
	cli.PrintfWithColor(ctx.Stderr(), cli.Yellow, "%s\n",
		i18n.T("Note: Could not load the remote plugin catalog (network or server). Install-related hints may be incomplete; installed plugins still work.",
			"提示：未能加载远程插件目录（网络或服务器原因），与安装插件相关的提示可能不完整；已安装的插件不受影响。").Text())
}

func (c *Commando) printAiModeHelpHint(ctx *cli.Context) {
	cfg, err := aimode.Load(config.GetConfigDir(ctx))
	if err != nil || !cfg.Enabled {
		return
	}
	msg := i18n.T(
		"Note: CLI AI mode is enabled in configuration; API requests include the AI User-Agent segment. If you do not want this, run `aliyun configure ai-mode disable`.",
		"提示：已在配置中开启 CLI AI 模式，API 请求会带上 AI UA。若不需要，请执行 `aliyun configure ai-mode disable` 关闭。",
	).Text()
	cli.PrintfWithColor(ctx.Stderr(), cli.Yellow, "%s\n", msg)
}

func (c *Commando) printHelpContextHints(ctx *cli.Context) {
	c.printPluginIndexLoadHint(ctx)
	c.printAiModeHelpHint(ctx)
}

func getPluginArgsForHelp(productCode string) []string {
	cmdIndex := -1
	for i, arg := range os.Args {
		if strings.EqualFold(arg, productCode) {
			cmdIndex = i
			break
		}
	}
	if cmdIndex != -1 && cmdIndex < len(os.Args) {
		args := os.Args[cmdIndex:]
		hasHelp := false
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				hasHelp = true
				break
			}
		}
		if !hasHelp {
			args = append(args, "--help")
		}
		return args
	}
	return []string{productCode, "--help"}
}

func (c *Commando) printProducts(ctx *cli.Context) {
	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	cli.PrintfWithColor(w, cli.ColorOff, "\nProducts:\n")

	type displayProduct struct {
		Code              string
		Name              string
		IsBuiltIn         bool
		HasPlugin         bool
		IsPluginInstalled bool
		PluginName        string
	}

	// collect all info from both plugin index and built-in
	productMap := make(map[string]*displayProduct)

	// 1. From plugin index: collect plugin products
	if c.pluginIndex != nil {
		for _, pInfo := range c.pluginIndex.Plugins {
			pluginName := pInfo.Name
			productCode := pInfo.ProductCode
			lowerCode := strings.ToLower(productCode)

			isInstalled := false
			if c.localManifest != nil {
				_, isInstalled = c.localManifest.Plugins[pluginName]
			}

			desc := pInfo.Description
			if pInfo.ProductName != nil {
				lang := i18n.GetLanguage()
				if name, ok := pInfo.ProductName[lang]; ok && name != "" {
					desc = name
				}
			}
			if desc == "" {
				desc = productCode
			}

			productMap[lowerCode] = &displayProduct{
				Code:              productCode,
				Name:              desc,
				HasPlugin:         true,
				IsPluginInstalled: isInstalled,
				PluginName:        pluginName,
			}
		}
	}

	// 2. From built-in: add or merge with existing
	for _, product := range c.library.builtinRepo.Products {
		lowerCode := strings.ToLower(product.Code)
		productName, _ := newmeta.GetProductName(i18n.GetLanguage(), product.Code)

		if p, ok := productMap[lowerCode]; ok {
			// Already from plugin index, add built-in info, for now it should always to be true
			p.IsBuiltIn = true
		} else {
			productMap[lowerCode] = &displayProduct{
				Code:      product.Code,
				Name:      productName,
				IsBuiltIn: true,
			}
		}
	}

	// 3. Convert to slice and sort
	displayProducts := make([]*displayProduct, 0, len(productMap))
	for _, p := range productMap {
		displayProducts = append(displayProducts, p)
	}
	sort.Slice(displayProducts, func(i, j int) bool {
		return strings.ToLower(displayProducts[i].Code) < strings.ToLower(displayProducts[j].Code)
	})

	// 4. Print with appropriate display
	var uninstalledPlugins []string
	for _, p := range displayProducts {
		displayName := p.Name
		switch {
		case p.IsBuiltIn && p.HasPlugin && p.IsPluginInstalled:
			displayName = fmt.Sprintf("%s (Plugin: %s)", displayName, p.PluginName)
		case p.IsBuiltIn && p.HasPlugin && !p.IsPluginInstalled:
			displayName = fmt.Sprintf("%s (Plugin available but not installed: %s)", displayName, p.PluginName)
			uninstalledPlugins = append(uninstalledPlugins, p.PluginName)
		case !p.IsBuiltIn && p.HasPlugin && !p.IsPluginInstalled:
			displayName = fmt.Sprintf("%s (Plugin: %s, Not Installed)", displayName, p.PluginName)
			uninstalledPlugins = append(uninstalledPlugins, p.PluginName)
		case !p.IsBuiltIn && p.HasPlugin && p.IsPluginInstalled:
			displayName = fmt.Sprintf("%s (Plugin: %s)", displayName, p.PluginName)
		}
		cli.PrintfWithColor(w, cli.Cyan, "  %-20s\t%s\n", strings.ToLower(p.Code), displayName)
	}
	w.Flush()

	// 5. Print installation hint for uninstalled plugins
	if len(uninstalledPlugins) > 0 {
		cli.PrintfWithColor(ctx.Stdout(), cli.ColorOff, "\nTo install plugins for uninstalled products, run:\n")
		cli.PrintfWithColor(ctx.Stdout(), cli.Green, "  aliyun plugin install --names <plugin_name> [--enable-pre]\n")
	}
	c.printHelpContextHints(ctx)
}

func (c *Commando) printProductUsage(ctx *cli.Context, productCode string) error {
	c.printHelpContextHints(ctx)
	// 1. Check if it's a plugin product
	var pluginName string
	var isInstalled bool

	if c.pluginIndex != nil {
		for _, pInfo := range c.pluginIndex.Plugins {
			name := pInfo.Name
			// aliyun-cli-<lower-product>
			pCode := pInfo.ProductCode
			if strings.EqualFold(pCode, productCode) {
				pluginName = name
				break
			}
		}
	}
	// Locally installed plugin (e.g. plugin install --source) not in remote index
	if pluginName == "" && c.localManifest != nil {
		if n, _, ok := plugin.FindInstalledPluginInManifest(c.localManifest, productCode); ok {
			pluginName = n
		}
	}

	// 2. Check if it's a built-in product
	product, ok := c.library.GetProduct(productCode)
	if !ok {
		// If not a built-in product, but is a valid plugin product
		if pluginName != "" {
			if c.localManifest != nil {
				_, isInstalled = c.localManifest.Plugins[pluginName]
			}
			if isInstalled {
				cli.Printf(ctx.Stdout(), "Product '%s' is provided by plugin '%s'\n", productCode, pluginName)
				c.setLangEnv(ctx)
				plugin.ExecutePlugin(productCode, getPluginArgsForHelp(productCode), ctx)
				return nil
			} else {
				return fmt.Errorf("'%s' is not a valid product.\nDid you mean to install corresponding product plugin?\n  aliyun plugin install --names %s", productCode, pluginName)
			}
		}
		var plugList []plugin.PluginInfo
		if c.pluginIndex != nil {
			plugList = c.pluginIndex.Plugins
		}
		return &InvalidProductOrPluginError{Code: productCode, library: c.library, plugins: plugList}
	}

	if pluginName != "" {
		if c.localManifest != nil {
			_, isInstalled = c.localManifest.Plugins[pluginName]
		}

		if isInstalled {
			// Built-in product AND installed plugin
			if os.Getenv("ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP") == "true" {
				// Show built-in help (fall through)
			} else {
				cli.Printf(ctx.Stdout(), "Note: The help information for product '%s' is provided by the installed plugin '%s'.\n", productCode, pluginName)
				cli.Printf(ctx.Stdout(), "To view legacy built-in help, set ALIBABA_CLOUD_ORIGINAL_PRODUCT_HELP=true\n")
				c.setLangEnv(ctx)
				plugin.ExecutePlugin(productCode, getPluginArgsForHelp(productCode), ctx)
				return nil
			}
		} else {
			cli.Printf(ctx.Stdout(), "\n[Suggestion] A dedicated product plugin is available for '%s'.\n", productCode)
			cli.Printf(ctx.Stdout(), "Run 'aliyun plugin install --names %s' to install it for enhanced features.\n\n", pluginName)
		}
	}

	if product.ApiStyle == "rpc" {
		cli.Printf(ctx.Stdout(), "\nUsage:\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ...\n", strings.ToLower(product.Code))
	} else {
		cli.Printf(ctx.Stdout(), "\nUsage 1:\n  aliyun %s [GET|PUT|POST|DELETE] <PathPattern> --body \"...\" \n", strings.ToLower(product.Code))
		cli.Printf(ctx.Stdout(), "\nUsage 2 (For API with NO PARAMS in PathPattern only.):\n  aliyun %s <ApiName> --parameter1 value1 --parameter2 value2 ... --body \"...\"\n", strings.ToLower(product.Code))
	}
	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), product.Code)
	cli.Printf(ctx.Stdout(), "\nProduct: %s (%s)\n", product.Code, productName)
	cli.Printf(ctx.Stdout(), "Version: %s \n", product.Version)

	if len(product.ApiNames) > 0 {
		cli.PrintfWithColor(ctx.Stdout(), cli.ColorOff, "\nAvailable Api List: \n")
	}

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
				// use new api metadata
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

func (c *Commando) printApiUsage(ctx *cli.Context, productCode string, apiName string) error {
	c.printHelpContextHints(ctx)
	// 0. Check if it's a plugin product
	var pluginName string
	var isInstalled bool
	var localPlugin plugin.LocalPlugin

	if c.pluginIndex != nil {
		for _, pInfo := range c.pluginIndex.Plugins {
			name := pInfo.Name
			pCode := pInfo.ProductCode
			if strings.EqualFold(pCode, productCode) {
				pluginName = name
				break
			}
		}
	}
	if pluginName == "" && c.localManifest != nil {
		if n, _, ok := plugin.FindInstalledPluginInManifest(c.localManifest, productCode); ok {
			pluginName = n
		}
	}

	if pluginName != "" && c.localManifest != nil {
		if lp, ok := c.localManifest.Plugins[pluginName]; ok {
			localPlugin = lp
			isInstalled = true
		}
	}

	// 1. Try to get built-in product info
	product, ok := c.library.builtinRepo.GetProduct(productCode)

	// Case A: Not a built-in product
	if !ok {
		if pluginName != "" { // should not happen, might be true in the future when plugin product has more than built-in product
			if isInstalled {
				cli.Printf(ctx.Stdout(), "Command '%s %s' is provided by plugin '%s'.\n", productCode, apiName, pluginName)
				c.setLangEnv(ctx)
				plugin.ExecutePlugin(productCode, getPluginArgsForHelp(productCode), ctx)
				return nil
			} else {
				return fmt.Errorf("'%s' is not a valid product.\nDid you mean to install corresponding product plugin?\n  aliyun plugin install --names %s", productCode, pluginName)
			}
		}
		// Not a built-in product and not a plugin product, like fuzzy cmd input
		var plugList []plugin.PluginInfo
		if c.pluginIndex != nil {
			plugList = c.pluginIndex.Plugins
		}
		return &InvalidProductOrPluginError{Code: productCode, library: c.library, plugins: plugList}
	}

	shouldTryPlugin := false
	if apiName == strings.ToLower(apiName) { // fuzzy cmd input should apply different logic for build-in and plugin product
		shouldTryPlugin = true
	}

	// Case B: Built-in product exists
	api, ok := c.library.builtinRepo.GetApi(productCode, product.Version, apiName)
	if !ok {
		// API not found in built-in metadata. api in plugin is different from api from built-in
		if pluginName != "" {
			if shouldTryPlugin { // 全小写进入插件执行及智能纠错系统， 未安装则提示安装
				if isInstalled {
					c.setLangEnv(ctx)
					plugin.ExecutePlugin(productCode, getPluginArgsForHelp(productCode), ctx)
					return nil
				} else {
					return fmt.Errorf("'%s' is not a valid built-in command.\nA plugin '%s' is available which might support this command.\nRun 'aliyun plugin install --names %s' to install it.", apiName, pluginName, pluginName)
				}
			} else { // 非插件命令形式，如果本地插件安装了，则返回插件帮助; 如果未安装，则打印插件提示信息，并继续原有api纠错系统
				if isInstalled {
					return &InvalidUnifiedApiError{Name: apiName, product: &product, lPlugin: localPlugin}
				} else {
					cli.Printf(ctx.Stdout(), "\n[Suggestion] A dedicated product plugin is available for '%s'.\n", productCode)
					cli.Printf(ctx.Stdout(), "Run 'aliyun plugin install --names %s' to install it for enhanced features.\n\n", pluginName)
				}
			}
		}
		return &InvalidApiError{Name: apiName, product: &product}
	}

	productName, _ := newmeta.GetProductName(i18n.GetLanguage(), productCode)

	if product.ApiStyle == "restful" {
		cli.Printf(ctx.Stdout(), "\nProduct:     %s (%s)\n", product.Code, productName)
		cli.Printf(ctx.Stdout(), "Method:      %s\n", api.Method)
		cli.Printf(ctx.Stdout(), "PathPattern: %s\n", api.PathPattern)
	} else {
		cli.Printf(ctx.Stdout(), "\nProduct: %s (%s)\n", product.Code, productName)
	}

	cli.Printf(ctx.Stdout(), "\nParameters:\n")

	w := tabwriter.NewWriter(ctx.Stdout(), 8, 0, 1, ' ', 0)
	detail, _ := newmeta.GetAPIDetail(i18n.GetLanguage(), productCode, apiName)
	printParameters(w, api.Parameters, "", detail)
	w.Flush()

	return nil
}
