package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/credentials-go/credentials"
)

func NewOssCommand() *cli.Command {
	result := &cli.Command{
		Name:   "oss",
		Usage:  "oss [command] [args...] [options...]",
		Hidden: false,
		Short:  i18n.T("Object Storage Service", "阿里云OSS对象存储"),
	}

	cmds := []Command{
		helpCommand.command,
		configCommand.command,
		makeBucketCommand.command,
		listCommand.command,
		removeCommand.command,
		statCommand.command,
		setACLCommand.command,
		setMetaCommand.command,
		copyCommand.command,
		restoreCommand.command,
		createSymlinkCommand.command,
		readSymlinkCommand.command,
		signURLCommand.command,
		hashCommand.command,
		updateCommand.command,
		probeCommand.command,
		mkdirCommand.command,
		corsCommand.command,
		bucketLogCommand.command,
		bucketRefererCommand.command,
		listPartCommand.command,
		allPartSizeCommand.command,
		appendFileCommand.command,
		catCommand.command,
		bucketTagCommand.command,
		bucketEncryptionCommand.command,
		corsOptionsCommand.command,
		bucketLifeCycleCommand.command,
		bucketWebsiteCommand.command,
		bucketQosCommand.command,
		userQosCommand.command,
		bucketVersioningCommand.command,
		duSizeCommand.command,
		bucketPolicyCommand.command,
		requestPaymentCommand.command,
		objectTagCommand.command,
		bucketInventoryCommand.command,
		revertCommand.command,
		syncCommand.command,
		wormCommand.command,
	}

	for _, cmd := range cmds {
		result.AddSubCommand(NewCommandBridge(cmd))
	}
	return result
}

func NewCommandBridge(cmd Command) *cli.Command {

	result := &cli.Command{
		Name:  cmd.name,
		Usage: cmd.specEnglish.syntaxText,
		Short: i18n.T(cmd.specEnglish.synopsisText, cmd.specChinese.synopsisText),
		Long:  i18n.T(cmd.specEnglish.detailHelpText, cmd.specChinese.detailHelpText),
		Run: func(ctx *cli.Context, args []string) error {
			return ParseAndRunCommandFromCli(ctx, args)
		},
	}

	config.AddFlags(result.Flags())

	for _, s := range cmd.validOptionNames {
		opt, ok := OptionMap[s]
		if !ok {
			continue
		}
		name := opt.nameAlias[2:]

		shorthand := rune(0)
		if len(opt.name) > 0 {
			shorthand = rune(opt.name[1])
		}

		if result.Flags().Get(name) == nil {
			result.Flags().Add(&cli.Flag{
				Name:      name,
				Shorthand: shorthand,
				Short:     i18n.T(opt.helpEnglish, opt.helpChinese),
				// Assignable: opt.optionType todo
			})
		}
	}
	return result
}

func ParseAndRunCommandFromCli(ctx *cli.Context, args []string) error {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	accessKeyID, accessSecret, stsToken, err := getSessionCredential(&profile)
	if err != nil {
		return fmt.Errorf("can't get credential %s", err)
	}

	configs := make(map[string]string, 0)
	configs["access-key-id"] = accessKeyID
	configs["access-key-secret"] = accessSecret
	configs["sts-token"] = stsToken

	if ep, ok := ctx.Flags().GetValue("endpoint"); !ok {
		configs["endpoint"] = "oss-" + profile.RegionId + ".aliyuncs.com"
	} else {
		configs["endpoint"] = ep
	}

	//if i18n.GetLanguage() == "zh" {
	//	configs[OptionLanguage] = "CH"
	//} else {
	//	configs[OptionLanguage] = "EN"
	//}

	a2 := []string{"aliyun", "oss"}
	a2 = append(a2, ctx.Command().Name)
	for _, a := range args {
		a2 = append(a2, a)
	}
	configFlagSet := cli.NewFlagSet()
	config.AddFlags(configFlagSet)

	for _, f := range ctx.Flags().Flags() {
		if configFlagSet.Get(f.Name) != nil {
			continue
		}
		if f.IsAssigned() {
			a2 = append(a2, "--"+f.Name)
			if s2, ok := f.GetValue(); ok && s2 != "" {
				a2 = append(a2, s2)
			}
		}
	}

	for k, v := range configs {
		if v != "" {
			a2 = append(a2, "--"+k)
			a2 = append(a2, v)
		}
	}
	os.Args = a2[1:]
	return ParseAndRunCommand()

}

func getSessionCredential(profile *config.Profile) (string, string, string, error) {
	var conf *credentials.Config
	switch profile.Mode {
	case config.AK:
		conf = &credentials.Config{
			Type:            tea.String("access_key"),
			AccessKeyId:     &profile.AccessKeyId,
			AccessKeySecret: &profile.AccessKeySecret,
		}
	case config.StsToken:
		conf = &credentials.Config{
			Type:            tea.String("sts"),
			AccessKeyId:     &profile.AccessKeyId,
			AccessKeySecret: &profile.AccessKeySecret,
			SecurityToken:   &profile.StsToken,
		}
	case config.RamRoleArn:
		conf = &credentials.Config{
			Type:                  tea.String("ram_role_arn"),
			AccessKeyId:           &profile.AccessKeyId,
			AccessKeySecret:       &profile.AccessKeySecret,
			RoleArn:               &profile.RamRoleArn,
			RoleSessionName:       &profile.RoleSessionName,
			Policy:                tea.String(""),
			RoleSessionExpiration: &profile.ExpiredSeconds,
		}
	case config.EcsRamRole:
		conf = &credentials.Config{
			Type:     tea.String("ecs_ram_role"),
			RoleName: &profile.RamRoleName,
		}
	case config.RamRoleArnWithEcs:
		client, _ := sdk.NewClientWithEcsRamRole(profile.RegionId, profile.RamRoleName)
		return profile.GetSessionCredential(client)
	case config.External:
		args := strings.Fields(profile.ProcessCommand)
		cmd := exec.Command(args[0], args[1:]...)
		buf, err := cmd.CombinedOutput()
		if err != nil {
			return "", "", "", err
		}
		cp := &config.Profile{}
		err = json.Unmarshal(buf, cp)
		if err != nil {
			fmt.Println(cp.ProcessCommand)
			fmt.Println(string(buf))
			return "", "", "", err
		}
		return getSessionCredential(cp)
	}
	credential, err := credentials.NewCredential(conf)
	if err != nil {
		return "", "", "", err
	}
	var lastErr error
	accessKeyID, err := credential.GetAccessKeyId()
	if err != nil {
		lastErr = err
	}
	accessSecret, err := credential.GetAccessKeySecret()
	if err != nil {
		lastErr = err
	}
	stsToken, err := credential.GetSecurityToken()
	if err != nil {
		lastErr = err
	}
	if lastErr != nil {
		return "", "", "", lastErr
	}
	return *accessKeyID, *accessSecret, *stsToken, nil
}
