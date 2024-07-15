package lib

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

var specChineseUserQos = SpecText{
	synopsisText: "查询用户的qos配置",

	paramText: "[local_file] [options]",

	syntaxText: ` 
    ossutil user-qos --method get [local_file] [options]
`,
	detailHelpText: ` 
    user-qos命令通过设置method选项值为get可以查询用户的qos配置

用法:
    该命令只有一种用法:
    1) ossutil user-qos --method get [local_xml_file] [options]
       这个命令查询用户的qos配置,如果输入参数local_xml_file,qos配置将输出到该文件,否则输出到屏幕上
`,
	sampleText: ` 
    1) 查询用户的qos配置，结果输出到标准输出
       ossutil user-qos --method get
	
    2) 查询用户的qos配置，结果输出到本地文件
       ossutil user-qos --method get local_xml_file
`,
}

var specEnglishUserQos = SpecText{
	synopsisText: "Get user's qos configuration",

	paramText: "[local_file] [options]",

	syntaxText: ` 
    ossutil user-qos --method get [local_xml_file] [options]
`,
	detailHelpText: ` 
    user-qos command can get the user's qos configuration by set method option value to get

Usage:
    There are only one usage for this command:

    1) ossutil user-qos --method get [local_xml_file] [options]
       The command gets the user's qos configuration
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
`,
	sampleText: ` 
    1) get user's qos configuration to stdout
       ossutil user-qos --method get
	
    2) get user's qos configuration to local file
       ossutil user-qos --method get local_xml_file
`,
}

type UserQosCommand struct {
	command Command
}

var userQosCommand = UserQosCommand{
	command: Command{
		name:        "user-qos",
		nameAlias:   []string{"user-qos"},
		minArgc:     0,
		maxArgc:     1,
		specChinese: specChineseUserQos,
		specEnglish: specEnglishUserQos,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionLogLevel,
			OptionMethod,
			OptionPassword,
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionReadTimeout,
			OptionConnectTimeout,
			OptionSTSRegion,
			OptionSkipVerifyCert,
			OptionUserAgent,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
		},
	},
}

// function for FormatHelper interface
func (uqc *UserQosCommand) formatHelpForWhole() string {
	return uqc.command.formatHelpForWhole()
}

func (uqc *UserQosCommand) formatIndependHelp() string {
	return uqc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (uqc *UserQosCommand) Init(args []string, options OptionMapType) error {
	return uqc.command.Init(args, options, uqc)
}

// RunCommand simulate inheritance, and polymorphism
func (uqc *UserQosCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, uqc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "get" {
		return fmt.Errorf("--method value must be get")
	}

	return uqc.GetUserQos()
}

func (uqc *UserQosCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("user qos: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (uqc *UserQosCommand) GetUserQos() error {
	client, err := uqc.command.ossClient("")
	if err != nil {
		return err
	}

	qosRes, err := client.GetUserQoSInfo()
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(qosRes, "  ", "    ")
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(uqc.command.args) >= 1 {
		fileName := uqc.command.args[0]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := uqc.confirm(fileName)
			if !bConitnue {
				return nil
			}
		}

		outFile, err = os.OpenFile(fileName, os.O_CREATE|os.O_TRUNC|os.O_RDWR, 0660)
		if err != nil {
			return err
		}
		defer outFile.Close()
	} else {
		outFile = os.Stdout
	}

	outFile.Write([]byte(xml.Header))
	outFile.Write(output)

	fmt.Printf("\n\n")

	return nil
}
