/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/cli"
	"fmt"
)

const configureGetHelpEn = `
`
const configureGetHelpZh = `
`

func NewConfigureGetCommand() (*cli.Command) {
	cmd := &cli.Command{
		Name:  "get",
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
	config, err := LoadConfiguration()
	if err != nil {
		cli.Errorf("load configuration failed %s", err)
	}

	profile := config.GetCurrentProfile()

	if pn, ok := c.Flags().GetValue("profile"); ok {
		profile, ok = config.GetProfile(pn)
		if !ok {
			cli.Errorf("profile %s not found!", pn)
		}
	}

	for _, arg := range args {
		switch arg {
		case "profile":
			fmt.Printf("profile=%s\n", profile.Name)
		case "mode":
			fmt.Printf("mode=%s\n", profile.Mode)
		case "access-key-id":
			fmt.Printf("access-key-id=%s\n", MosaicString(profile.AccessKeyId, 3))
		case "access-key-secret":
			fmt.Printf("access-key-secret=%s\n", MosaicString(profile.AccessKeySecret, 3))
		case "sts-token":
			fmt.Printf("sts-token=%s\n", profile.StsToken)
		case "ram-role-name":
			fmt.Printf("ram-role-name=%s\n", profile.RamRoleName)
		case "ram-role-arn":
			fmt.Printf("ram-role-arn=%s\n", profile.RamRoleArn)
		case "role-session-name":
			fmt.Printf("role-session-name=%s\n", profile.RoleSessionName)
		case "key-pair-name":
			fmt.Printf("key-pair-name=%s\n", profile.KeyPairName)
		case "private-key":
			fmt.Printf("private-key=%s\n", profile.PrivateKey)
		case "region":
			fmt.Printf("region=%s\n", profile.RegionId)
		case "output":
			fmt.Printf("output=%s\n", profile.OutputFormat)
		case "language":
			fmt.Printf("language=%s\n", profile.Language)
		}
	}
}