package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketAccessMonitor = SpecText{
	synopsisText: "设置、查询bucket的access monitor配置",
	paramText:    "bucket_url local_xml_file [options]",

	syntaxText: ` 
	ossutil access-monitor --method put oss://bucket local_xml_file [options]
    ossutil access-monitor --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    access-monitor命令通过设置method选项值为put、get,可以设置、查询bucket的access monitor配置

用法:
    该命令有二种用法:
	
    1) ossutil access-monitor --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取access monitor配置，然后设置bucket的access monitor规则
        配置文件是一个xml格式的文件，可以选择只配置部分规则,下面是一个所有规则的例子
   
        <?xml version="1.0" encoding="UTF-8"?>
        <AccessMonitorConfiguration>
            <Status>Enabled</Status>
        </AccessMonitorConfiguration>

    2) ossutil access-monitor --method get oss://bucket [local_xml_file] [options]
        这个命令查询bucket的access monitor配置
        如果输入参数local_xml_file，access monitor配置将输出到该文件，否则输出到屏幕上

`,
	sampleText: ` 
    1) 设置bucket的access monitor配置
       ossutil access-monitor --method put oss://bucket local_xml_file

    2) 查询bucket的access monitor配置，结果输出到标准输出
       ossutil access-monitor --method get oss://bucket
	
    3) 查询bucket的access monitor配置，结果输出到本地文件
       ossutil access-monitor --method get oss://bucket local_xml_file
`,
}

var specEnglishBucketAccessMonitor = SpecText{
	synopsisText: "Set, get bucket access monitor configuration",
	paramText:    "bucket_url local_xml_file [options]",

	syntaxText: ` 
	ossutil access-monitor --method put oss://bucket local_xml_file [options]
    ossutil access-monitor --method get oss://bucket [local_xml_file] [options]
`,

	detailHelpText: ` 
    access-monitor command can set, get the access monitor configuration of the oss bucket by
    set method option value to put, get

Usage:
    1) ossutil access-monitor --method put oss://bucket local_xml_file [options]
	   The command sets the lifecycle configuration of bucket from local file local_xml_file
        the local_xml_file is xml format, you can choose to configure only some rules
        The following is an example of all rules:

        <?xml version="1.0" encoding="UTF-8"?>
        <AccessMonitorConfiguration>
            <Status>Enabled</Status>
        </AccessMonitorConfiguration>
	
	2) ossutil access-monitor --method get oss://bucket [local_xml_file] [options]
	   The command gets the access monitor configuration of bucket
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
`,

	sampleText: ` 
    1) put bucket access monitor
       ossutil access-monitor --method put oss://bucket local_xml_file

    2) get bucket access monitor configuration to stdout
       ossutil access-monitor --method get oss://bucket
	
    3) get bucket access monitor configuration to local file
       ossutil access-monitor --method get oss://bucket local_xml_file
`,
}

type bucketAccessMonitorOptionType struct {
	bucketName string
}

type BucketAccessMonitorCommand struct {
	command  Command
	blOption bucketAccessMonitorOptionType
}

var bucketAccessMonitorCommand = BucketAccessMonitorCommand{
	command: Command{
		name:        "access-monitor",
		nameAlias:   []string{"access-monitor"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketAccessMonitor,
		specEnglish: specEnglishBucketAccessMonitor,
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
func (blc *BucketAccessMonitorCommand) formatHelpForWhole() string {
	return blc.command.formatHelpForWhole()
}

func (blc *BucketAccessMonitorCommand) formatIndependHelp() string {
	return blc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (blc *BucketAccessMonitorCommand) Init(args []string, options OptionMapType) error {
	return blc.command.Init(args, options, blc)
}

// RunCommand simulate inheritance, and polymorphism
func (blc *BucketAccessMonitorCommand) RunCommand() error {
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

	blc.blOption.bucketName = srcBucketUrL.bucket

	switch strMethod {
	case "put":
		err = blc.PutBucketAccessMonitor()
	case "get":
		err = blc.GetBucketAccessMonitor()
	}
	return err
}

func (blc *BucketAccessMonitorCommand) PutBucketAccessMonitor() error {
	if len(blc.command.args) < 2 {
		return fmt.Errorf("put bucket access monitor need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := blc.command.args[1]
	fileInfo, err := os.Stat(xmlFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", xmlFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", xmlFile)
	}

	// parsing the xml file
	file, err := os.Open(xmlFile)
	if err != nil {
		return err
	}
	defer file.Close()
	xmlBody, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// put bucket lifecycle
	client, err := blc.command.ossClient(blc.blOption.bucketName)
	if err != nil {
		return err
	}

	options := []oss.Option{oss.AllowSameActionOverLap(true)}
	return client.PutBucketAccessMonitorXml(blc.blOption.bucketName, string(xmlBody), options...)
}

func (blc *BucketAccessMonitorCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket access monitor: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (blc *BucketAccessMonitorCommand) GetBucketAccessMonitor() error {
	client, err := blc.command.ossClient(blc.blOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketAccessMonitorXml(blc.blOption.bucketName)
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

	outFile.Write([]byte(output))

	fmt.Printf("\n\n")

	return nil
}
