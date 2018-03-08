/*
 * Copyright (C) 2017-2018 Alibaba Group Holding Limited
 */
package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"fmt"
	"strings"
	"io/ioutil"
	"text/tabwriter"
	"os"
	"github.com/aliyun/aliyun-cli/i18n"
)

var profile string
var mode string

func NewConfigureCommand() (*cli.Command) {
	c := &cli.Command{
		Name: "configure",
		Short: i18n.T("configure credential and settings", "配置身份认证和其他信息"),
		Usage: "configure --mode <AuthenticateMode> --profile <profileName>",
		Run: func(c *cli.Context, args []string) error {
			if len(args) > 0 {
				return cli.NewInvalidCommandError(args[0], c)
			}
			if profile == "" {
				profile = "default"
			}
			return doConfigure(profile)
		},
	}

	f := c.Flags().PersistentStringVar(&profile, "profile", "default",
		i18n.T("use `--profile <profileName>` to select profile",
			"使用 `--profile <profileName>` 来指定操作的配置集"))
	f.Persistent = true

	c.Flags().PersistentStringVar(&mode, "mode", "AK",
		i18n.T("use `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` to assign authenticate mode",
			"使用 `--mode {AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair}` 指定认证方式"))

	c.AddSubCommand(NewConfigureGetCommand())
	c.AddSubCommand(NewConfigureSetCommand())
	c.AddSubCommand(&cli.Command{
		Name: "list",
		Run: func(c *cli.Context, args []string) error {
			doConfigureList()
			return nil
		},
	})

	return c
}

func doConfigure(profileName string) error {
	conf, err := LoadConfiguration()
	if err != nil {
		return err
	}

	cp, ok := conf.GetProfile(profileName)
	if !ok {
		cp = conf.NewProfile(profileName)
	}

	fmt.Printf("Configuring profile '%s' ...\n", profileName)
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
	fmt.Printf("Default Output Format [%s]: json", cp.OutputFormat)
	// cp.OutputFormat = ReadInput(cp.OutputFormat)
	cp.OutputFormat = "json"
	fmt.Printf("Default Language [zh|en] %s: ", cp.Language)
	cp.Language = ReadInput(cp.Language)
	if cp.Language != "zh" && cp.Language != "en" {
		cp.Language = "en"
	}
	fmt.Printf("User site: [china|international|japan] %s", cp.Site)
	cp.Site = ReadInput(cp.Site)

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

func doConfigureList() {
	conf, err := LoadConfiguration()
	if err != nil {
		cli.Errorf("ERROR: load configure failed: %v\n", err)
	}
	w := tabwriter.NewWriter(os.Stdout, 8, 0, 1, ' ', 0)
	fmt.Fprint(w, "Profile\t| Credential \t| Valid\t| Region\t| Language\n")
	fmt.Fprint(w, "---------\t| ------------------\t| -------\t| ----------------\t| --------\n")
	for _, profile := range conf.Profiles {
		name := profile.Name
		if name == conf.CurrentProfile {
			name = name + " *"
		}
		err := profile.Validate()
		valid := "Valid"
		if err != nil {
			valid = "Invalid"
		}

		cred := ""
		switch profile.Mode {
		case AK:
			cred = "AK:" + "***" + GetLastChars(profile.AccessKeyId, 3)
		case StsToken:
			cred = "StsToken:" +  "***" + GetLastChars(profile.AccessKeyId, 3)
		case RamRoleArn:
			cred = "RamRoleArn:" +  "***" + GetLastChars(profile.AccessKeyId, 3)
		case EcsRamRole:
			cred = "EcsRamRole:" + profile.RamRoleName
		case RsaKeyPair:
			cred = "RsaKeyPair:" + profile.KeyPairName
		}
		fmt.Fprintf(w, "%s\t| %s\t| %s\t| %s\t| %s\n", name, cred, valid, profile.RegionId, profile.Language)
	}
	w.Flush()
}

func configureAK(cp *Profile) error  {
	fmt.Printf("Access Key Id [%s]: ", MosaicString(cp.AccessKeyId, 3))
	cp.AccessKeyId = ReadInput(cp.AccessKeyId)
	fmt.Printf("Access Key Secret [%s]: ", MosaicString(cp.AccessKeySecret, 3))
	cp.AccessKeySecret = ReadInput(cp.AccessKeySecret)
	return nil
}

func configureStsToken(cp *Profile) error  {
	err := configureAK(cp)
	if err != nil {
		return err
	}
	fmt.Printf("Sts Token [%s]: ", cp.StsToken)
	cp.StsToken = ReadInput(cp.StsToken)
	return nil
}

func configureRamRoleArn(cp *Profile) error  {
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

func ReadInput(defaultValue string) (string) {
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
