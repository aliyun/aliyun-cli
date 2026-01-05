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
	"bytes"

	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/cli/plugin"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/meta"

	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	jmespath "github.com/jmespath/go-jmespath"
)

// main entrance of aliyun cli
type Commando struct {
	profile config.Profile
	library *Library
}

var hookdo = func(fn func() (*responses.CommonResponse, error)) func() (*responses.CommonResponse, error) {
	return fn
}

func NewCommando(w io.Writer, profile config.Profile) *Commando {
	r := &Commando{
		profile: profile,
	}
	r.library = NewLibrary(w, profile.Language) //TODO: load from local repository
	return r
}

func (c *Commando) InitWithCommand(cmd *cli.Command) {
	cmd.Run = c.main
	cmd.Help = c.help
	cmd.AutoComplete = c.complete
}

func DetectInConfigureMode(flags *cli.FlagSet) bool {
	_, modeExist := flags.GetValue(config.ModeFlagName)
	if !modeExist {
		return true
	}
	// if mode exist, check if other flags exist
	_, akExist := flags.GetValue(config.AccessKeyIdFlagName)
	if akExist {
		return true
	}
	_, skExist := flags.GetValue(config.AccessKeySecretFlagName)
	if skExist {
		return true
	}
	_, stsExist := flags.GetValue(config.StsTokenFlagName)
	if stsExist {
		return true
	}
	// RamRoleNameFlagName
	_, ramRoleNameExist := flags.GetValue(config.RamRoleNameFlagName)
	if ramRoleNameExist {
		return true
	}
	// RamRoleArnFlagName
	_, ramRoleArnExist := flags.GetValue(config.RamRoleArnFlagName)
	if ramRoleArnExist {
		return true
	}
	// RoleSessionNameFlagName
	_, roleSessionNameExist := flags.GetValue(config.RoleSessionNameFlagName)
	if roleSessionNameExist {
		return true
	}
	// PrivateKeyFlagName
	_, privateKeyExist := flags.GetValue(config.PrivateKeyFlagName)
	if privateKeyExist {
		return true
	}
	// KeyPairNameFlagName
	_, keyPairNameExist := flags.GetValue(config.KeyPairNameFlagName)
	if keyPairNameExist {
		return true
	}
	// OIDCProviderARNFlagName
	_, oidcProviderArnExist := flags.GetValue(config.OIDCProviderARNFlagName)
	if oidcProviderArnExist {
		return true
	}
	// OIDCTokenFileFlagName
	_, oidcTokenFileExist := flags.GetValue(config.OIDCTokenFileFlagName)
	if oidcTokenFileExist {
		return true
	}
	return false
}

