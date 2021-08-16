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
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewConfigureDeleteCommand() *cli.Command {
	cmd := &cli.Command{
		Name:  "delete",
		Usage: "delete --profile <profileName>",
		Short: i18n.T("delete the specified profile", "删除指定配置"),
		Run: func(c *cli.Context, args []string) error {
			profileName, ok := ProfileFlag(c.Flags()).GetValue()
			if !ok {
				cli.Errorf(c.Stderr(), "missing --profile <profileName>\n")
				cli.Noticef(c.Stderr(), "\nusage:\n  aliyun configure delete --profile <profileName>\n")
				return nil
			}
			doConfigureDelete(c, profileName)
			return nil
		},
	}
	return cmd
}

func doConfigureDelete(ctx *cli.Context, profileName string) {
	conf, err := loadConfiguration()
	if err != nil {
		cli.Errorf(ctx.Stderr(), "ERROR: load configure failed: %v\n", err)
	}
	deleted := false
	r := make([]Profile, 0)
	for _, p := range conf.Profiles {
		if p.Name != profileName {
			r = append(r, p)
		} else {
			deleted = true
		}
	}

	if !deleted {
		cli.Errorf(ctx.Stderr(), "Error: configuration profile `%s` not found\n", profileName)
		return
	}

	conf.Profiles = r
	if conf.CurrentProfile == profileName {
		if len(conf.Profiles) > 0 {
			conf.CurrentProfile = conf.Profiles[0].Name
		} else {
			conf.CurrentProfile = DefaultConfigProfileName
		}
	}

	err = hookSaveConfiguration(SaveConfiguration)(conf)
	if err != nil {
		cli.Errorf(ctx.Stderr(), "Error: save configuration failed %s\n", err)
	}
}
