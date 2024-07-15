package lib

import (
	"encoding/xml"
	"fmt"
	"strconv"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseWorm = SpecText{
	synopsisText: "设置、删除、修改、提交bucket的Worm配置",

	paramText: "command_name bucket_url [days] [wormId] [options]",

	syntaxText: ` 
    ossutil worm init  oss://bucket days
    ossutil worm abort oss://bucket
    ossutil worm complete oss://bucket wormId
    ossutil worm extend oss://bucket days wormId
    ossutil worm get oss://bucket

`,
	detailHelpText: ` 
    worm命令通过设置第一个参数为init、abort、complete、extend、get,可以创建、删除、提交、修改或者查询bucket的worm配置

用法:
    该命令有五种用法:
	
    1) ossutil worm init oss://bucket days
        这个命令创建worm配置，Object的保留天数为days
	
    2) ossutil worm abort oss://bucket
        这个命令删除bucket的Worm配置
	
    3) ossutil worm complete oss://bucket wormId
        这个命令提交worm配置，成功后worm状态将由InProgress变为Locked

    4) ossutil worm extend oss://bucket days wormId
        这个命令修改worm配置，将Object的保留天数修改为days
    
    5) ossutil worm get oss://bucket
        这个命令查询worm配置
    
`,
	sampleText: ` 
`,
}

var specEnglishWorm = SpecText{
	synopsisText: "set、delete、complete、get bucket's worm configuration",

	paramText: "command_name bucket_url [days] [wormId] [options]",

	syntaxText: ` 
    ossutil worm init  oss://bucket days
    ossutil worm abort oss://bucket
    ossutil worm complete oss://bucket wormId
    ossutil worm extend oss://bucket days wormId
    ossutil worm get oss://bucket

`,
	detailHelpText: ` 
    The worm command can create, delete, complete, modify or get the worm configuration of the bucket 
    by setting the first parameter to init, abort, complete, extend, and get

Usage:
    There are 5 usages for this command:
	
    1) ossutil worm init oss://bucket days
       This command creates a worm configuration, the object's retention period is days
	
    2) ossutil worm abort oss://bucket
       This command deletes the worm configuration of the bucket
	
    3) ossutil worm complete oss://bucket wormId
       This command complete the worm configuration. 
       After success, the worm status will change from InProgress to Locked

    4) ossutil worm extend oss://bucket days wormId
       This command modifies the worm configuration and changes the retention period of objects to days
    
    5) ossutil worm get oss://bucket
       This command get worm configuration
    
`,
	sampleText: ` 
`,
}

type wormOptionType struct {
	bucketName string
	wormConfig oss.WormConfiguration
}

type WormCommand struct {
	command  Command
	wmOption wormOptionType
}

var wormCommand = WormCommand{
	command: Command{
		name:        "worm",
		nameAlias:   []string{"worm"},
		minArgc:     2,
		maxArgc:     4,
		specChinese: specChineseWorm,
		specEnglish: specEnglishWorm,
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
func (wormc *WormCommand) formatHelpForWhole() string {
	return wormc.command.formatHelpForWhole()
}

func (wormc *WormCommand) formatIndependHelp() string {
	return wormc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (wormc *WormCommand) Init(args []string, options OptionMapType) error {
	return wormc.command.Init(args, options, wormc)
}

// RunCommand simulate inheritance, and polymorphism
func (wormc *WormCommand) RunCommand() error {
	// init all command name
	commandDict := make(map[string]string)
	commandDict["init"] = "init"
	commandDict["abort"] = "abort"
	commandDict["complete"] = "complete"
	commandDict["extend"] = "extend"
	commandDict["get"] = "get"

	// check command name
	strCommand := wormc.command.args[0]
	_, ok := commandDict[strCommand]
	if !ok {
		return fmt.Errorf("invalid parameter %s,which must be init, abort, complete, extend, get", strCommand)
	}

	bucketUrL, err := StorageURLFromString(wormc.command.args[1], "")
	if err != nil {
		return err
	}

	if !bucketUrL.IsCloudURL() {
		return fmt.Errorf("parameter is not a cloud url,url is %s", bucketUrL.ToString())
	}

	cloudUrl := bucketUrL.(CloudURL)
	if cloudUrl.bucket == "" {
		return fmt.Errorf("bucket name is empty,url is %s", bucketUrL.ToString())
	}

	wormc.wmOption.bucketName = cloudUrl.bucket

	if strCommand == "init" {
		err = wormc.InitiateBucketWorm()
	} else if strCommand == "abort" {
		err = wormc.AbortBucketWorm()
	} else if strCommand == "complete" {
		err = wormc.CompleteBucketWorm()
	} else if strCommand == "extend" {
		err = wormc.ExtendBucketWorm()
	} else if strCommand == "get" {
		err = wormc.GetBucketWorm()
	}
	return err
}

func (wormc *WormCommand) InitiateBucketWorm() error {
	if len(wormc.command.args) < 3 {
		return fmt.Errorf("missing parameter,the parameter day is empty")
	}

	retentionDays, err := strconv.Atoi(wormc.command.args[2])
	if err != nil {
		return err
	}

	client, err := wormc.command.ossClient(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}

	wormID, err := client.InitiateBucketWorm(wormc.wmOption.bucketName, retentionDays)
	if wormID != "" {
		fmt.Printf("init success,worm id is %s", wormID)
	}
	return err
}

func (wormc *WormCommand) AbortBucketWorm() error {
	client, err := wormc.command.ossClient(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}
	return client.AbortBucketWorm(wormc.wmOption.bucketName)
}

func (wormc *WormCommand) CompleteBucketWorm() error {
	if len(wormc.command.args) < 3 {
		return fmt.Errorf("missing parameter,the wormId is empty")
	}

	client, err := wormc.command.ossClient(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}
	return client.CompleteBucketWorm(wormc.wmOption.bucketName, wormc.command.args[2])
}

func (wormc *WormCommand) ExtendBucketWorm() error {
	if len(wormc.command.args) < 4 {
		return fmt.Errorf("missing parameter, need 4 parameters")
	}

	retentionDays, err := strconv.Atoi(wormc.command.args[2])
	if err != nil {
		return err
	}

	client, err := wormc.command.ossClient(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}
	return client.ExtendBucketWorm(wormc.wmOption.bucketName, retentionDays, wormc.command.args[3])
}

func (wormc *WormCommand) GetBucketWorm() error {
	client, err := wormc.command.ossClient(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}

	wormConfig, err := client.GetBucketWorm(wormc.wmOption.bucketName)
	if err != nil {
		return err
	}

	wormc.wmOption.wormConfig = wormConfig
	output, err := xml.MarshalIndent(wormConfig, "  ", "    ")
	if err == nil {
		fmt.Println(string(output))
	}
	return err
}