func (c *Commando) main(ctx *cli.Context, args []string) error {
	// aliyun
	if len(args) == 0 {
		c.printUsage(ctx)
		return nil
	}
	// Strategy: Plugin Execution
	// If the second argument (API name) is kebab-case (contains '-'), try plugin first.
	// fmt.Println("args", args)
	// fmt.Println("os.Args", os.Args)
	if len(args) > 1 {
		apiOrMethod := args[1]
		// Check if it's kebab-case (plugin format)
		if strings.Contains(apiOrMethod, "-") || apiOrMethod == "version" {
			// Extract plugin arguments from os.Args
			var pluginArgs []string
			cmdIndex := -1
			for i, arg := range os.Args {
				if arg == args[0] {
					cmdIndex = i
					break
				}
			}
			if cmdIndex != -1 && cmdIndex < len(os.Args)-1 {
				pluginArgs = os.Args[cmdIndex:]
			}

			installed, pluginName, err := plugin.IsPluginInstalled(args[0])
			if err != nil {
				return fmt.Errorf("failed to check plugin status: %w", err)
			}
			if !installed {
				return fmt.Errorf("plugin '%s' not found. Install it with: aliyun plugin install %s", args[0], args[0])
			}

			ok, err := plugin.ExecutePlugin(args[0], pluginArgs, ctx)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("plugin %s not found", pluginName)
			}
			return nil
		}
	} else if len(args) == 1 {
		installed, pluginName, err := plugin.IsPluginInstalled(args[0])
		if err != nil {
			return fmt.Errorf("failed to check plugin status: %w", err)
		}
		if installed {
			ok, err := plugin.ExecutePlugin(args[0], args, ctx)
			if err != nil {
				return err
			}
			if !ok {
				return fmt.Errorf("plugin %s not found", pluginName)
			}
			return nil
		}
	}

	if cli.HelpFlag(ctx.Flags()).IsAssigned() {
		return c.help(ctx, args)
	}

	// detect if in configure mode
	ctx.SetInConfigureMode(DetectInConfigureMode(ctx.Flags()))

	// update current `Profile` with flags
	var err error
	c.profile, err = config.LoadProfileWithContext(ctx)
	if err != nil {
		return cli.NewErrorWithTip(err, "Configuration failed, use `aliyun configure` to configure it")
	}
	err = c.profile.Validate()
	if err != nil {
		return cli.NewErrorWithTip(err, "Configuration failed, use `aliyun configure` to configure it.")
	}
	i18n.SetLanguage(c.profile.Language)

	// process following commands:
	//   aliyun <productCode>
	//   aliyun <productCode> <method> --param1 value1
	//   aliyun <productCode> GET <path>
	productName := args[0]
	if len(args) == 1 {
		// aliyun <productCode>
		// TODO: aliyun pluginName ...
		return c.library.PrintProductUsage(productName, true)
	} else if len(args) == 2 {
		// rpc or restful call
		// aliyun <productCode> <method> --param1 value1
		product, _ := c.library.GetProduct(args[0])
		if product.Code != "" {
			if version, _ := ctx.Flags().Get("version").GetValue(); version != "" {
				if style, ok := c.library.GetStyle(productName, version); ok {
					product.ApiStyle = style
				} else {
					return cli.NewErrorWithTip(fmt.Errorf("unchecked version %s", version),
						"Please contact the customer support to get more info about API version")
				}
			}
		}
		if product.ApiStyle == "restful" {
			api, _ := meta.HookGetApi(c.library.GetApi)(product.Code, product.Version, args[1])
			c.CheckApiParamWithBuildInArgs(ctx, api)
			ctx.Command().Name = args[1]
			if ShouldUseOpenapi(ctx, &product) {
				return c.processApiInvoke(ctx, &product, &api, api.Method, api.PathPattern)
			}
			return c.processInvoke(ctx, productName, api.Method, api.PathPattern)
		} else {
			// RPC need check API parameters too
			api, _ := c.library.GetApi(product.Code, product.Version, args[1])
			c.CheckApiParamWithBuildInArgs(ctx, api)
		}

		return c.processInvoke(ctx, productName, args[1], "")
	} else if len(args) == 3 {
		// restful call
		// aliyun <productCode> {GET|PUT|POST|DELETE} <path> --
		product, _ := c.library.GetProduct(productName)
		api, find := meta.HookGetApiByPath(c.library.GetApiByPath)(product.Code, product.Version, args[1], args[2])
		force := ForceFlag(ctx.Flags()).IsAssigned()
		if !find && !force {
			// throw error, can not find api by path
			return cli.NewErrorWithTip(fmt.Errorf("can not find api by path %s", args[2]),
				"Please confirm if the API path exists")
		}
		if find {
			c.CheckApiParamWithBuildInArgs(ctx, api)
		}
		if ShouldUseOpenapi(ctx, &product) {
			if !find {
				return cli.NewErrorWithTip(fmt.Errorf("can not find api by path %s", args[2]),
					"Please confirm if the API path exists")
			}
			if args[2] == "/" {
				return cli.NewErrorWithTip(fmt.Errorf("too broad path: %s for method: %s, please use specific ApiName instead",
					args[2], args[1]), "Please confirm the API path")
			}
			return c.processApiInvoke(ctx, &product, &api, args[1], args[2])
		}
		return c.processInvoke(ctx, productName, args[1], args[2])
	} else {
		return cli.NewErrorWithTip(fmt.Errorf("too many arguments"),
			"Use `aliyun --help` to show usage")
	}
}

