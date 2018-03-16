package lib

import (
	"github.com/aliyun/aliyun-cli/cli"
	"fmt"
	"github.com/aliyun/aliyun-cli/i18n"
	"encoding/json"
)

func NewOssCommand() *cli.Command {
	result :=  &cli.Command{
		Name: "oss",
		Usage: "oss ... ",
		Hidden: true,
		Short: i18n.T("oss commands", ""),
		Run: func(ctx *cli.Context, args []string) error {
			return nil
		},
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
	result.AddSubCommand(NewCommandBridge(&updateCommand))
	return result
}

func NewCommandBridge(a Commander) *cli.Command {
	cmd := a.GetCommand()
	result := &cli.Command{
		Name: cmd.name,
		Usage: cmd.specEnglish.synopsisText,
		Short: i18n.T(cmd.specEnglish.synopsisText, cmd.specChinese.synopsisText),
		Long: i18n.T(cmd.specEnglish.detailHelpText, cmd.specChinese.detailHelpText),
		Run: func(ctx *cli.Context, args []string) error {
			j, _ := json.MarshalIndent(cmd, "", "\t")
			fmt.Printf("%s\n",string(j))
			return nil
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
				Name:  opt.name,
				Usage: i18n.T(opt.helpEnglish, opt.helpChinese),
				// Assignable: opt.optionType todo
			})
		}
	}
	return result
}

