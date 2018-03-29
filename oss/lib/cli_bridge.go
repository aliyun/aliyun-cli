package lib

import (
	"fmt"
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/config"
	"github.com/aliyun/aliyun-cli/i18n"
	"os"
	"strings"
	"time"
)

func NewOssCommand() *cli.Command {
	result := &cli.Command{
		Name:   "oss",
		Usage:  "aliyun oss [command] [args...] [options...]",
		Hidden: false,
		Short:  i18n.T("Object Storage Service", "阿里云OSS对象存储"),
	}

	result.AddSubCommand(NewCommandBridge(&makeBucketCommand))
	result.AddSubCommand(NewCommandBridge(&listCommand))
	result.AddSubCommand(NewCommandBridge(&removeCommand))
	result.AddSubCommand(NewCommandBridge(&statCommand))
	result.AddSubCommand(NewCommandBridge(&setACLCommand))
	result.AddSubCommand(NewCommandBridge(&setMetaCommand))
	result.AddSubCommand(NewCommandBridge(&copyCommand))
	result.AddSubCommand(NewCommandBridge(&restoreCommand))
	result.AddSubCommand(NewCommandBridge(&createSymlinkCommand))
	result.AddSubCommand(NewCommandBridge(&readSymlinkCommand))
	result.AddSubCommand(NewCommandBridge(&signURLCommand))
	result.AddSubCommand(NewCommandBridge(&hashCommand))
	result.AddSubCommand(NewCommandBridge(&helpCommand))
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

	for _, s := range cmd.validOptionNames {
		opt, ok := OptionMap[s]
		if !ok {
			// fmt.Printf("INIT ERROR: unknown oss options: %s\n", s)
			break
		}
		name := opt.nameAlias[2:]

		shorthand := ""
		if len(opt.name) > 0 {
			shorthand = opt.name[1:]
		}

		if result.Flags().Get(name, "") == nil {
			result.Flags().Add(cli.Flag{
				Name:  name,
				Shorthand: shorthand,
				Usage: i18n.T(opt.helpEnglish, opt.helpChinese),
				// Assignable: opt.optionType todo
			})
		}
	}
	return result
}

func ParseAndRunCommandFromCli(ctx *cli.Context, args []string) error {
	profile, err := config.LoadCurrentProfile()
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
	configs["endpoint"] = "oss-" + profile.RegionId + ".aliyuncs.com"

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
		if configFlagSet.Get(f.Name, f.Shorthand) != nil {
			continue
		}
		if f.IsAssigned() {
			a2 = append(a2, "--"+f.Name)
			if f.GetValue() != "" {
				a2 = append(a2, f.GetValue())
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