func (c *Commando) processApiInvoke(ctx *cli.Context, product *meta.Product, api *meta.Api, method string, path string) error {
	if product == nil {
		return fmt.Errorf("invalid product, please check product code")
	}

	apiContext, err := c.createHttpContext(ctx, product, api, method, path)
	if err != nil {
		return err
	}
	err = apiContext.Prepare(ctx)
	if err != nil {
		return err
	}

	err = hookHttpContextCall(apiContext.Call)()
	if err != nil {
		return err
	}
	out, err := hookHttpContextGetResponse(apiContext.GetResponse)()
	if err != nil {
		return err
	}
	if out == "" {
		return nil
	}

	// if `--quiet` assigned. do not print anything
	if QuietFlag(ctx.Flags()).IsAssigned() {
		return nil
	}

	if QueryFlag(ctx.Flags()).IsAssigned() {
		out, err = ApplyQueryFilter(ctx, out)
		if err != nil {
			return err
		}
	}

	if filter := GetOutputFilter(ctx); filter != nil {
		out, err = filter.FilterOutput(out)
		if err != nil {
			return err
		}
	}
	out = sortJSON(out)
	cli.Println(ctx.Stdout(), out)
	return nil
}

func (c *Commando) processInvoke(ctx *cli.Context, productCode string, apiOrMethod string, path string) error {

	// create specific invoker
	invoker, err := c.createInvoker(ctx, productCode, apiOrMethod, path)
	if err != nil {
		return err
	}
	err = invoker.Prepare(ctx)
	if err != nil {
		return err
	}

	// process --dryrun
	if DryRunFlag(ctx.Flags()).IsAssigned() {
		invoker.getRequest().TransToAcsRequest()
		invoker.getClient().BuildRequestWithSigner(invoker.getRequest(), nil)
		cli.Printf(ctx.Stdout(), "Skip invoke in dry-run mode, request is:\n------------------------------------\n%s\n",
			invoker.getRequest().String())
		return nil
	}

	// if invoke with helper
	out, err, ok := c.invokeWithHelper(invoker)

	// cli.Printf("invoker %v %v \n", invoker, reflect.TypeOf(invoker))
	if ok {
		if err != nil { // call with helper failed
			return err
		}
	} else {
		resp, err := hookdo(invoker.Call)()
		if err != nil {
			// if unmarshal failed,
			if !strings.Contains(strings.ToLower(err.Error()), "unmarshal") {
				return err
			}
		}
		out = resp.GetHttpContentString()
	}

	// if `--quiet` assigned. do not print anything
	if QuietFlag(ctx.Flags()).IsAssigned() {
		return nil
	}

	if QueryFlag(ctx.Flags()).IsAssigned() {
		out, err = ApplyQueryFilter(ctx, out)
		if err != nil {
			return err
		}
	}

	// process `--output ...`
	if filter := GetOutputFilter(ctx); filter != nil {
		out, err = filter.FilterOutput(out)
		if err != nil {
			return err
		}
	}

	out = sortJSON(out)

	cli.Println(ctx.Stdout(), out)
	return nil
}

func sortJSON(content string) string {
	var v interface{}
	dec := json.NewDecoder(bytes.NewReader([]byte(content)))
	dec.UseNumber()
	err := dec.Decode(&v)
	if err != nil {
		return content
	}
	buf := new(bytes.Buffer)
	encoder := json.NewEncoder(buf)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "\t")
	err = encoder.Encode(v)
	if err != nil {
		return content
	}
	return strings.TrimSuffix(buf.String(), "\n")
}

// invoke with helper
func (c *Commando) invokeWithHelper(invoker Invoker) (resp string, err error, ok bool) {
	if pager := GetPager(); pager != nil {
		// cli.Printf("call with pager")
		resp, err = pager.CallWith(invoker)
		ok = true
		return
	}

	if waiter := GetWaiter(); waiter != nil {
		// cli.Printf("call with waiter")
		resp, err = waiter.CallWith(invoker)
		ok = true
		return
	}

	ok = false
	return
}

