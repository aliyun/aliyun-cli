package openapi

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/config"
	"fmt"
	"strings"
	"github.com/aliyun/aliyun-cli/i18n"
)

type Commando struct {
	profile config.Profile
	library *Library
}

func NewCommando(profile config.Profile) *Commando {
	r := &Commando{
		profile: profile,
	}
	r.library = NewLibrary(profile.Language)	//TODO: load from local repository
	return r
}

func (c *Commando) InitWithCommand(cmd *cli.Command) {
	cmd.Run = c.main
	cmd.Help = c.help
	cmd.AutoComplete = c.complete
}

//
func (c *Commando) main(ctx *cli.Context, args []string) error {
	//
	// aliyun
	if len(args) == 0 {
		c.printUsage(ctx.Command())
		return nil
	}

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
	// 	 aliyun <productCode>
	//   aliyun <productCode> <method> --param1 value1
	//   aliyun <productCode> GET <path>
	productName := args[0]
	if len(args) == 1 {
		// aliyun <productCode>
		// TODO: aliyun pluginName ...
		return c.library.PrintProductUsage(productName, true)
	} else if len(args) == 2 {
		// rpc call
		// aliyun <productCode> <method> --param1 value1
		return c.processInvoke(ctx, productName, args[1], "")
	} else if len(args) == 3 {
		// restful call
		// aliyun <productCode> {GET|PUT|POST|DELETE} <path> --
		return c.processInvoke(ctx, productName, args[1], args[2])
	} else {
		return cli.NewErrorWithTip(fmt.Errorf("too many arguments"),
			"Use `aliyun --help` to show usage")
	}
}

func (c *Commando) processInvoke(ctx *cli.Context, productCode string, apiOrMethod string, path string) error {
	// create specific invoker
	invoker, err := c.createInvoker(ctx, productCode, apiOrMethod, path)
	if err != nil {
		return err
	}
	invoker.Prepare(ctx)

	// if invoke with helper
	out, err, ok := c.invokeWithHelper(invoker)
	// fmt.Printf("invoker %v %v \n", invoker, reflect.TypeOf(invoker))
	if !ok {
		resp, err := invoker.Call()
		if err != nil {
			// if unmarshal failed,
			if !strings.Contains(strings.ToLower(err.Error()), "unmarshal") {
				return err
			}
		}
		out = resp.GetHttpContentString()
	}

	if filter := GetOutputFilter(); filter != nil {
		out, err = filter.FilterOutput(out)
		if err != nil {
			return err
		}
	}

	if QuietFlag.IsAssigned() {
		return nil
	}

	fmt.Println(out)
	return nil
}

// invoke with helper
func (c *Commando) invokeWithHelper(invoker Invoker) (resp string, err error, ok bool) {
	if pager := GetPager(); pager != nil {
		// fmt.Printf("call with pager")
		resp, err = pager.CallWith(invoker)
		ok = true
		return
	}

	if waiter := GetWaiter(); waiter != nil {
		// fmt.Printf("call with waiter")
		resp, err = waiter.CallWith(invoker)
		ok = true
		return
	}

	ok = false
	return
}

