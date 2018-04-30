/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"io"
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
			doConfigureGet(w, c, args)
			return nil
		},
	}
	return cmd
}

func doConfigureGet(c *cli.Context, args []string) {
	config, err := LoadConfiguration(c.Writer())
	if err != nil {
		cli.Errorf(c.Writer(), "load configuration failed %s", err)
	}

	profile := config.GetCurrentProfile(c)

	if pn, ok := ProfileFlag.GetValue(); ok {
		profile, ok = config.GetProfile(pn)
		if !ok {
			cli.Errorf(c.Writer(), "profile %s not found!", pn)
		}
	}

	for _, arg := range args {
		switch arg {
		case ProfileFlag.Name:
			cli.Printf(c.Writer(), "profile=%s\n", profile.Name)
		case ModeFlag.Name:
			cli.Printf(c.Writer(), "mode=%s\n", profile.Mode)
		case AccessKeyIdFlag.Name:
			cli.Printf(c.Writer(), "access-key-id=%s\n", MosaicString(profile.AccessKeyId, 3))
		case AccessKeySecretFlag.Name:
			cli.Printf(c.Writer(), "access-key-secret=%s\n", MosaicString(profile.AccessKeySecret, 3))
		case StsTokenFlag.Name:
			cli.Printf(c.Writer(), "sts-token=%s\n", profile.StsToken)
		case RamRoleNameFlag.Name:
			cli.Printf(c.Writer(), "ram-role-name=%s\n", profile.RamRoleName)
		case RamRoleArnFlag.Name:
			cli.Printf(c.Writer(), "ram-role-arn=%s\n", profile.RamRoleArn)
		case RoleSessionNameFlag.Name:
			cli.Printf(c.Writer(), "role-session-name=%s\n", profile.RoleSessionName)
		case KeyPairNameFlag.Name:
			cli.Printf(c.Writer(), "key-pair-name=%s\n", profile.KeyPairName)
		case PrivateKeyFlag.Name:
			cli.Printf(c.Writer(), "private-key=%s\n", profile.PrivateKey)
		case RegionFlag.Name:
			cli.Printf(c.Writer(), profile.RegionId)
		case LanguageFlag.Name:
			cli.Printf(c.Writer(), "language=%s\n", profile.Language)
		}
	}
}
