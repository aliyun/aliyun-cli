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
	"encoding/json"
	"reflect"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewConfigureGetCommand() *cli.Command {
	return &cli.Command{
		Name: "get",
		Short: i18n.T(
			"print configuration values",
			"打印配置信息"),
		Usage: "get [profile] [language] ...",
		Run: func(c *cli.Context, args []string) error {
			doConfigureGet(c, args)
			return nil
		},
	}
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
		cli.Println(c.Stdout(), string(data))
		cli.Printf(c.Stdout(), "\n")
		return
	}

	for _, arg := range args {
		switch arg {
		case ProfileFlagName:
			cli.Printf(c.Stdout(), "profile=%s\n", profile.Name)
		case ModeFlagName:
			cli.Printf(c.Stdout(), "mode=%s\n", profile.Mode)
		case AccessKeyIdFlagName:
			cli.Printf(c.Stdout(), "access-key-id=%s\n", MosaicString(profile.AccessKeyId, 3))
		case AccessKeySecretFlagName:
			cli.Printf(c.Stdout(), "access-key-secret=%s\n", MosaicString(profile.AccessKeySecret, 3))
		case StsTokenFlagName:
			cli.Printf(c.Stdout(), "sts-token=%s\n", profile.StsToken)
		case StsRegionFlagName:
			cli.Printf(c.Stdout(), "sts-region=%s\n", profile.StsRegion)
		case RamRoleNameFlagName:
			cli.Printf(c.Stdout(), "ram-role-name=%s\n", profile.RamRoleName)
		case RamRoleArnFlagName:
			cli.Printf(c.Stdout(), "ram-role-arn=%s\n", profile.RamRoleArn)
		case ExternalIdFlagName:
			cli.Printf(c.Stdout(), "external-id=%s\n", profile.ExternalId)
		case RoleSessionNameFlagName:
			cli.Printf(c.Stdout(), "role-session-name=%s\n", profile.RoleSessionName)
		case KeyPairNameFlagName:
			cli.Printf(c.Stdout(), "key-pair-name=%s\n", profile.KeyPairName)
		case PrivateKeyFlagName:
			cli.Printf(c.Stdout(), "private-key=%s\n", profile.PrivateKey)
		case RegionFlagName:
			cli.Printf(c.Stdout(), profile.RegionId)
		case LanguageFlagName:
			cli.Printf(c.Stdout(), "language=%s\n", profile.Language)
		case CloudSSOSignInUrlFlagName:
			cli.Printf(c.Stdout(), "cloud-sso-sign-in-url=%s\n", profile.CloudSSOSignInUrl)
		case CloudSSOAccessConfigFlagName:
			cli.Printf(c.Stdout(), "cloud-sso-access-config=%s\n", profile.CloudSSOAccessConfig)
		case CloudSSOAccountIdFlagName:
			cli.Printf(c.Stdout(), "cloud-sso-account-id=%s\n", profile.CloudSSOAccountId)
		}
	}

	cli.Printf(c.Stdout(), "\n")
}