//
// create invoker for specific case
// rpc: RpcInvoker, ForceRpcInvoker
// restful: RestfulInvoker
func (c *Commando) createInvoker(ctx *cli.Context, productCode string, apiOrMethod string, path string) (Invoker, error) {
	force := ForceFlag.IsAssigned()
	basicInvoker := NewBasicInvoker(&c.profile)

	//
	// get product info
	if product, ok := c.library.GetProduct(productCode); ok {
		err := basicInvoker.Init(ctx, &product)
		if err != nil {
			return nil, err
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
			} else {
				if api, ok := c.library.GetApi(product.Code, product.Version, apiOrMethod); ok {
					return &RpcInvoker{
						basicInvoker,
						&api,
					}, nil
				} else {
					return nil, &InvalidApiError{apiOrMethod, &product}
				}
			}
		} else {
			//
			// Restful Call
			// aliyun cs GET /clusters
			// aliyun cs /clusters --roa GET
			ok, method, path, err := checkRestfulMethod(apiOrMethod, path)
			if err != nil {
				return nil, err
			}
			if !ok {
				return nil, cli.NewErrorWithTip(fmt.Errorf("product %s need restful call", product.GetLowerCode()),
					"Use `aliyun %s {GET|PUT|POST|DELETE} <path> ...`", product.GetLowerCode())
			}
			return &RestfulInvoker{
				basicInvoker,
				method,
				path,
				force,
			}, nil
			//if err != nil {
			//	ctx.Command().PrintFailed(fmt.Errorf("call restful %s%s.%s faild %v", product.Code, path, method, err), "")
			//	return nil
			//}
		}
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
		ok, method, path, err := checkRestfulMethod(apiOrMethod, path)
		if err != nil {
			return nil, err
		}
		if ok {
			return &RestfulInvoker {
				basicInvoker,
				method,
				path,
				force,
			}, nil
			// return invoker, nil
			// c.InvokeRestful(ctx, &product, method, path)
		} else {
			return &ForceRpcInvoker {
				basicInvoker,
				method,
			}, nil
			// c.InvokeRpcForce(ctx, &product, apiOrMethod)
		}
	}
}

//
func (c *Commando) help(ctx *cli.Context, args []string) error {
	cmd := ctx.Command()
	//if err != nil {
	//	cli.Errorf("ERROR: %s\n", err.Error())
	//	printUsage(ctx.Command(), nil)
	// } else {
	if len(args) == 0 {
		cmd.PrintHead()
		cmd.PrintUsage()
		cmd.PrintFlags(ctx)
		cmd.PrintSample()
		c.library.PrintProducts()
		cmd.PrintTail()
		return nil
	} else if len(args) == 1 {
		cmd.PrintHead()
		return c.library.PrintProductUsage(args[0], true)
		// c.PrintFlags() TODO add later
	} else if len(args) == 2 {
		cmd.PrintHead()
		return c.library.PrintApiUsage(args[0], args[1])
		// c.PrintFlags() TODO add later
	} else {
		return fmt.Errorf("too many arguments: %d", len(args))
	}
}

//
func (c *Commando) complete(ctx *cli.Context, args []string) []string {
	r := make([]string, 0)
	//
	// aliyun
	if len(args) == 0 {
		ctx.Command().ExecuteComplete(ctx, args)
		for _, p := range c.library.GetProducts() {
			if !strings.HasPrefix(p.GetLowerCode(), ctx.Completion().Current) {
				continue
			}
			fmt.Printf("%s\n", p.GetLowerCode())
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
				if !strings.HasPrefix(name, ctx.Completion().Current) {
					continue
				}
				fmt.Printf("%s\n", name)
			}
			return r
		}
		api, ok := c.library.GetApi(product.Code, product.Version, args[1])
		if !ok {
			return r
		}

		api.ForeachParameters(func(s string, p meta.Parameter) {
			if strings.HasPrefix("--"+s, ctx.Completion().Current) && !p.Hidden {
				fmt.Printf("--%s\n", s)
			}
		})
	} else if product.ApiStyle == "restful" {
		if len(args) == 1 {
			fmt.Printf("GET\n")
			fmt.Printf("POST\n")
			fmt.Printf("DELETE\n")
			fmt.Printf("PUT\n")
			return r
		}
	}

	return r
}


func (c *Commando) printUsage(cmd *cli.Command) {
	cmd.PrintHead()
	cmd.PrintUsage()
	cmd.PrintSubCommands()
	cmd.PrintFlags(nil)
	cmd.PrintSample()
	//if configError != nil {
	//	fmt.Printf("Configuration Invailed: %s\n", configError)
	//	fmt.Printf("Run `aliyun configure` first:\n  %s\n", configureCommand.Usage)
	//}
	cmd.PrintTail()
}
