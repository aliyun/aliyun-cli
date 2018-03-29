package main

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/command"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/openapi"
	"github.com/aliyun/aliyun-cli/oss/lib"
	"github.com/aliyun/aliyun-cli/resource"
	"os"
	"strings"
)

/**
## Configure

$ aliyuncli configure
	Aliyun Access Key ID [****************wQ7v]:
	Aliyun Access Key Secret [****************fxGu]:
	Default Region Id [cn-hangzhou]:
	Default output format [json]:

## OpenApi mode
	$ aliyuncli Ecs DescribeInstances
	$ aliyuncli Ecs StartInstance --InstanceId your_instance_id
	$ aliyuncli Rds DescribeDBInstances

## use HTTPS(SSL/TLS)

	$ aliyuncli Ecs DescribeInstances --secure
*/

var library = meta.LoadLibrary(resource.NewReader())
var helper = openapi.NewHelper(library)
var configureCommand = config.NewConfigureCommand()

func main() {
	//
	// set language from current configuration
	profile, err := config.LoadCurrentProfile()
	if err != nil {
		cli.Errorf("get current failed %s", err)
		return
	}
	i18n.SetLanguage(profile.Language)

	rootCmd := &cli.Command{
		Name:              "aliyun",
		Short:             i18n.T("Alibaba Cloud Command Line Interface Version "+cli.Version, "阿里云CLI命令行工具 "+cli.Version),
		Usage:             "aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample:            "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
		Run: func(ctx *cli.Context, args []string) error {
			return processMain(ctx, args)
		},
		Help: func(ctx *cli.Context, args []string) error {
			return processHelp(ctx, args)
		},
		AutoComplete: func(ctx *cli.Context, args []string) []string {
			return processCompletion(ctx, args)
		},
	}

	fs := rootCmd.Flags()
	config.AddFlags(fs)
	openapi.AddFlags(fs)

	rootCmd.AddSubCommand(configureCommand)
	rootCmd.AddSubCommand(command.NewTestCommand())
	rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	rootCmd.Execute(os.Args[1:])
}

func processMain(ctx *cli.Context, args []string) error {
	//
	// aliyun
	if len(args) == 0 {
		printUsage(ctx.Command(), nil)
		return nil
	}

	//
	// aliyun ecs
	// 1. check configure
	productName := args[0]
	cfg, err := config.LoadConfiguration()
	if err != nil {
		ctx.Command().PrintFailed(err, "Use `aliyun configure` again.")
		return nil
	}

	prof := cfg.GetCurrentProfile(ctx)
	err = prof.Validate()
	if err != nil {
		ctx.Command().PrintFailed(err, "Use `aliyun configure` again.")
		return nil
	}
	i18n.SetLanguage(prof.Language)

	caller := openapi.NewCaller(&prof, library)
	if len(args) < 2 {
		return helper.PrintProductUsage(productName, true)
	} else if len(args) == 2 {
		return caller.Run(ctx, productName, args[1], "")
	} else if len(args) == 3 {
		return caller.Run(ctx, productName, args[1], args[2])
	} else {
		ctx.Command().PrintFailed(fmt.Errorf("too many arguments"),
			"Use aliyun --help to show usage")
		return nil
	}
}

func processCompletion(ctx *cli.Context, args []string) []string {
	r := make([]string, 0)
	//
	// aliyun
	if len(args) == 0 {
		ctx.Command().ExecuteComplete(ctx, args)
		for _, p := range library.Products {
			if !strings.HasPrefix(p.GetLowerCode(), ctx.Completion().Current) {
				continue
			}
			fmt.Printf("%s\n", p.GetLowerCode())
		}
		return r
	}

	product, ok := library.GetProduct(args[0])
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
		api, ok := library.GetApi(product.Code, product.Version, args[1])
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

func processHelp(ctx *cli.Context, args []string) error {
	c := ctx.Command()
	//if err != nil {
	//	cli.Errorf("ERROR: %s\n", err.Error())
	//	printUsage(ctx.Command(), nil)
	// } else {
	if len(args) == 0 {
		c.PrintHead()
		c.PrintUsage()
		c.PrintFlags(ctx)
		c.PrintSample()
		helper.PrintProducts()
		c.PrintTail()
		return nil
	} else if len(args) == 1 {
		c.PrintHead()
		return helper.PrintProductUsage(args[0], true)
		// c.PrintFlags() TODO add later
	} else if len(args) == 2 {
		c.PrintHead()
		return helper.PrintApiUsage(args[0], args[1])
		// c.PrintFlags() TODO add later
	} else {
		return fmt.Errorf("too many arguments: %d", len(args))
	}
}

func printUsage(c *cli.Command, configError error) {
	c.PrintHead()
	c.PrintUsage()
	c.PrintSubCommands()
	c.PrintFlags(nil)
	c.PrintSample()
	if configError != nil {
		fmt.Printf("Configuration Invailed: %s\n", configError)
		fmt.Printf("Run `aliyun configure` first:\n  %s\n", configureCommand.Usage)
	}
	c.PrintTail()
}
