// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

import (
	"os"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/openapi"
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

func main() {
	cli.PlatformCompatible()
	writer := cli.DefaultWriter()
	//
	// load current configuration
	profile, err := config.LoadCurrentProfile(writer)
	if err != nil {
		cli.Errorf(writer, "ERROR: load current configuration failed %s", err)
		return
	}

	// set user agent
	userAgentFromEnv := os.Getenv("ALIYUN_USER_AGENT")
	config.SetUserAgent(userAgentFromEnv)

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
