package lib

import (
	"fmt"
	"os"
	"strings"
)

var specChineseBucketCname = SpecText{
	synopsisText: "管理bucket cname",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    cname命令通过设置method选项值为get可以查询bucket的cname配置

用法:
    该命令只有一种种用法:
    1) ossutil bucket-cname --method get oss://bucket  [local_xml_file] [options]
        这个命令查询bucket的cname配置
        如果输入参数local_xml_file，cname配置将输出到该文件，否则输出到屏幕上
`,
	sampleText: ` 
    1) 查询bucket的cname配置，结果输出到标准输出
       ossutil bucket-cname --method get oss://bucket
`,
}

var specEnglishBucketCname = SpecText{
	synopsisText: "get bucket cname configuration",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil bucket-cname --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    bucket-cname command can get the cname configuration of the oss bucket by
    set method option value to get

Usage:
    There are only one usage for this command:
    1) ossutil bucket-cname --method get oss://bucket  [local_xml_file] [options]
       The command gets the cname configuration of bucket
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
`,
	sampleText: ` 
    1) get cname configuration to stdout
       ossutil bucket-cname --method get oss://bucket
`,
}

type bucketCnameOptionType struct {
	bucketName string
}

type BucketCnameCommand struct {
	command  Command
	bwOption bucketCnameOptionType
}

var bucketCnameCommand = BucketCnameCommand{
	command: Command{
		name:        "bucket-cname",
		nameAlias:   []string{"bucket-cname"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketCname,
		specEnglish: specEnglishBucketCname,
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
			OptionSkipVerfiyCert,
			OptionUserAgent,
		},
	},
}

// function for FormatHelper interface
func (bwc *BucketCnameCommand) formatHelpForWhole() string {
	return bwc.command.formatHelpForWhole()
}

func (bwc *BucketCnameCommand) formatIndependHelp() string {
	return bwc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bwc *BucketCnameCommand) Init(args []string, options OptionMapType) error {
	return bwc.command.Init(args, options, bwc)
}

// RunCommand simulate inheritance, and polymorphism
func (bwc *BucketCnameCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bwc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "get" {
		return fmt.Errorf("--method only support get")
	}

	srcBucketUrL, err := GetCloudUrl(bwc.command.args[0], "")
	if err != nil {
		return err
	}

	bwc.bwOption.bucketName = srcBucketUrL.bucket
	return bwc.GetBucketCname()
}

func (bwc *BucketCnameCommand) GetBucketCname() error {
	client, err := bwc.command.ossClient(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketCname(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bwc.command.args) >= 2 {
		fileName := bwc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bwc.confirm(fileName)
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

	outFile.Write([]byte(output))

	fmt.Printf("\n\n")

	return nil
}

func (bwc *BucketCnameCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket cname: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}
