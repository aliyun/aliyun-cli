/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"io"
)

const configureSetHelpEn = ``
const configureSetHelpZh = ``

func NewConfigureSetCommand(w io.Writer) *cli.Command {
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
			doConfigureSet(w, c)
			return nil
		},
		Writer: w,
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

func doConfigureSet(w io.Writer, c *cli.Context) {
	config, err := LoadConfiguration(w)
	if err != nil {
		cli.Errorf(w, "load configuration failed %s", err)
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
		profile.AccessKeyId = AccessKeyIdFlag.GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag.GetStringOrDefault(profile.AccessKeySecret)
	case StsToken:
		profile.AccessKeyId = AccessKeyIdFlag.GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeyIdFlag.GetStringOrDefault(profile.AccessKeySecret)
		profile.StsToken = StsTokenFlag.GetStringOrDefault(profile.StsToken)
	case RamRoleArn:
		profile.AccessKeyId = AccessKeyIdFlag.GetStringOrDefault(profile.AccessKeyId)
		profile.AccessKeySecret = AccessKeySecretFlag.GetStringOrDefault(profile.AccessKeySecret)
		profile.RamRoleArn = RamRoleArnFlag.GetStringOrDefault(profile.RamRoleArn)
		profile.RoleSessionName = RoleSessionNameFlag.GetStringOrDefault(profile.RoleSessionName)
	case EcsRamRole:
		profile.RamRoleName = RamRoleNameFlag.GetStringOrDefault(profile.RamRoleName)
	case RsaKeyPair:
		profile.PrivateKey = PrivateKeyFlag.GetStringOrDefault(profile.PrivateKey)
		profile.KeyPairName = KeyPairNameFlag.GetStringOrDefault(profile.KeyPairName)
	}

	profile.RegionId = RegionFlag.GetStringOrDefault(profile.RegionId)
	profile.Language = LanguageFlag.GetStringOrDefault(profile.Language)
	profile.OutputFormat = "json" // "output", profile.OutputFormat)
	profile.Site = "china"        // "site", profile.Site)
	profile.RetryTimeout = RetryTimeoutFlag.GetIntegerOrDefault(profile.RetryTimeout)
	profile.RetryCount = RetryCountFlag.GetIntegerOrDefault(profile.RetryCount)

	err = profile.Validate()
	if err != nil {
		cli.Errorf(w, "fail to set configuration: %s", err.Error())
		return
	}

	config.PutProfile(profile)
	config.CurrentProfile = profile.Name
	err = SaveConfiguration(config)
	if err != nil {
		cli.Errorf(w, "save configuration failed %s", err)
	}
}
