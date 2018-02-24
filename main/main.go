
package main

import (
	"github.com/aliyun/aliyun-cli/cli"
	"os"
	"github.com/aliyun/aliyun-cli/openapi"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/resource"
	"fmt"
	"github.com/aliyun/aliyun-cli/meta"
	"github.com/aliyun/aliyun-cli/command"
	"github.com/aliyun/aliyun-cli/i18n"
)

var usage = `
	Alibaba Cloud CLI(Command Line Interface)
	Usage:
`


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

## 用HTTPS(SSL/TLS)通信

	$ aliyuncli Ecs DescribeInstances --secure
*/

var profileName string
var library = meta.LoadLibrary(resource.NewReader())
var helper = openapi.NewHelper(library)
var configureCommand = config.NewConfigureCommand()

func main() {
	rootCmd := &cli.Command{
		Name: "aliyun",
		Short: i18n.T("Alibaba Cloud Command Line Interface Version 0.33 Beta", "阿里云CLI命令行工具 0.33 Beta"),
		Usage: "aliyun <product> <operation> --parameter1 value1 --parameter2 value2 ...",
		Sample: "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
		Run: func(ctx *cli.Context, args []string) error {
			return processMain(ctx, args)
		},
		Help: func(ctx *cli.Context, args []string, err error) {
			processHelp(ctx, args, err)
		},
	}

	rootCmd.Flags().StringVar(&profileName, "profile", "default",
		i18n.En("root.profile", "use configured profile"))

	rootCmd.Flags().Add(cli.Flag{Name: "force", Assignable:false,
		Usage: i18n.T("root.force", "call OpenAPI without check")})
	rootCmd.Flags().Add(cli.Flag{Name: "endpoint", Assignable:true,
		Usage: i18n.En("root.endpoint", "use assigned endpoint")})
	rootCmd.Flags().Add(cli.Flag{Name: "region", Assignable:true,
		Usage: i18n.En("root.region","use assigned region")})
	rootCmd.Flags().Add(cli.Flag{Name: "version", Assignable:true,
		Usage: i18n.En("root.version", "assign product version")})
	rootCmd.Flags().Add(cli.Flag{Name: "header", Assignable:true, Repeatable:true,
		Usage: i18n.En("root.header", "add custom HTTP header with --header x-foo=bar")})
	rootCmd.Flags().Add(cli.Flag{Name: "body", Assignable:true,
		Usage: i18n.En("root.body", "assign http body in Restful call")})
	rootCmd.Flags().Add(cli.Flag{Name: "body-file", Assignable:true, Hidden: true,
		Usage: i18n.En("root.body", "assign http body in Restful call with local file")})

	rootCmd.AddSubCommand(configureCommand)
	rootCmd.AddSubCommand(command.NewTestCommand())
	rootCmd.Execute(os.Args[1:])
}

func processMain(ctx *cli.Context, args []string) error  {
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
	prof, err := config.LoadProfile(profileName)
	if err != nil {
		ctx.Command().PrintFailed(err, "Use `aliyun configure` again.")
		return nil
	}
	err = prof.Validate()
	if err != nil {
		ctx.Command().PrintFailed(err, "Use `aliyun configure` again.")
		return nil
	}

	caller := openapi.NewCaller(&prof, library)
	if len(args) < 2 {
		helper.PrintProductUsage(productName)
	} else if len(args) == 2 {
		caller.Run(ctx, productName, args[1], "")
	} else if len(args) == 3 {
		caller.Run(ctx, productName, args[1], args[2])
	} else {
		ctx.Command().PrintFailed(fmt.Errorf("too many arguments"),
			"Use aliyun --help to show usage")
	}
	return nil
}

func processHelp(ctx *cli.Context, args []string, err error) {
	c := ctx.Command()
	if err != nil {
		cli.Errorf("ERROR: %s\n", err.Error())
		printUsage(ctx.Command(), nil)
	} else {
		if len(args) == 0 {
			c.PrintHead()
			c.PrintUsage()
			c.PrintFlags()
			c.PrintSample()
			helper.PrintProducts()
			c.PrintTail()
		} else if len(args) == 1 {
			c.PrintHead()
			helper.PrintProductUsage(args[0])
			// c.PrintFlags() TODO add later
		} else if len(args) == 2 {
			c.PrintHead()
			helper.PrintApiUsage(args[0], args[1])
			// c.PrintFlags() TODO add later
		}
	}
}

func printUsage(c *cli.Command, configError error) {
	c.PrintHead()
	c.PrintUsage()
	c.PrintSubCommands()
	c.PrintFlags()
	c.PrintSample()
	if configError != nil {
		fmt.Printf("Configuration Invailed: %s\n", configError)
		fmt.Printf("Run `aliyun configure` first:\n  %s\n", configureCommand.Usage)
	}
	c.PrintTail()
}