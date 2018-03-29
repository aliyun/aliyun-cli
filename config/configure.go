/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"io/ioutil"
	"strings"
)

func NewConfigureCommand() *cli.Command {
	c := &cli.Command{
		Name: "configure",
		Short: i18n.T(
			"configure credential and settings",
			"配置身份认证和其他信息"),
		Usage: "configure --mode <AuthenticateMode> --profile <profileName>",
		Run: func(ctx *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], ctx)
			}
			profileName, _ := ctx.Flags().GetValue(ProfileFlag.Name)
			mode, _ := ctx.Flags().GetValue(ModeFlag.Name)

			return doConfigure(profileName, mode)
		},
	}

	c.Flags().Add(ProfileFlag)
	c.Flags().Add(ModeFlag)

	c.AddSubCommand(NewConfigureGetCommand())
	c.AddSubCommand(NewConfigureSetCommand())
	c.AddSubCommand(NewConfigureListCommand())
	c.AddSubCommand(NewConfigureDeleteCommand())
	return c
}

func doConfigure(profileName string, mode string) error {
	conf, err := LoadConfiguration()
	if err != nil {
		return err
	}

	cp, ok := conf.GetProfile(profileName)
	if !ok {
		cp = conf.NewProfile(profileName)
	}

	fmt.Printf("Configuring profile '%s' in '%s' authenticate mode...\n", profileName, mode)
	if mode != "" {
		switch AuthenticateMode(mode) {
		case AK:
			cp.Mode = AK
			configureAK(&cp)
		case StsToken:
			cp.Mode = StsToken
			configureStsToken(&cp)
		case RamRoleArn:
			cp.Mode = RamRoleArn
			configureRamRoleArn(&cp)
		case EcsRamRole:
			cp.Mode = EcsRamRole
			configureEcsRamRole(&cp)
		case RsaKeyPair:
			cp.Mode = RsaKeyPair
			configureRsaKeyPair(&cp)
		default:
			return fmt.Errorf("unexcepted authenticate mode: %s", mode)
		}
	} else {
		configureAK(&cp)
	}

	//
	// configure common
	fmt.Printf("Default Region Id [%s]: ", cp.RegionId)
	cp.RegionId = ReadInput(cp.RegionId)
	fmt.Printf("Default Output Format [%s]: json (Only support json))\n", cp.OutputFormat)
	// cp.OutputFormat = ReadInput(cp.OutputFormat)
	cp.OutputFormat = "json"

	fmt.Printf("Default Language [zh|en] %s: ", cp.Language)
	cp.Language = ReadInput(cp.Language)
	if cp.Language != "zh" && cp.Language != "en" {
		cp.Language = "en"
	}

	//fmt.Printf("User site: [china|international|japan] %s", cp.Site)
	//cp.Site = ReadInput(cp.Site)

	fmt.Printf("Saving profile[%s] ...", profileName)
	conf.PutProfile(cp)
	conf.CurrentProfile = cp.Name
	err = SaveConfiguration(conf)

	if err != nil {
		return err
	}
	fmt.Printf("Done.\n")

	DoHello(&cp)
	return nil
}

func configureAK(cp *Profile) error {
	fmt.Printf("Access Key Id [%s]: ", MosaicString(cp.AccessKeyId, 3))
	cp.AccessKeyId = ReadInput(cp.AccessKeyId)
	fmt.Printf("Access Key Secret [%s]: ", MosaicString(cp.AccessKeySecret, 3))
	cp.AccessKeySecret = ReadInput(cp.AccessKeySecret)
	return nil
}

func configureStsToken(cp *Profile) error {
	err := configureAK(cp)
	if err != nil {
		return err
	}
	fmt.Printf("Sts Token [%s]: ", cp.StsToken)
	cp.StsToken = ReadInput(cp.StsToken)
	return nil
}

func configureRamRoleArn(cp *Profile) error {
	err := configureAK(cp)
	if err != nil {
		return err
	}
	fmt.Printf("Ram Role Arn [%s]: ", cp.RamRoleArn)
	cp.RamRoleArn = ReadInput(cp.RamRoleArn)
	fmt.Printf("Role Session Name [%s]: ", cp.RoleSessionName)
	cp.RoleSessionName = ReadInput(cp.RoleSessionName)
	cp.ExpiredSeconds = 900
	return nil
}

func configureEcsRamRole(cp *Profile) error {
	fmt.Printf("Ecs Ram Role [%s]: ", cp.RamRoleName)
	cp.RamRoleName = ReadInput(cp.RamRoleName)
	return nil
}

func configureRsaKeyPair(cp *Profile) error {
	fmt.Printf("Rsa Private Key File: ")
	keyFile := ReadInput("")
	buf, err := ioutil.ReadFile(keyFile)
	if err != nil {
		return fmt.Errorf("read key file %s failed %v", keyFile, err)
	}
	cp.PrivateKey = string(buf)
	fmt.Printf("Rsa Key Pair Name: ")
	cp.KeyPairName = ReadInput("")
	cp.ExpiredSeconds = 900
	return nil
}

func ReadInput(defaultValue string) string {
	var s string
	fmt.Scanf("%s\n", &s)
	if s == "" {
		return defaultValue
	}
	return s
}

func MosaicString(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return strings.Repeat("*", r) + s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}

func GetLastChars(s string, lastChars int) string {
	r := len(s) - lastChars
	if r > 0 {
		return s[r:]
	} else {
		return strings.Repeat("*", len(s))
	}
}
