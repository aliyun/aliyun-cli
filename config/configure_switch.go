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
package config

import (
	"fmt"
	"io"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewConfigureSwitchCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "switch",
		Usage: "switch [--profile <profileName>]",
		Short: i18n.T("switch default profile", "切换默认配置"),
		Run: func(c *cli.Context, args []string) error {
			return doConfigureSwitch(c.Writer(), c.Flags())
		},
	}

	AddFlags(cmd.Flags())
	return cmd
}

func doConfigureSwitch(stdout io.Writer, flags *cli.FlagSet) error {
	config, err := loadConfiguration()
	if err != nil {
		return fmt.Errorf("load configuration failed: %s", err)
	}

	profileName, ok := ProfileFlag(flags).GetValue()
	if !ok {
		return fmt.Errorf("the --profile <profileName> is required")
	}

	_, ok = config.GetProfile(profileName)
	if !ok {
		return fmt.Errorf("the profile `%s` is inexist", profileName)
	}

	config.CurrentProfile = profileName
	err = hookSaveConfiguration(SaveConfiguration)(config)
	if err != nil {
		return fmt.Errorf("save configuration failed: %s", err)
	}

	cli.Println(stdout, fmt.Sprintf("The default profile is `%s` now.", profileName))
	return nil
}
