package lib

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
)

func NewOssCommand() *cli.Command {
	result := &cli.Command{
		Name:   "oss",
		Usage:  "oss [command] [args...] [options...]",
		Hidden: false,
		Short:  i18n.T("Object Storage Service", "阿里云OSS对象存储"),
	}

	result.AddSubCommand(NewCommandBridge(&allPartSizeCommand))
	result.AddSubCommand(NewCommandBridge(&appendFileCommand))
	result.AddSubCommand(NewCommandBridge(&corsCommand))
	result.AddSubCommand(NewCommandBridge(&bucketEncryptionCommand))
	result.AddSubCommand(NewCommandBridge(&bucketLifeCycleCommand))
	result.AddSubCommand(NewCommandBridge(&bucketLogCommand))
	result.AddSubCommand(NewCommandBridge(&bucketPolicyCommand))
	result.AddSubCommand(NewCommandBridge(&bucketQosCommand))
	result.AddSubCommand(NewCommandBridge(&bucketRefererCommand))
	result.AddSubCommand(NewCommandBridge(&bucketTagCommand))
	result.AddSubCommand(NewCommandBridge(&bucketVersioningCommand))
	result.AddSubCommand(NewCommandBridge(&bucketWebsiteCommand))
	result.AddSubCommand(NewCommandBridge(&catCommand))
	result.AddSubCommand(NewCommandBridge(&corsOptionsCommand))
	result.AddSubCommand(NewCommandBridge(&copyCommand))
	result.AddSubCommand(NewCommandBridge(&createSymlinkCommand))
	result.AddSubCommand(NewCommandBridge(&duSizeCommand))
	result.AddSubCommand(NewCommandBridge(&hashCommand))
	result.AddSubCommand(NewCommandBridge(&helpCommand))
	result.AddSubCommand(NewCommandBridge(&listPartCommand))
	result.AddSubCommand(NewCommandBridge(&listCommand))
	result.AddSubCommand(NewCommandBridge(&makeBucketCommand))
	result.AddSubCommand(NewCommandBridge(&mkdirCommand))
	result.AddSubCommand(NewCommandBridge(&objectTagCommand))
	result.AddSubCommand(NewCommandBridge(&probeCommand))
	result.AddSubCommand(NewCommandBridge(&readSymlinkCommand))
	result.AddSubCommand(NewCommandBridge(&requestPaymentCommand))
	result.AddSubCommand(NewCommandBridge(&restoreCommand))
	result.AddSubCommand(NewCommandBridge(&removeCommand))
	result.AddSubCommand(NewCommandBridge(&setACLCommand))
	result.AddSubCommand(NewCommandBridge(&setMetaCommand))
	result.AddSubCommand(NewCommandBridge(&signURLCommand))
	result.AddSubCommand(NewCommandBridge(&statCommand))
	result.AddSubCommand(NewCommandBridge(&userQosCommand))
	return result
}

func NewCommandBridge(a Commander) *cli.Command {
	cmd := a.GetCommand()
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

	sc, err := profile.GetSessionCredential()
	if err != nil {
		return fmt.Errorf("can't get credential %s", err)
	}

	configs := make(map[string]string, 0)
	configs["access-key-id"] = sc.AccessKeyId
	configs["access-key-secret"] = sc.AccessKeySecret
	configs["sts-token"] = sc.StsToken

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

	ts := time.Now().UnixNano()
	commandLine = strings.Join(os.Args[1:], " ")
	// os.Args = []string {"aliyun", "oss", "ls"}

	clearEnv()
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

	os.Args = a2
	// cli.Noticef("%v", os.Args)

	args, options, err := ParseArgOptions()
	if err != nil {
		return err
	}

	args = args[1:]
	// fmt.Printf("%v", args)
	showElapse, err := RunCommand(args, options)
	if err != nil {
		return err
	}

	if showElapse {
		te := time.Now().UnixNano()
		fmt.Printf("%.6f(s) elapsed\n", float64(te-ts)/1e9)
		return nil
	}
	return nil
}
