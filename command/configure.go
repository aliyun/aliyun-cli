package command

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/core"
	"fmt"
)

var profile string
var ecsRamUser string

func NewConfigureCommand() (*cli.Command) {
	c := &cli.Command{
		Name: "configure",
		Short: "configure [get|set|list] --profile profileName",
		Run: func(c *cli.Command, args []string) error {
			if len(args) > 0 {
				c.PrintHelp(fmt.Errorf("unknown args"))
			}
			if profile == "" {
				profile = "default"
			}
			return doConfigure(profile)
		},
	}

	c.Flags().PersistentStringVar(&profile, "profile", "default", "--profile UserName")
	c.Flags().PersistentStringVar(&ecsRamUser, "ecs-ram-role", "", "--ecs-ram-role RAMName")

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
	fmt.Printf("configuring profile[%s]...", profileName)
	conf, err := core.LoadConfiguration()
	if err != nil {
		return err
	}

	cp, ok := conf.GetProfile(profileName)
	if !ok {
		cp = conf.NewProfile(profileName)
	}

	if ecsRamUser != "" {
		err = configureEcsRamUser(&cp, ecsRamUser)
	} else {
		err = configureAK(&cp)
	}

	fmt.Printf("Saving profile[%s] ...", profileName)
	conf.PutProfile(cp)
	conf.CurrentProfile = cp.Name
	err = core.SaveConfiguration(conf)

	if err != nil {
		return err
	}
	fmt.Printf("Done.\n")

	DoHello()
	return nil
}

func configureAK(cp *core.Profile) error  {
	cp.Mode = core.AK

	fmt.Printf("Aliyun Access Key Id [%s]: ", cp.AccessKeyId)
	cp.AccessKeyId = ReadInput(cp.AccessKeyId)
	fmt.Printf("Aliyun Access Key Secret [%s]: ", cp.AccessKeySecret)
	cp.AccessKeySecret = ReadInput(cp.AccessKeySecret)
	fmt.Printf("Default Region Id [%s]: ", cp.RegionId)
	cp.RegionId = ReadInput(cp.RegionId)
	fmt.Printf("Default Output Format [%s]: ", cp.OutputFormat)
	cp.OutputFormat = ReadInput(cp.OutputFormat)

	return nil
}

func configureEcsRamUser(cp *core.Profile, ecsRamUser string) error {
	cp.Mode = core.EcsRamUser
	cp.RamRole = ecsRamUser

	fmt.Printf("Aliyun Ecs Ram Role [%s]: ", cp.RamRole)
	cp.RamRole = ReadInput(cp.RamRole)
	fmt.Printf("Default Region Id [%s]: ", cp.RegionId)
	cp.RegionId = ReadInput(cp.RegionId)
	fmt.Printf("Default Output Format [%s]: ", cp.OutputFormat)
	cp.OutputFormat = ReadInput(cp.OutputFormat)

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


