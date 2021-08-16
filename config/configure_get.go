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
	"encoding/json"
	"reflect"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

const configureGetHelpEn = `
`
const configureGetHelpZh = `
`

func NewConfigureGetCommand() *cli.Command {

	cmd := &cli.Command{
		Name: "get",
		Short: i18n.T(
			"print configuration values",
			"打印配置信息"),
		Usage: "get [profile] [language] ...",
		Long: i18n.T(
			configureGetHelpEn,
			configureGetHelpZh,
		),
		Run: func(c *cli.Context, args []string) error {
			doConfigureGet(c, args)
			return nil
		},
	}

	return cmd
}

func doConfigureGet(c *cli.Context, args []string) {
	config, err := loadConfiguration()
	if err != nil {
		cli.Errorf(c.Stderr(), "load configuration failed %s", err)
		cli.Printf(c.Stderr(), "\n")
		return
	}

	profile := config.GetCurrentProfile(c)

	if pn, ok := ProfileFlag(c.Flags()).GetValue(); ok {
		profile, ok = config.GetProfile(pn)
		if !ok {
			cli.Errorf(c.Stderr(), "profile %s not found!", pn)
			cli.Printf(c.Stderr(), "\n")
			return
		}
	}

	if len(args) == 0 && !reflect.DeepEqual(profile, Profile{}) {
		data, err := json.MarshalIndent(profile, "", "\t")
		if err != nil {
			cli.Printf(c.Stderr(), "ERROR:"+err.Error())
			cli.Printf(c.Stderr(), "\n")
			return
		}
		cli.Println(c.Writer(), string(data))
		cli.Printf(c.Writer(), "\n")
		return
	}

	for _, arg := range args {
		switch arg {
		case ProfileFlagName:
			cli.Printf(c.Writer(), "profile=%s\n", profile.Name)
		case ModeFlagName:
			cli.Printf(c.Writer(), "mode=%s\n", profile.Mode)
		case AccessKeyIdFlagName:
			cli.Printf(c.Writer(), "access-key-id=%s\n", MosaicString(profile.AccessKeyId, 3))
		case AccessKeySecretFlagName:
			cli.Printf(c.Writer(), "access-key-secret=%s\n", MosaicString(profile.AccessKeySecret, 3))
		case StsTokenFlagName:
			cli.Printf(c.Writer(), "sts-token=%s\n", profile.StsToken)
		case StsRegionFlagName:
			cli.Printf(c.Writer(), "sts-region=%s\n", profile.StsRegion)
		case RamRoleNameFlagName:
			cli.Printf(c.Writer(), "ram-role-name=%s\n", profile.RamRoleName)
		case RamRoleArnFlagName:
			cli.Printf(c.Writer(), "ram-role-arn=%s\n", profile.RamRoleArn)
		case RoleSessionNameFlagName:
			cli.Printf(c.Writer(), "role-session-name=%s\n", profile.RoleSessionName)
		case KeyPairNameFlagName:
			cli.Printf(c.Writer(), "key-pair-name=%s\n", profile.KeyPairName)
		case PrivateKeyFlagName:
			cli.Printf(c.Writer(), "private-key=%s\n", profile.PrivateKey)
		case RegionFlagName:
			cli.Printf(c.Writer(), profile.RegionId)
		case LanguageFlagName:
			cli.Printf(c.Writer(), "language=%s\n", profile.Language)
		}
	}

	cli.Printf(c.Writer(), "\n")
}
