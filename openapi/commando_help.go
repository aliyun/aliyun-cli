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
	// Locally installed plugin (e.g. plugin install --package) not in remote index
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

var (
	helpDelegateIsInstalled = plugin.IsPluginInstalled
	helpDelegateExecute     = plugin.ExecutePlugin
)

// tryDelegatePluginHelp is layer-3 of the help hierarchy:
//
//	len(args)==1 → printProductUsage      (this file)
//	len(args)==2 → printApiUsage          (this file)
//	len(args)>2  → tryDelegatePluginHelp  (this file)

// OpenAPI APIs are conventionally PascalCase (DescribeRegions, GetAlias)
// and RESTful invocations carry an uppercase HTTP method in args[1].
// A fully-lowercase args[1] is therefore the strongest cheap signal
// that we're inside a plugin sub-tree (config, dt, list, …).
// Anything else is treated as legacy-mode and falls through to the historical `too many arguments` error so we don't shadow OpenAPI semantics.
//
// Decision tree (caller has already ensured len(args) > 2):
//
//	Gate-1  args[1] in {GET,POST,PUT,DELETE} → fall through
//	        (RESTful OpenAPI invocation shape — not a plugin command)
//
//	Gate-2  args[1] contains any uppercase character → fall through
//	        (PascalCase OpenAPI APIName or other legacy shape)
//
//	Step 1  Look up args[0] in the remote plugin catalog
//	        (case-insensitive) and remember the canonical PluginInfo.
//
//	Step 2  Plugin is known to the index. Locally installed?
//	        YES → forward to plugin --help (let it self-render).
//	              Binary vanished → reinstall guidance (no fall-through).
//	        NO  → install guidance (uses canonical Name from index).
//
//	Step 3  Index miss but locally installed (dev side-load case)?
//	        YES → forward to plugin --help anyway.
//	        NO  → continue.
//
//	Step 4  Neither plugin nor product → InvalidProductOrPluginError
//	        with fuzzy suggestions sourced from the plugin index.
//
// Beyond the two gates this function NEVER returns (false, nil):
// once we've decided the shape IS a plugin command,
// the legacy "too many arguments" error would actively mislead the user,
// so every reachable path produces an actionable plugin-flavoured diagnostic.
func (c *Commando) tryDelegatePluginHelp(ctx *cli.Context, args []string) (bool, error) {
	// Defensive caller-invariant: production callers guarantee len > 2.
	if len(args) < 2 {
		return false, nil
	}

	// Gate-1: RESTful HTTP method in args[1] is the legacy `aliyun <product> <METHOD> <path>` shape.
	upper := strings.ToUpper(args[1])
	if upper == "GET" || upper == "POST" || upper == "PUT" || upper == "DELETE" {
		return false, nil
	}

	// Gate-2: any uppercase character in args[1] strongly suggests an OpenAPI APIName (PascalCase by convention).
	// Treat it as legacy mode so users typing `aliyun ecs DescribeRegions extra` keep
	// seeing the historical "too many arguments" wording instead of a plugin-flavoured error that would point at the wrong problem.
	if strings.ToLower(args[1]) != args[1] {
		return false, nil
	}

	productCode := args[0]

	// Step 1: look up args[0] in the remote plugin catalog. We hold the canonical PluginInfo (Name, ProductCode) so install guidance can
	// reference the exact `aliyun plugin install --names <name>` form.
	var indexHit *plugin.PluginInfo
	if c.pluginIndex != nil {
		for i := range c.pluginIndex.Plugins {
			if strings.EqualFold(c.pluginIndex.Plugins[i].ProductCode, productCode) {
				indexHit = &c.pluginIndex.Plugins[i]
				break
			}
		}
	} else if c.pluginIndexErr != nil {
		// Remote catalog unavailable (offline / network).
		// Print the fetch-failure note so the user knows install guidance and fuzzy suggestions may be incomplete, then continue.
		c.printPluginIndexLoadFailureNote(ctx)
	}

	userCmd := "aliyun " + strings.Join(args, " ")

	// Step 2: known plugin → branch on local install state.
	if indexHit != nil {
		installed, _, ierr := helpDelegateIsInstalled(productCode)
		if ierr != nil || !installed {
			// Manifest unreadable is treated as "not installed" — the install guidance is the right next step in either case
			// (re-running install will repair a corrupted manifest).
			return true, fmt.Errorf(
				"'%s' looks like a plugin command but plugin '%s' is not installed.\n"+
					"Run 'aliyun plugin install --name %s' to install it and try again.",
				userCmd, productCode, indexHit.Name,
			)
		}
		// Installed → forward to plugin --help.
		pluginArgs := getPluginArgsForHelp(productCode)
		c.setLangEnv(ctx)
		if ok, perr := helpDelegateExecute(productCode, pluginArgs, ctx); ok {
			return true, perr
		}
		// Manifest said installed but the binary vanished between checks.
		// Don't fall through to the legacy error — surface reinstall guidance so the user gets a one-line fix.
		return true, fmt.Errorf(
			"'%s' looks like a plugin command and plugin '%s' is registered as installed but its binary cannot be located.\n"+
				"Run 'aliyun plugin install --name %s' to reinstall it and try again.",
			userCmd, productCode, indexHit.Name,
		)
	}

	// Step 3: index miss but locally installed (dev side-load case — internal plugins distributed before they hit the public catalog).
	// Forward anyway so these users still get plugin-rendered help.
	if installed, _, ierr := helpDelegateIsInstalled(productCode); ierr == nil && installed {
		pluginArgs := getPluginArgsForHelp(productCode)
		c.setLangEnv(ctx)
		if ok, perr := helpDelegateExecute(productCode, pluginArgs, ctx); ok {
			return true, perr
		}
		// Binary vanished and we have no canonical Name to suggest a reinstall command — fall through to step 4 fuzzy diagnostics.
	}

	// Step 4: completely unknown plugin name — fuzzy-suggest from the
	// remote index via InvalidProductOrPluginError.GetSuggestions.
	//
	// Note: built-in OpenAPI products technically can't reach this
	// step (Gate-2 filters out PascalCase APINames), but a user who types an all-lowercase product name like `ecs foo bar` lands here as if they'd typo'd a plugin.
	// The Hint surfaces the product-vs-plugin distinction so OpenAPI users who took the "wrong" path get the right syntax instead of a misleading "'ecs' is not a valid product" with no further context.
	// The fuzzy suggester is still loaded with plugin names so genuine plugin typos see "Did you mean ..." underneath.
	var plugList []plugin.PluginInfo
	if c.pluginIndex != nil {
		plugList = c.pluginIndex.Plugins
	}
	return true, &InvalidProductOrPluginError{
		Code:    productCode,
		library: c.library,
		plugins: plugList,
		Hint: "If you meant an OpenAPI built-in call, the form is " +
			"'aliyun <product> <APIName>' (API names are PascalCase, e.g. 'DescribeRegions').",
	}
}
