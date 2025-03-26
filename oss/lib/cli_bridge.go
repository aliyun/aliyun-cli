package lib

import (
	"fmt"
	"os"

	"github.com/alibabacloud-go/tea/tea"
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
		lrbCommand.command,
		replicationCommand.command,
		bucketCnameCommand.command,
		lcbCommand.command,
		bucketAccessMonitorCommand.command,
		bucketResourceGroupCommand.command,
	}

	for _, cmd := range cmds {
		result.AddSubCommand(NewCommandBridge(cmd))
	}
	return result
}

func NewCommandBridge(cmd Command) *cli.Command {

	result := &cli.Command{
		Name:     cmd.name,
		Usage:    cmd.specEnglish.syntaxText,
		Short:    i18n.T(cmd.specEnglish.synopsisText, cmd.specChinese.synopsisText),
		Long:     i18n.T(cmd.specEnglish.detailHelpText, cmd.specChinese.detailHelpText),
		KeepArgs: true,
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

// ParseAndGetEndpoint get oss endpoint from cli context
func ParseAndGetEndpoint(ctx *cli.Context, args []string) (string, error) {
	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return "", fmt.Errorf("config failed: %s", err.Error())
	}
	// try fetch endpoint from args
	if len(args) > 0 {
		for i, arg := range args {
			if arg == "--endpoint" {
				if i+1 < len(args) {
					return args[i+1], nil
				}
			}
		}
	}
	// try fetch region from args
	if len(args) > 0 {
		for i, arg := range args {
			if arg == "--region" {
				if i+1 < len(args) {
					return "oss-" + args[i+1] + ".aliyuncs.com", nil
				}
			}
		}
	}
	// check endpoint from flags
	if ep, ok := ctx.Flags().GetValue("endpoint"); !ok {
		return "oss-" + profile.RegionId + ".aliyuncs.com", nil
	} else {
		return ep, nil
	}
}

func ParseAndRunCommandFromCli(ctx *cli.Context, args []string) error {
	// 利用 parser 解析 flags，否则下文读不到
	parser := cli.NewParser(args, ctx)
	parser.ReadAll()

	profile, err := config.LoadProfileWithContext(ctx)
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	proxyHost, ok := ctx.Flags().GetValue("proxy-host")
	if !ok {
		proxyHost = ""
	}
	credential, err := profile.GetCredential(ctx, tea.String(proxyHost))
	if err != nil {
		return fmt.Errorf("can't get credential %s", err)
	}

	model, err := credential.GetCredential()
	if err != nil {
		return fmt.Errorf("can't get credential %s", err)
	}

	configs := make(map[string]string, 0)
	if model.AccessKeyId != nil {
		configs["access-key-id"] = *model.AccessKeyId
	}

	if model.AccessKeySecret != nil {
		configs["access-key-secret"] = *model.AccessKeySecret
	}

	if model.SecurityToken != nil {
		configs["sts-token"] = *model.SecurityToken
	}

	// read endpoint from flags
	endpoint, err := ParseAndGetEndpoint(ctx, args)
	if err != nil {
		return fmt.Errorf("parse endpoint failed: %s", err)
	}
	configs["endpoint"] = endpoint

	a2 := []string{"aliyun", "oss"}
	a2 = append(a2, ctx.Command().Name)
	a2 = append(a2, args...)
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
