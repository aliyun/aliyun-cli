/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewConfigureSetCommand() (*cli.Command) {
	cmd := &cli.Command{
		Name: "set",
		Short: i18n.T("set config in command mode", ""),
		Run: func(c *cli.Context, args []string) error {
			doConfigureSet(c)
			return nil
		},
	}

	//Name            string          `json:"name"`
	//Mode            CertificateMode `json:"mode"`
	//AccessKeyId     string          `json:"access_key_id"`
	//AccessKeySecret string          `json:"access_key_secret"`
	//StsToken		string			`json:"sts_token"`
	//RamRoleName		string          `json:"ram_role_name"`
	//RamRoleArn		string			`json:"ram_role_arn"`
	//RoleSessionName	string 			`json:"ram_session_name"`
	//PrivateKey		string 			`json:"private_key"`
	//KeyPairName		string 			`json:"key_pair_name"`
	//ExpiredSeconds	int				`json:"expired_seconds"`
	//Verified		string			`json:"verified"`
	//RegionId        string          `json:"region_id"`
	//OutputFormat    string          `json:"output_format"`
	//Language        string          `json:"language"`

	fs := cmd.Flags()
	// fs.Add(cli.Flag{Name: "output", Assignable: true, Usage: "set output "})
	fs.Add(cli.Flag{Name: "access-key-id", Assignable: true, Usage: i18n.T("--access-key-id <access-key-id>", "")})
	fs.Add(cli.Flag{Name: "access-key-secret", Assignable: true, Usage: i18n.T("--access-key-secret <access-key-secret>", "")})
	fs.Add(cli.Flag{Name: "sts-token", Assignable: true, Usage: i18n.T("--sts-token <sts-token>", "")})
	fs.Add(cli.Flag{Name: "ram-role-name", Assignable: true, Usage: i18n.T("--ram-role-name <ram-role-name>", "")})
	fs.Add(cli.Flag{Name: "ram-role-arn", Assignable: true, Usage: i18n.T("--ram-role-arn <ram-role-arn>", "")})
	fs.Add(cli.Flag{Name: "role-session-name", Assignable: true, Usage: i18n.T("--role-session-name <role-session-name>", "")})
	fs.Add(cli.Flag{Name: "private-key", Assignable: true, Usage: i18n.T("--private-key <private-key>", "")})
	fs.Add(cli.Flag{Name: "key-pair-name", Assignable: true, Usage: i18n.T("--key-pair-name <key-pair-name>", "")})
	fs.Add(cli.Flag{Name: "region", Assignable: true, Usage: i18n.T("--region <region>", "")})
	fs.Add(cli.Flag{Name: "output", Assignable: true, Usage: i18n.T("--output [json]", "")})
	fs.Add(cli.Flag{Name: "language", Assignable: true, Usage: i18n.T("--language [en|zh]", "")})

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
		profile.Mode = CertificateMode(mode)
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

