package lib

import (
	"fmt"
	"reflect"
	"strings"
)

// global public variable for formating help text
const (
	FormatTAB         = "    "
	MaxCommandNameLen = 18
	UsageTextChinese  = "用法: ossutil [command] [args...] [options...]\n请使用ossutil help command来显示command命令的帮助"
	UsageTextEnglish  = "Usage: ossutil [command] [args...] [options...]\nPlease use 'ossutil help command' to show help of command"
)

var specChineseHelp = SpecText{

	synopsisText: "获取命令的帮助文档",

	paramText: "[command]",

	syntaxText: ` 
    ossutil help [command]
`,

	detailHelpText: ` 
    该命令提供ossutil所有命令的帮助文档，或者针对用户输入的某具体命令提供它的帮助文档。

用法：

    该命令有两种用法：

    1) ossutil help
        该用法提供ossutil支持的所有命令的简介，对每个命令显示该命令的摘要和语法简介。

    2) ossutil help command
        该用法提供指定命令(command)的帮助文档，包括该命令的详细介绍、示例、可选参数。
`,

	sampleText: ` 
    ossutil help
    ossutil help help
    ossutil help ls
`,
}

var specEnglishHelp = SpecText{

	synopsisText: "Get help about commands",

	paramText: "[command]",

	syntaxText: ` 
    ossutil help [command]
`,

	detailHelpText: ` 
    The command provide the usage of all commands on which help is available, 
    or the usage of the specified command.

Usage:

    There are two usages:

    1) ossutil help
        The usage provides a summary of all commands, each command shows the
    synopsis and a simplified expression of the syntax.

    2) ossutil help command
        The usage provides help about the specified command, which contains
    a detailed description of the command, include samples and optional options.
`,

	sampleText: ` 
    ossutil help
    ossutil help help
    ossutil help ls
`,
}

// HelpCommand is the command format help text
type HelpCommand struct {
	command Command
}

var helpCommand = HelpCommand{
	command: Command{
		name:        "help",
		nameAlias:   []string{},
		minArgc:     0,
		maxArgc:     1,
		specChinese: specChineseHelp,
		specEnglish: specEnglishHelp,
		group:       GroupTypeAdditionalCommand,
		validOptionNames: []string{
			OptionLanguage,
			OptionLogLevel,
		},
	},
}

// function for RewriteLoadConfiger interface
func (hc *HelpCommand) rewriteLoadConfig(configFile string) error {
	// read config file, if error exist, do not print error
	var err error
	if hc.command.configOptions, err = LoadConfig(configFile); err != nil {
		hc.command.configOptions = OptionMapType{}
	}
	return nil
}

// function for FormatHelper interface
func (hc *HelpCommand) formatHelpForWhole() string {
	return hc.command.formatHelpForWhole()
}

func (hc *HelpCommand) formatIndependHelp() string {
	return hc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (hc *HelpCommand) Init(args []string, options OptionMapType) error {
	return hc.command.Init(args, options, hc)
}

// RunCommand simulate inheritance, and polymorphism
func (hc *HelpCommand) RunCommand() error {
	groupCommandMap, subCommandMap := hc.getCommandMap()
	if len(hc.command.args) == 0 {
		// ossutil help
		text := hc.formatWholeHelp(groupCommandMap)
		Output(text)
	} else {
		//ossutil help command
		if text, err := hc.formatCommandHelp(subCommandMap); err == nil {
			Output(text)
		} else {
			return err
		}
	}
	return nil
}

func (hc *HelpCommand) getCommandMap() (map[string][]interface{}, map[string]interface{}) {
	commandList := GetAllCommands()
	groupCommandMap := map[string][]interface{}{}
	subCommandMap := map[string]interface{}{}
	for _, cmd := range commandList {
		group := reflect.ValueOf(cmd).Elem().FieldByName("command").FieldByName("group").String()
		groupCommandMap[group] = append(groupCommandMap[group], cmd)
		name := reflect.ValueOf(cmd).Elem().FieldByName("command").FieldByName("name").String()
		subCommandMap[name] = cmd
	}
	return groupCommandMap, subCommandMap
}

func (hc *HelpCommand) formatWholeHelp(groupCommandMap map[string][]interface{}) string {
	if len(groupCommandMap) == 0 {
		return ""
	}

	commandsText := ""
	for _, group := range CommandGroups {
		commandList := groupCommandMap[group]
		if len(commandList) == 0 {
			continue
		}

		commandsText += group
		for _, cmd := range commandList {
			commandsText += cmd.(FormatHelper).formatHelpForWhole()
		}
	}
	return fmt.Sprintf("%s\n%s", hc.getUsageText(), commandsText)
}

func (hc *HelpCommand) getUsageText() string {
	val, _ := GetString(OptionLanguage, helpCommand.command.options)
	switch strings.ToLower(val) {
	case LEnglishLanguage:
		return UsageTextEnglish
	default:
		return UsageTextChinese
	}

}

func (hc *HelpCommand) formatCommandHelp(subCommandMap map[string]interface{}) (string, error) {
	subCommandName := hc.command.args[0]
	if cmd, ok := subCommandMap[subCommandName]; ok {
		return cmd.(FormatHelper).formatIndependHelp(), nil
	}
	return "", fmt.Errorf("no such command: \"%s\", please try \"help\" for more information", subCommandName)
}
