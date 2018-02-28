
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
const Version = "0.50 BETA"

var profileName string
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
		Help: func(ctx *cli.Context, args []string, err error) {
			processHelp(ctx, args, err)
		},
	}

	rootCmd.Flags().StringVar(&profileName, "profile", "default",
		i18n.T("use --profile <profileName> to use specified profile", "使用 --profile <profileName> 来使用指定的配置"))

	rootCmd.Flags().Add(cli.Flag{Name: "secure", Assignable:false,
		Usage: i18n.T("use --secure to force https", "使用 --secure 开关强制使用https方式调用")})

	rootCmd.Flags().Add(cli.Flag{Name: "force", Assignable:false,
		Usage: i18n.T("use --force to skip api and parameters check", "添加 --force 开关可跳过API与参数的合法性检查")})

	rootCmd.Flags().Add(cli.Flag{Name: "endpoint", Assignable:true,
		Usage: i18n.T("use --endpoint <endpoint> to assign endpoint", "使用 --endpoint <endpoint> 来指定接入点地址")})

	rootCmd.Flags().Add(cli.Flag{Name: "region", Assignable:true,
		Usage: i18n.T("use --region <regionId> to assign region", "使用 --region <regionId> 来指定访问地域")})
	rootCmd.Flags().Add(cli.Flag{Name: "version", Assignable:true,
		Usage: i18n.T("use --version <YYYY-MM-DD> to assign product api version", "使用 --version <YYYY-MM-DD> 来指定访问的API版本")})

	rootCmd.Flags().Add(cli.Flag{Name: "header", Assignable:true, Repeatable:true,
		Usage: i18n.T("use --header X-foo=bar to add custom HTTP header, repeatable", "使用 --header X-foo=bar 来添加特定的HTTP头, 可多次添加")})

	rootCmd.Flags().Add(cli.Flag{Name: "body", Assignable:true,
		Usage: i18n.T("use --body $(cat foo.json) to assign http body in RESTful call", "使用 --body $(cat foo.json) 来指定在RESTful调用中的HTTP包体")})

	rootCmd.Flags().Add(cli.Flag{Name: "body-file", Assignable:true, Hidden: true,
		Usage: i18n.T("assign http body in Restful call with local file", "")})

	rootCmd.AddSubCommand(configureCommand)
	rootCmd.AddSubCommand(command.NewTestCommand())
	rootCmd.AddSubCommand(openapi.NewResolveCommand())
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
		helper.PrintProductUsage(productName, true)
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
			helper.PrintProductUsage(args[0], true)
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