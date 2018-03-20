
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
	"github.com/aliyun/aliyun-cli/oss/lib"
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
const Version = "0.70 BETA"

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
		Name: "aliyun",
		Short: i18n.T("Alibaba Cloud Command Line Interface Version " + Version, "阿里云CLI命令行工具 " + Version),
		Usage: "aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample: "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
		Run: func(ctx *cli.Context, args []string) error {
			return processMain(ctx, args)
		},
		Help: func(ctx *cli.Context, args []string) error {
			return processHelp(ctx, args)
		},
	}

	fs := rootCmd.Flags()
	config.AddFlags(fs)
	openapi.AddFlags(fs)

	rootCmd.AddSubCommand(configureCommand)
	rootCmd.AddSubCommand(command.NewTestCommand())
	rootCmd.AddSubCommand(lib.NewOssCommand())
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

func processHelp(ctx *cli.Context, args []string) error {
	c := ctx.Command()
	//if err != nil {
	//	cli.Errorf("ERROR: %s\n", err.Error())
	//	printUsage(ctx.Command(), nil)
	// } else {
	if len(args) == 0 {
		c.PrintHead()
		c.PrintUsage()
		c.PrintFlags()
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
	c.PrintFlags()
	c.PrintSample()
	if configError != nil {
		fmt.Printf("Configuration Invailed: %s\n", configError)
		fmt.Printf("Run `aliyun configure` first:\n  %s\n", configureCommand.Usage)
	}
	c.PrintTail()
}