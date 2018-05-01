package main

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/openapi"
	"github.com/aliyun/aliyun-cli/oss/lib"
	"os"
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

func main() {

	writer := cli.DefaultWriter()
	//
	// load current configuration
	profile, err := config.LoadCurrentProfile(writer)
	if err != nil {
		cli.Errorf(writer, "ERROR: load current configuration failed %s", err)
		return
	}

	// set language with current profile
	i18n.SetLanguage(profile.Language)

	// create root command
	rootCmd := &cli.Command{
		Name:              "aliyun",
		Short:             i18n.T("Alibaba Cloud Command Line Interface Version "+cli.Version, "阿里云CLI命令行工具 "+cli.Version),
		Usage:             "aliyun <product> <operation> [--parameter1 value1 --parameter2 value2 ...]",
		Sample:            "aliyun ecs DescribeRegions",
		EnableUnknownFlag: true,
	}

	// add default flags
	config.AddFlags(rootCmd.Flags())
	openapi.AddFlags(rootCmd.Flags())

	// new open api commando to process rootCmd
	commando := openapi.NewCommando(writer, profile)
	commando.InitWithCommand(rootCmd)

	ctx := cli.NewCommandContext(writer)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	// rootCmd.AddSubCommand(command.NewTestCommand())
	rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewVersionCommand())
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	rootCmd.Execute(ctx, os.Args[1:])
}