// create invoker for specific case
// rpc: RpcInvoker, ForceRpcInvoker
// restful: RestfulInvoker
func (c *Commando) createInvoker(ctx *cli.Context, productCode string, apiOrMethod string, path string) (Invoker, error) {
	force := ForceFlag(ctx.Flags()).IsAssigned()
	basicInvoker := NewBasicInvoker(&c.profile)

	//
	// get product info
	if product, ok := c.library.GetProduct(productCode); ok {
		err := basicInvoker.Init(ctx, &product)
		if err != nil {
			return nil, err
		}
		if force {
			if version, _ := ctx.Flags().Get("version").GetValue(); version != "" {
				if style, ok := c.library.GetStyle(productCode, version); ok {
					product.ApiStyle = style
				} else {
					// 没有在 versions.json 中配置的版本可以通过 --style 自行指定
					style, _ := ctx.Flags().Get("style").GetValue()
					if style == "" {
						return nil, cli.NewErrorWithTip(fmt.Errorf("uncheked version %s", version),
							"Please use --style to specify API style, rpc or restful.")
					}
					product.ApiStyle = style
				}
			}
		}

		if strings.ToLower(product.ApiStyle) == "rpc" {
			//
			// Rpc call
			if path != "" {
				return nil, cli.NewErrorWithTip(fmt.Errorf("invalid argument %s", path),
					"Use `aliyun help %s` see more information.", product.GetLowerCode())
			}
			if force {
				return &ForceRpcInvoker{
					basicInvoker,
					apiOrMethod,
				}, nil
			}
			if api, ok := c.library.GetApi(product.Code, product.Version, apiOrMethod); ok {
				return &RpcInvoker{
					basicInvoker,
					&api,
				}, nil
			}
			return nil, &InvalidApiError{apiOrMethod, &product}
		}

		//
		// Restful Call
		// aliyun cs GET /clusters
		// aliyun cs /clusters --roa GET
		ok, method, path, err := checkRestfulMethod(ctx, apiOrMethod, path)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, cli.NewErrorWithTip(fmt.Errorf("product '%s' need restful call", product.GetLowerCode()),
				"Use `aliyun %s {GET|PUT|POST|DELETE} <path> ...`", product.GetLowerCode())
		}

		if api, ok := c.library.GetApi(product.Code, product.Version, ctx.Command().Name); ok {
			return &RestfulInvoker{
				basicInvoker,
				method,
				path,
				force,
				&api,
			}, nil
		}

		return &RestfulInvoker{
			basicInvoker,
			method,
			path,
			force,
			nil,
		}, nil
	} else {
		if !force {
			return nil, &InvalidProductError{Code: productCode, library: c.library}
		}

		//
		// force call on unknown product, use temporary product info
		product = meta.Product{
			Code: productCode,
		}
		err := basicInvoker.Init(ctx, &product)
		if err != nil {
			return nil, err
		}

		//
		// Restful Call
		// aliyun cs GET /clusters
		// aliyun cs /clusters --roa GET
		ok, method, path, err := checkRestfulMethod(ctx, apiOrMethod, path)
		if err != nil {
			return nil, err
		}
		if ok {
			return &RestfulInvoker{
				basicInvoker,
				method,
				path,
				force,
				nil,
			}, nil
		}
		return &ForceRpcInvoker{
			basicInvoker,
			apiOrMethod,
		}, nil
	}
}

func ApplyQueryFilter(ctx *cli.Context, output string) (string, error) {
	queryExpr, ok := QueryFlag(ctx.Flags()).GetValue()
	if !ok || queryExpr == "" {
		return output, nil
	}

	if output == "" {
		return output, nil
	}

	var v interface{}
	decoder := json.NewDecoder(bytes.NewBufferString(output))
	decoder.UseNumber()
	err := decoder.Decode(&v)
	if err != nil {
		return output, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	result, err := jmespath.Search(queryExpr, v)
	if err != nil {
		return output, fmt.Errorf("JMESPath query failed: %w", err)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return output, fmt.Errorf("failed to marshal query result: %w", err)
	}

	return string(resultBytes), nil
}

// openapi context
func (c *Commando) createHttpContext(ctx *cli.Context, product *meta.Product, api *meta.Api, method string, path string) (HttpInvoker, error) {
	if product == nil {
		return nil, fmt.Errorf("invalid product, please check product code")
	}

	force := ForceFlag(ctx.Flags()).IsAssigned()
	apiContext := NewHttpContext(&c.profile)
	err := apiContext.Init(ctx, product)
	if err != nil {
		return nil, err
	}
	if force {
		if version, _ := ctx.Flags().Get("version").GetValue(); version != "" {
			if style, ok := c.library.GetStyle(product.Code, version); ok {
				product.ApiStyle = style
			} else {
				// 没有在 versions.json 中配置的版本可以通过 --style 自行指定
				style, _ := ctx.Flags().Get("style").GetValue()
				if style == "" {
					return nil, cli.NewErrorWithTip(fmt.Errorf("unchecked version %s", version),
						"Please use --style to specify API style, rpc or restful.")
				}
				product.ApiStyle = style
			}
		}
	}

	if strings.ToLower(product.ApiStyle) == "rpc" || !ShouldUseOpenapi(ctx, product) {
		return nil, cli.NewErrorWithTip(fmt.Errorf("unchecked api style: %s or product: %s", product.ApiStyle, product.Code),
			"Unsupported api style or product")
	}
	ok, method, path, err := checkRestfulMethod(ctx, method, path)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, cli.NewErrorWithTip(fmt.Errorf("product '%s' need proper restful call with ApiName or {GET|PUT|POST|DELETE} <path>",
			product.GetLowerCode()),
			"Use `aliyun %s <ApiName> ...` or `aliyun %s {GET|PUT|POST|DELETE} <path> ...`",
			product.GetLowerCode(),
			product.GetLowerCode())
	}
	return &OpenapiContext{apiContext, method, path, api}, nil
}

