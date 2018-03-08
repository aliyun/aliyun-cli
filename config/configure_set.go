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

	fs.Add(cli.Flag{Name: "access-key-id", Assignable: true,
		Usage: i18n.T("assign AccessKeyId, required in AK/StsToken/RamRoleArn mode", "")})

	fs.Add(cli.Flag{Name: "access-key-secret", Assignable: true,
		Usage: i18n.T("assign AccessKeySecret, required in AK/StsToken/RamRoleArn mode", "")})

	fs.Add(cli.Flag{Name: "sts-token", Assignable: true,
		Usage: i18n.T("assign StsToken, required in StsToken mode", "")})

	fs.Add(cli.Flag{Name: "ram-role-name", Assignable: true,
		Usage: i18n.T("assign RamRoleName, required in RamRoleArn/EcsRamRole mode", "")})

	fs.Add(cli.Flag{Name: "ram-role-arn", Assignable: true,
		Usage: i18n.T("assign RamRoleArn, required in RamRoleArn mode", "")})

	fs.Add(cli.Flag{Name: "role-session-name", Assignable: true,
		Usage: i18n.T("assign RoleSessionName, required in RamRoleArn mode", "")})

	fs.Add(cli.Flag{Name: "private-key", Assignable: true,
		Usage: i18n.T("assign PrivateKey, required in RsaKeyPair mode", "")})

	fs.Add(cli.Flag{Name: "key-pair-name", Assignable: true,
		Usage: i18n.T("assign KeyPairName, required in RsaKeyPair mode", "")})

	fs.Add(cli.Flag{Name: "region", Assignable: true,
		Usage: i18n.T("assign default Region", "")})

	fs.Add(cli.Flag{Name: "output", Assignable: true, Hidden: true,
		Usage: i18n.T("* assign output format, only support json", "")})

	fs.Add(cli.Flag{Name: "language", Assignable: true,
		Usage: i18n.T("assign language, support en/zh", "")})

	fs.Add(cli.Flag{Name: "site", Assignable: true,
		Usage: i18n.T("assign site, support china/international/japan", "")})

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

	mode, ok := c.Flags().GetValue("mode")
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
		profile.AccessKeyId = fs.GetValueOrDefault("access-key-id", profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault("access-key-secret", profile.AccessKeySecret)
	case StsToken:
		profile.AccessKeyId = fs.GetValueOrDefault("access-key-id", profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault("access-key-secret", profile.AccessKeySecret)
		profile.StsToken = fs.GetValueOrDefault("sts-token", profile.StsToken)
	case RamRoleArn:
		profile.AccessKeyId = fs.GetValueOrDefault("access-key-id", profile.AccessKeyId)
		profile.AccessKeySecret = fs.GetValueOrDefault("access-key-secret", profile.AccessKeySecret)
		profile.RamRoleArn = fs.GetValueOrDefault("ram-role-arn", profile.RamRoleArn)
		profile.RoleSessionName = fs.GetValueOrDefault("role-session-name", profile.RoleSessionName)
	case EcsRamRole:
		profile.RamRoleName = fs.GetValueOrDefault("ram-role-name", profile.RamRoleName)
	case RsaKeyPair:
		profile.PrivateKey = fs.GetValueOrDefault("private-key", profile.PrivateKey)
		profile.KeyPairName = fs.GetValueOrDefault("key-pair-name", profile.KeyPairName)
	}

	profile.RegionId = fs.GetValueOrDefault("region", profile.RegionId)
	profile.Language = fs.GetValueOrDefault("language", profile.Language)
	profile.OutputFormat = fs.GetValueOrDefault("output", profile.OutputFormat)
	profile.Site = fs.GetValueOrDefault("site", profile.Site)

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

