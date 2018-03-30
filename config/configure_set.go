/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
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
			doConfigureSet(c)
			return nil
		},
	}

	fs := cmd.Flags()

	fs.Add(ModeFlag)
	fs.Add(AccessKeyIdFlag)
	fs.Add(AccessKeySecretFlag)
	fs.Add(StsTokenFlag)
	fs.Add(RamRoleNameFlag)
	fs.Add(RamRoleArnFlag)
	fs.Add(RoleSessionNameFlag)
	fs.Add(PrivateKeyFlag)
	fs.Add(KeyPairNameFlag)
	fs.Add(RegionFlag)
	fs.Add(LanguageFlag)

	//fs.Add(cli.Flag{Name: "output", AssignedMode: cli.AssignedOnce, Hidden: true,
	//	Usage: i18n.T("* assign output format, only support json", "")})

	//fs.Add(cli.Flag{Name: "site", AssignedMode: cli.AssignedOnce,
	//	Usage: i18n.T("assign site, support china/international/japan", "")})

	return cmd
}

func doConfigureSet(c *cli.Context) {
	config, err := LoadConfiguration()
	if err != nil {
		cli.Errorf("load configuration failed %s", err)
		return
	}

	profileName, ok := ProfileFlag.GetValue()
	if !ok {
		profileName = config.CurrentProfile
	}

	profile, ok := config.GetProfile(profileName)
	if !ok {
		profile = NewProfile(profileName)
	}

	mode, ok := ModeFlag.GetValue()
	if ok {
		profile.Mode = AuthenticateMode(mode)
	} else {
		if profile.Mode == "" {
			profile.Mode = AK
		}
	}

	switch profile.Mode {
	case AK:
		profile.AccessKeyId = AccessKeyIdFlag.GetValueOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag.GetValueOrDefault(profile.AccessKeySecret)
	case StsToken:
		profile.AccessKeyId = AccessKeyIdFlag.GetValueOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeyIdFlag.GetValueOrDefault(profile.AccessKeySecret)
		profile.StsToken = StsTokenFlag.GetValueOrDefault(profile.StsToken)
	case RamRoleArn:
		profile.AccessKeyId = AccessKeyIdFlag.GetValueOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag.GetValueOrDefault(profile.AccessKeySecret)
		profile.RamRoleArn = RamRoleArnFlag.GetValueOrDefault(profile.RamRoleArn)
		profile.RoleSessionName = RoleSessionNameFlag.GetValueOrDefault(profile.RoleSessionName)
	case EcsRamRole:
		profile.RamRoleName = RamRoleNameFlag.GetValueOrDefault(profile.RamRoleName)
	case RsaKeyPair:
		profile.PrivateKey = PrivateKeyFlag.GetValueOrDefault(profile.PrivateKey)
		profile.KeyPairName = KeyPairNameFlag.GetValueOrDefault(profile.KeyPairName)
	}

	profile.RegionId = RegionFlag.GetValueOrDefault(profile.RegionId)
	profile.Language = LanguageFlag.GetValueOrDefault(profile.Language)
	profile.OutputFormat = "json" // "output", profile.OutputFormat)
	profile.Site = "china"        // "site", profile.Site)

	err = profile.Validate()
	if err != nil {
		cli.Errorf("fail to set configuration: %s", err.Error())
		return
	}

	config.PutProfile(profile)
	config.CurrentProfile = profile.Name
	err = SaveConfiguration(config)
	if err != nil {
		cli.Errorf("save configuration failed %s", err)
	}
}
