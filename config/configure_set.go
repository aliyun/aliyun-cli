// Copyright 1999-2019 Alibaba Group Holding Limited
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
	"io"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

const configureSetHelpEn = ``
const configureSetHelpZh = ``

func NewConfigureSetCommand() *cli.Command {

	cmd := &cli.Command{
		Name: "set",
		Short: i18n.T(
			"set config in non interactive mode",
			"使用非交互式方式进行配置"),
		Long: i18n.T(
			configureSetHelpEn,
			configureSetHelpZh,
		),
		Usage: "set [--profile <profileName>] [--language {en|zh}] ...",
		Run: func(c *cli.Context, args []string) error {
			doConfigureSet(c.Writer(), c.Flags())
			return nil
		},
	}

	AddFlags(cmd.Flags())

	//fs.Add(cli.Flag{Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T("* assign output format, only support json", "")})

	//fs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
	//	Usage: i18n.T("assign site, support china/international/japan", "")})

	return cmd
}

func doConfigureSet(w io.Writer, flags *cli.FlagSet) {
	config, err := hookLoadConfiguration(LoadConfiguration)(GetConfigPath()+"/"+configFile, w)
	if err != nil {
		cli.Errorf(w, "load configuration failed %s", err)
		return
	}

	profileName, ok := ProfileFlag(flags).GetValue()
	if !ok {
		profileName = config.CurrentProfile
	}

	profile, ok := config.GetProfile(profileName)
	if !ok {
		profile = NewProfile(profileName)
	}

	path, ok := ConfigurePathFlag(flags).GetValue()
	if ok {
		profile, err = LoadProfile(path, w, profileName)
		if err != nil {
			cli.Errorf(w, "load configuration file failed %s", err)
			return
		}
	}

	mode, ok := ModeFlag(flags).GetValue()
	if ok {
		profile.Mode = AuthenticateMode(mode)
	} else {
		if profile.Mode == "" {
			profile.Mode = AK
		}
	}

	switch profile.Mode {
	case AK:
		profile.AccessKeyId = AccessKeyIdFlag(flags).GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag(flags).GetStringOrDefault(profile.AccessKeySecret)
	case StsToken:
		profile.AccessKeyId = AccessKeyIdFlag(flags).GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeyIdFlag(flags).GetStringOrDefault(profile.AccessKeySecret)
		profile.StsToken = StsTokenFlag(flags).GetStringOrDefault(profile.StsToken)
	case RamRoleArn:
		profile.AccessKeyId = AccessKeyIdFlag(flags).GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag(flags).GetStringOrDefault(profile.AccessKeySecret)
		profile.RamRoleArn = RamRoleArnFlag(flags).GetStringOrDefault(profile.RamRoleArn)
		profile.RoleSessionName = RoleSessionNameFlag(flags).GetStringOrDefault(profile.RoleSessionName)
	case EcsRamRole:
		profile.RamRoleName = RamRoleNameFlag(flags).GetStringOrDefault(profile.RamRoleName)
	case RsaKeyPair:
		profile.PrivateKey = PrivateKeyFlag(flags).GetStringOrDefault(profile.PrivateKey)
		profile.KeyPairName = KeyPairNameFlag(flags).GetStringOrDefault(profile.KeyPairName)
	}

	profile.RegionId = RegionFlag(flags).GetStringOrDefault(profile.RegionId)
	profile.Language = LanguageFlag(flags).GetStringOrDefault(profile.Language)
	profile.OutputFormat = "json" // "output", profile.OutputFormat)
	profile.Site = "china"        // "site", profile.Site)
	profile.RetryTimeout = RetryTimeoutFlag(flags).GetIntegerOrDefault(profile.RetryTimeout)
	profile.RetryCount = RetryCountFlag(flags).GetIntegerOrDefault(profile.RetryCount)

	err = profile.Validate()
	if err != nil {
		cli.Errorf(w, "fail to set configuration: %s", err.Error())
		return
	}

	config.PutProfile(profile)
	config.CurrentProfile = profile.Name
	err = hookSaveConfiguration(SaveConfiguration)(config)
	if err != nil {
		cli.Errorf(w, "save configuration failed %s", err)
	}
}
