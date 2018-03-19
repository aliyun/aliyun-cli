/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

const configureSetHelpZh = `
`
const configureSetHelpEn = `
`

func NewConfigureSetCommand() (*cli.Command) {
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

	profileName, ok := c.Flags().GetValue("profile")
	if !ok {
		profileName = config.CurrentProfile
	}

	profile, ok := config.GetProfile(profileName)
	if !ok {
		profile = NewProfile(profileName)
	}

	mode, ok := c.Flags().GetValue(ModeFlag.Name)
	if ok {
		profile.Mode = AuthenticateMode(mode)
	} else {
		if profile.Mode == "" {
			profile.Mode = AK
		}
	}

	fs := c.Flags()
	switch profile.Mode {
	case AK:
		profile.AccessKeyId = fs.GetValueOrDefault(AccessKeyIdFlag.Name, profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault(AccessKeySecretFlag.Name, profile.AccessKeySecret)
	case StsToken:
		profile.AccessKeyId = fs.GetValueOrDefault(AccessKeyIdFlag.Name, profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault(AccessKeyIdFlag.Name, profile.AccessKeySecret)
		profile.StsToken = fs.GetValueOrDefault(StsTokenFlag.Name, profile.StsToken)
	case RamRoleArn:
		profile.AccessKeyId = fs.GetValueOrDefault(AccessKeyIdFlag.Name, profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault(AccessKeySecretFlag.Name, profile.AccessKeySecret)
		profile.RamRoleArn = fs.GetValueOrDefault(RamRoleArnFlag.Name, profile.RamRoleArn)
		profile.RoleSessionName = fs.GetValueOrDefault(RoleSessionNameFlag.Name, profile.RoleSessionName)
	case EcsRamRole:
		profile.RamRoleName = fs.GetValueOrDefault(RamRoleNameFlag.Name, profile.RamRoleName)
	case RsaKeyPair:
		profile.PrivateKey = fs.GetValueOrDefault(PrivateKeyFlag.Name, profile.PrivateKey)
		profile.KeyPairName = fs.GetValueOrDefault(KeyPairNameFlag.Name, profile.KeyPairName)
	}

	profile.RegionId = fs.GetValueOrDefault(RegionFlag.Name, profile.RegionId)
	profile.Language = fs.GetValueOrDefault(LanguageFlag.Name, profile.Language)
	profile.OutputFormat = "json" 	// fs.GetValueOrDefault("output", profile.OutputFormat)
	profile.Site = "china"			// fs.GetValueOrDefault("site", profile.Site)

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

