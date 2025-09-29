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

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewConfigureDeleteCommand() *cli.Command {
	return &cli.Command{
		Name:  "delete",
		Usage: "delete --profile <profileName> [--config-path <configPath>]",
		Short: i18n.T("delete the specified profile", "删除指定配置"),
		Run: func(c *cli.Context, args []string) error {
			profileName, ok := ProfileFlag(c.Flags()).GetValue()
			if !ok {
				cli.Noticef(c.Stderr(), "\nusage:\n  aliyun configure delete --profile <profileName> [--config-path <configPath>]\n")
				return fmt.Errorf("missing --profile <profileName>")
			}
			return doConfigureDelete(c, profileName)
		},
	}
}

func doConfigureDelete(ctx *cli.Context, profileName string) error {
	conf, err := hookLoadConfigurationWithContext(LoadConfigurationWithContext)(ctx)
	if err != nil {
		return fmt.Errorf("ERROR: load configure failed: %v", err)
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
		return fmt.Errorf("error: configuration profile `%s` not found", profileName)
	}

	conf.Profiles = r
	if conf.CurrentProfile == profileName {
		if len(conf.Profiles) > 0 {
			conf.CurrentProfile = conf.Profiles[0].Name
		} else {
			conf.CurrentProfile = DefaultConfigProfileName
		}
	}

	err = hookSaveConfigurationWithContext(SaveConfigurationWithContext)(ctx, conf)
	if err != nil {
		return fmt.Errorf("error: save configuration failed %v", err)
	}
	return nil
}