func (c *Commando) help(ctx *cli.Context, args []string) error {
	cmd := ctx.Command()
	if len(args) == 0 {
		cmd.PrintHead(ctx)
		cmd.PrintUsage(ctx)
		cmd.PrintFlags(ctx)
		cmd.PrintSample(ctx)
		c.library.PrintProducts()
		cmd.PrintTail(ctx)
		return nil
	} else if len(args) == 1 {
		cmd.PrintHead(ctx)
		return c.library.PrintProductUsage(args[0], true)
	} else if len(args) == 2 {
		cmd.PrintHead(ctx)
		return c.library.PrintApiUsage(args[0], args[1])
	} else {
		return fmt.Errorf("too many arguments: %d", len(args))
	}
}

func (c *Commando) complete(ctx *cli.Context, args []string) []string {
	w := ctx.Stdout()

	r := make([]string, 0)
	//
	// aliyun
	if len(args) == 0 {
		// Case insensitive strings.ToLower()
		ctx.Command().ExecuteComplete(ctx, args)
		for _, p := range c.library.GetProducts() {
			if !strings.HasPrefix(p.GetLowerCode(), strings.ToLower(ctx.Completion().Current)) {
				continue
			}
			cli.PrintfWithColor(w, "", "%s\n", p.GetLowerCode())
		}
		return r
	}

	product, ok := c.library.GetProduct(args[0])
	if !ok {
		return r
	}

	if product.ApiStyle == "rpc" {
		if len(args) == 1 {
			for _, name := range product.ApiNames {
				if !strings.HasPrefix(strings.ToLower(name), strings.ToLower(ctx.Completion().Current)) {
					continue
				}
				cli.PrintfWithColor(w, "", "%s\n", name)
			}
			return r
		}
		api, ok := c.library.GetApi(product.Code, product.Version, args[1])
		if !ok {
			return r
		}

		api.ForeachParameters(func(s string, p meta.Parameter) {
			if strings.HasPrefix("--"+strings.ToLower(s), strings.ToLower(ctx.Completion().Current)) && !p.Hidden {
				cli.Printf(ctx.Stdout(), "--%s\n", s)
			}
		})
	} else if product.ApiStyle == "restful" {
		if len(args) == 1 {
			cli.PrintfWithColor(w, "", "GET\n")
			cli.PrintfWithColor(w, "", "POST\n")
			cli.PrintfWithColor(w, "", "DELETE\n")
			cli.PrintfWithColor(w, "", "PUT\n")

			return r
		}
	}

	return r
}

func (c *Commando) printUsage(ctx *cli.Context) {
	cmd := ctx.Command()
	cmd.PrintHead(ctx)
	cmd.PrintUsage(ctx)
	cmd.PrintSubCommands(ctx)
	cmd.PrintFlags(ctx)
	cmd.PrintSample(ctx)
	cmd.PrintTail(ctx)
}

func (c *Commando) CheckApiParamWithBuildInArgs(ctx *cli.Context, api meta.Api) {
	for _, p := range api.Parameters {
		// 如果参数中包含了known参数，且 known参数已经被赋值，则将 known 参数拷贝给 unknown 参数
		if ep, ok := ctx.Flags().GetValue(p.Name); ok {
			if p.Position != "Query" {
				continue
			}
			var flagNew = &cli.Flag{
				Name: p.Name,
			}
			flagNew.SetValue(ep)
			flagNew.SetAssigned(true)
			ctx.UnknownFlags().Add(flagNew)
		}
	}
}
