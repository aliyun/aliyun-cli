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

func main() {
	cli.PlatformCompatible()
	writer := cli.DefaultWriter()
	stderr := cli.DefaultStderrWriter()

	// load current configuration
	profile, err := config.LoadCurrentProfile()
	if err != nil {
		cli.Errorf(stderr, "ERROR: load current configuration failed %s", err)
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

	ctx := cli.NewCommandContext(writer, stderr)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewVersionCommand())
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	rootCmd.Execute(ctx, os.Args[1:])
}
