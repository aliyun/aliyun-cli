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
	"text/tabwriter"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/i18n"
)

func NewConfigureListCommand() *cli.Command {
	return &cli.Command{
		Name:  "list",
		Usage: "list",
		Short: i18n.T("list all config profile", "列出所有配置集"),
		Run: func(c *cli.Context, args []string) error {
			doConfigureList(c.Stdout())
			return nil
		},
	}
}

func doConfigureList(w io.Writer) {
	conf, err := loadConfiguration()
	if err != nil {
		cli.Errorf(w, "ERROR: load configure failed: %v\n", err)
	}
	tw := tabwriter.NewWriter(w, 8, 0, 1, ' ', 0)
	fmt.Fprint(tw, "Profile\t| Credential \t| Valid\t| Region\t| Language\n")
	fmt.Fprint(tw, "---------\t| ------------------\t| -------\t| ----------------\t| --------\n")
	for _, pf := range conf.Profiles {
		name := pf.Name
		if name == conf.CurrentProfile {
			name = name + " *"
		}
		err := pf.Validate()
		valid := "Valid"
		if err != nil {
			valid = "Invalid"
		}

		cred := ""
		switch pf.Mode {
		case AK:
			cred = "AK:" + "***" + GetLastChars(pf.AccessKeyId, 3)
		case StsToken:
			cred = "StsToken:" + "***" + GetLastChars(pf.AccessKeyId, 3)
		case RamRoleArn:
			cred = "RamRoleArn:" + "***" + GetLastChars(pf.AccessKeyId, 3)
			if pf.ExternalId != "" {
				cred = cred + ":" + GetLastChars(pf.ExternalId, 3)
			}
		case EcsRamRole:
			cred = "EcsRamRole:" + pf.RamRoleName
		case RamRoleArnWithEcs:
			cred = "arn:" + "***" + GetLastChars(pf.AccessKeyId, 3)
		case ChainableRamRoleArn:
			cred = "ChainableRamRoleArn:" + pf.SourceProfile + ":" + pf.RamRoleArn
			if pf.ExternalId != "" {
				cred = cred + ":" + GetLastChars(pf.ExternalId, 3)
			}
		case RsaKeyPair:
			cred = "RsaKeyPair:" + pf.KeyPairName
		case External:
			cred = "ProcessCommand:" + pf.ProcessCommand
		case CredentialsURI:
			cred = "CredentialsURI:" + pf.CredentialsURI
		case OIDC:
			cred = "OIDC:" + "***" + GetLastChars(pf.OIDCProviderARN, 5) + "@***" + GetLastChars(pf.OIDCTokenFile, 5) + "@" + pf.RamRoleArn
		case CloudSSO:
			cred = "CloudSSO:" + pf.CloudSSOAccountId + "@" + pf.CloudSSOAccessConfig
		}
		fmt.Fprintf(tw, "%s\t| %s\t| %s\t| %s\t| %s\n", name, cred, valid, pf.RegionId, pf.Language)
	}
	tw.Flush()
}
