package config

import (
	"github.com/aliyun/aliyun-cli/cli"
	"fmt"
	"strings"
	"io/ioutil"
)

var profile string
var mode string

func NewConfigureCommand() (*cli.Command) {
	c := &cli.Command{
		Name: "configure",
		Short: "configure credential",
		Usage: "configure --mode certificatedMode --profile profileName",
		Run: func(c *cli.Context, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unknown args")
			}
			if profile == "" {
				profile = "default"
			}
			return doConfigure(profile)
		},
	}

	f := c.Flags().StringVar(&profile, "profile", "default", "--profile ProfileName")
	f.Persistent = true
	c.Flags().StringVar(&mode, "mode", "AK", "--mode [AK|StsToken|RamRoleArn|EcsRamRole|RsaKeyPair]")

	//c.AddSubCommand(&cli.Command{
	//	Name: "get",
	//	Short: "",
	//	Run: func(c *cli.Command, args []string) error {
	//		profile, _ := c.Flags().GetValue("profile")
	//		return doConfigure(profile)
	//	},
	//})
	//
	//c.AddSubCommand(&cli.Command{
	//	Name: "set",
	//	Run: func(cmd *cli.Command, args []string) error {
	//		profile, _ := c.Flags().GetValue("profile")
	//		return doSetConfigure()
	//	},
	//})
	//c.AddSubCommand(&cli.Command{
	//	Name: "list",
	//	Run: func(cmd *cli.Command, args []string) error {
	//		// profile, _ := c.Flags().GetValue("profile")
	//		// return true, nil
	//	},
	//})

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

	fmt.Printf("Configuring profile '%s'...\n", profileName)
	if mode != "" {
		switch CertificateMode(mode) {
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
			return fmt.Errorf("unexcepted certificated mode: %s", mode)
		}
	} else {
		configureAK(&cp)
	}

	//
	// configure common
	fmt.Printf("Default Region Id [%s]: ", cp.RegionId)
	cp.RegionId = ReadInput(cp.RegionId)
	fmt.Printf("Default Output Format [%s]: ", cp.OutputFormat)
	cp.OutputFormat = ReadInput(cp.OutputFormat)

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

