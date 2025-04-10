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
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	aliyunopenapimeta "github.com/aliyun/aliyun-cli/v3/aliyun-openapi-meta"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/i18n"
	"github.com/aliyun/aliyun-cli/v3/openapi"
	"github.com/aliyun/aliyun-cli/v3/oss/lib"
)

func Main(args []string) {
	stdout := cli.DefaultStdoutWriter()
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
	commando := openapi.NewCommando(stdout, profile)
	commando.InitWithCommand(rootCmd)

	ctx := cli.NewCommandContext(stdout, stderr)
	ctx.EnterCommand(rootCmd)
	ctx.SetCompletion(cli.ParseCompletionForShell())
	ctx.SetInConfigureMode(openapi.DetectInConfigureMode(ctx.Flags()))
	// use http force, current use in oss bridge
	insecure, _ := ParseInSecure(args)
	ctx.SetInsecure(insecure)

	rootCmd.AddSubCommand(config.NewConfigureCommand())
	rootCmd.AddSubCommand(lib.NewOssCommand())
	rootCmd.AddSubCommand(cli.NewVersionCommand())
	rootCmd.AddSubCommand(cli.NewAutoCompleteCommand())
	if os.Getenv("GENERATE_METADATA") == "YES" {
		generateMetadata(rootCmd)
	} else {
		rootCmd.Execute(ctx, args)
	}
}

func ParseInSecure(args []string) (bool, interface{}) {
	// check has insecure flag
	for _, arg := range args {
		if arg == "--insecure" {
			return true, nil
		}
	}
	return false, nil
}

func main() {
	Main(os.Args[1:])
}

func dumpFiles(fs embed.FS, filePath string, outputDir string) {
	filePath = strings.TrimPrefix(filePath, "./")

	entries, err := fs.ReadDir(filePath)
	if err != nil {
		fmt.Println(err.Error())
		return
	}

	for _, entry := range entries {
		entryPath := path.Join(filePath, entry.Name())
		if entry.IsDir() {
			dumpFiles(fs, entryPath, outputDir)
		} else {
			content, err := fs.ReadFile(entryPath)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			targetPath := path.Join(outputDir, entryPath)
			fmt.Println("copy file from " + entryPath + " to " + targetPath)
			_, err = os.Stat(path.Dir(targetPath))
			if os.IsNotExist(err) {
				err = os.MkdirAll(path.Dir(targetPath), 0755)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			}
			err = os.WriteFile(targetPath, content, 0666)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}
}

func generateMetadata(rootCmd *cli.Command) {
	metadata := make(map[string]*cli.Metadata)
	rootCmd.GetMetadata(metadata)
	b, _ := json.MarshalIndent(metadata, "", "  ")
	cwd, _ := os.Getwd()
	targetDir := cwd + "/cli-metadata"
	_, err := os.Stat(targetDir)
	if os.IsNotExist(err) {
		err := os.Mkdir(targetDir, 0755)
		if err != nil {
			fmt.Println(err.Error())
			return
		}
	}

	targetPath := targetDir + "/commands.json"
	err = os.WriteFile(targetPath, b, 0666)
	if err != nil {
		fmt.Println(err.Error())
	}

	versionPath := targetDir + "/version"
	os.WriteFile(versionPath, []byte(cli.Version), 0666)

	dumpFiles(aliyunopenapimeta.Metadatas, ".", targetDir)
}
