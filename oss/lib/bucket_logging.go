package lib

import (
	"encoding/xml"
	"fmt"
	"os"
	"strings"
)

var specChineseBucketLog = SpecText{
	synopsisText: "设置、查询或者删除bucket的log配置",

	paramText: "src_bucket_url target_bucket_url [options]",

	syntaxText: ` 
	ossutil logging --method put oss://bucket oss://target-bucket/[prefix]
    ossutil logging --method get oss://bucket [local_xml_file]
    ossutil logging --method delete oss://bucket
`,
	detailHelpText: ` 
    logging命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的log配置

用法:
    该命令有三种用法:
	
    1) ossutil logging --method put oss://bucket  oss://target-bucket/[prefix]
        这个命令将bucket的访问日志设置成输出到target-bucket中
        如果输入prefix，可以设置访问日志文件的前缀
	
    2) ossutil logging --method get oss://bucket  [local_xml_file]
        这个命令查询bucket的log配置
        如果输入参数local_xml_file，log配置将输出到该文件，否则输出到屏幕上
	
    3)  ossutil logging --method delete oss://bucket
        这个命令删除bucket的log配置
`,
	sampleText: ` 
    1) 设置bucket的日志配置
       ossutil logging --method put oss://bucket oss://target-bucket/

    2) 查询bucket的日志配置，结果输出到标准输出
       ossutil logging --method get oss://bucket
	
    3) 查询bucket的日志配置，结果输出到本地文件
       ossutil logging --method get oss://bucket local_xml_file
	
    4) 删除bucket的日志配置
       ossutil logging --method delete oss://bucket
`,
}

var specEnglishBucketLog = SpecText{
	synopsisText: "Set、get or delete bucket log configuration",

	paramText: "src_bucket_url target_bucket_url [options]",

	syntaxText: ` 
	ossutil logging --method put oss://bucket oss://target-bucket/[prefix]
    ossutil logging --method get oss://bucket [local_xml_file]
    ossutil logging --method delete oss://bucket
`,
	detailHelpText: ` 
    logging command can set、get and delete the log configuration of the oss bucket by
    set method option value to put, get,delete

Usage:
    There are three usages for this command::
	
    1) ossutil logging --method put oss://bucket  oss://target-bucket/[prefix]
        The command sets the log configuration of bucket to output to target-bucket
        If prefix is set, bucket log objects will have this prefix
	
    2) ossutil logging --method get oss://bucket  [local_xml_file]
        The command gets the log configuration of bucket
        if you input parameter local_xml_file,the configuration will be output to local_xml_file
        if you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3)  ossutil logging --method delete oss://bucket
        The command deletes the log configuration of bucket
`,
	sampleText: ` 
    1) put bucket log configuration  
       ossutil logging --method put oss://bucket  oss://target-bucket/

    2) get bucket log configuration to stdout
       ossutil logging --method get oss://bucket
	
    3) get bucket log configuration to local file
       ossutil logging --method get oss://bucket  local_xml_file
	
    4) delete bucket log configuration
       ossutil logging --method delete oss://bucket
`,
}

type bucketLogOptionType struct {
	srcBucketName  string
	destBucketName string
	destPrefix     string
}

type BucketLogCommand struct {
	command  Command
	blOption bucketLogOptionType
}

var bucketLogCommand = BucketLogCommand{
	command: Command{
		name:        "logging",
		nameAlias:   []string{"logging"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketLog,
		specEnglish: specEnglishBucketLog,
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
			OptionMethod,
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
func (blc *BucketLogCommand) formatHelpForWhole() string {
	return blc.command.formatHelpForWhole()
}

func (blc *BucketLogCommand) formatIndependHelp() string {
	return blc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (blc *BucketLogCommand) Init(args []string, options OptionMapType) error {
	return blc.command.Init(args, options, blc)
}

// RunCommand simulate inheritance, and polymorphism
func (blc *BucketLogCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, blc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(blc.command.args[0], "")
	if err != nil {
		return err
	}

	blc.blOption.srcBucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = blc.PutBucketLog()
	} else if strMethod == "get" {
		err = blc.GetBucketLog()
	} else if strMethod == "delete" {
		err = blc.DeleteBucketLog()
	}
	return err
}

func (blc *BucketLogCommand) PutBucketLog() error {
	if len(blc.command.args) < 2 {
		return fmt.Errorf("missing parameter,the target bucket is empty")
	}

	destBucketUrL, err := GetCloudUrl(blc.command.args[1], "")
	if err != nil {
		return err
	}

	blc.blOption.destBucketName = destBucketUrL.bucket
	blc.blOption.destPrefix = destBucketUrL.object

	// put bucket log
	client, err := blc.command.ossClient(blc.blOption.srcBucketName)
	if err != nil {
		return err
	}

	return client.SetBucketLogging(blc.blOption.srcBucketName, blc.blOption.destBucketName, blc.blOption.destPrefix, true)
}

func (blc *BucketLogCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket log: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (blc *BucketLogCommand) GetBucketLog() error {
	client, err := blc.command.ossClient(blc.blOption.srcBucketName)
	if err != nil {
		return err
	}

	logRes, err := client.GetBucketLogging(blc.blOption.srcBucketName)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(logRes, "  ", "    ")
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(blc.command.args) >= 2 {
		fileName := blc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := blc.confirm(fileName)
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

func (blc *BucketLogCommand) DeleteBucketLog() error {
	client, err := blc.command.ossClient(blc.blOption.srcBucketName)
	if err != nil {
		return err
	}
	return client.DeleteBucketLogging(blc.blOption.srcBucketName)
}
