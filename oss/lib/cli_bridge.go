package lib

import (
	"github.com/aliyun/aliyun-cli/cli"
	"github.com/aliyun/aliyun-cli/i18n"
	"github.com/aliyun/aliyun-cli/config"
	"fmt"
	"strings"
	"os"
	"time"
)

func NewOssCommand() *cli.Command {
	result :=  &cli.Command{
		Name: "oss",
		Usage: "aliyun oss [command] [args...] [options...]",
		Hidden: false,
		Short: i18n.T("Object Storage Service", "阿里云OSS对象存储"),
	}

	result.AddSubCommand(NewCommandBridge(&configCommand))
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
		Name: cmd.name,
		Usage: cmd.specEnglish.syntaxText,
		Short: i18n.T(cmd.specEnglish.synopsisText, cmd.specChinese.synopsisText),
		Long: i18n.T(cmd.specEnglish.detailHelpText, cmd.specChinese.detailHelpText),
		Run: func(ctx *cli.Context, args []string) error {
			return ParseAndRunCommandFromCli(ctx, cmd)
		},
	}

	for _, s := range cmd.validOptionNames {
		opt, ok := OptionMap[s]
		if !ok {
			// fmt.Printf("INIT ERROR: unknown oss options: %s\n", s)
			break
		}
		if result.Flags().Get(opt.name) == nil {
			result.Flags().Add(cli.Flag{
				Name:  opt.nameAlias[2:],
				Usage: i18n.T(opt.helpEnglish, opt.helpChinese),
				// Assignable: opt.optionType todo
			})
		}
	}
	return result
}

func ParseAndRunCommandFromCli(ctx *cli.Context, cmd *Command) error {
	profile, err := config.LoadCurrentProfile()
	if err != nil {
		return fmt.Errorf("config failed: %s", err.Error())
	}

	switch profile.Mode {
	case config.AK:
	case config.StsToken:
	default:
		return fmt.Errorf("oss only support AK|StsToken mode")
	}

	configs := make(map[string]string, 0)
	configs["access-key-id"] = profile.AccessKeyId
	configs["access-key-secret"] = profile.AccessKeySecret
	configs["sts-token"] = profile.StsToken
	configs["endpoint"] = "oss-" + profile.RegionId + ".aliyuncs.com"

	//if i18n.GetLanguage() == "zh" {
	//	configs[OptionLanguage] = "CH"
	//} else {
	//	configs[OptionLanguage] = "EN"
	//}
	// cmd.assignExternalConfig(configs)

	ts := time.Now().UnixNano()
	commandLine = strings.Join(os.Args[1:], " ")
	// os.Args = []string {"aliyun", "oss", "ls"}

	clearEnv()
	for k, v := range configs {
		if v != "" {
			os.Args = append(os.Args, "--" + k)
			os.Args = append(os.Args, v)
		}
	}

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