package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketResourceGroup = SpecText{
	synopsisText: "设置、查询bucket的resource group配置",
	paramText:    "bucket_url local_xml_file [options]",

	syntaxText: ` 
	ossutil resource-group --method put oss://bucket local_xml_file [options]
	ossutil resource-group --method get oss://bucket [local_xml_file] [options]
`,
	detailHelpText: ` 
    resource-group命令通过设置method选项值为put、get,可以设置、查询bucket的resource group配置

用法:
    该命令有二种用法:
	
    1) ossutil resource-group --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取resource group配置，然后设置bucket的resource group规则
        配置文件是一个xml格式的文件，可以选择只配置部分规则,下面是一个所有规则的例子
   
        <?xml version="1.0" encoding="UTF-8"?>
        <BucketResourceGroupConfiguration>
            <ResourceGroupId>rg-xxxxxx</ResourceGroupId>
        </BucketResourceGroupConfiguration>

    2) ossutil resource-group --method get oss://bucket [local_xml_file] [options]
        这个命令查询bucket的resource group配置
        如果输入参数local_xml_file，resource group配置将输出到该文件，否则输出到屏幕上

`,
	sampleText: ` 
    1) 设置bucket的resource group配置
       ossutil resource-group --method put oss://bucket local_xml_file

    2) 查询bucket的resource group配置，结果输出到标准输出
       ossutil resource-group --method get oss://bucket
	
    3) 查询bucket的resource group配置，结果输出到本地文件
       ossutil resource-group --method get oss://bucket local_xml_file
`,
}

var specEnglishBucketResourceGroup = SpecText{
	synopsisText: "Set, get bucket resource group configuration",
	paramText:    "bucket_url local_xml_file [options]",

	syntaxText: ` 
	ossutil resource-group --method put oss://bucket local_xml_file [options]
	ossutil resource-group --method get oss://bucket [local_xml_file] [options]
`,

	detailHelpText: ` 
    resource-group command can set, get the resource group configuration of the oss bucket by
    set method option value to put, get

Usage:
    1) ossutil resource-group --method put oss://bucket local_xml_file [options]
	   The command sets the resource group configuration of bucket from local file local_xml_file
        the local_xml_file is xml format, you can choose to configure only some rules
        The following is an example of all rules:

        <?xml version="1.0" encoding="UTF-8"?>
        <BucketResourceGroupConfiguration>
            <ResourceGroupId>rg-xxxxxx</ResourceGroupId>
        </BucketResourceGroupConfiguration>
	
	2) ossutil resource-group --method get oss://bucket [local_xml_file] [options]
	   The command gets the resource group configuration of bucket
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
`,

	sampleText: ` 
    1) put bucket resource group
       ossutil resource-group --method put oss://bucket local_xml_file

    2) get bucket resource group configuration to stdout
       ossutil resource-group --method get oss://bucket
	
    3) get bucket resource group configuration to local file
       ossutil resource-group --method get oss://bucket local_xml_file
`,
}

type bucketResourceGroupOptionType struct {
	bucketName string
}

type BucketResourceGroupCommand struct {
	command  Command
	blOption bucketResourceGroupOptionType
}

var bucketResourceGroupCommand = BucketResourceGroupCommand{
	command: Command{
		name:        "resource-group",
		nameAlias:   []string{"resource-group"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketResourceGroup,
		specEnglish: specEnglishBucketResourceGroup,
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
func (brgc *BucketResourceGroupCommand) formatHelpForWhole() string {
	return brgc.command.formatHelpForWhole()
}

func (brgc *BucketResourceGroupCommand) formatIndependHelp() string {
	return brgc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (brgc *BucketResourceGroupCommand) Init(args []string, options OptionMapType) error {
	return brgc.command.Init(args, options, brgc)
}

// RunCommand simulate inheritance, and polymorphism
func (brgc *BucketResourceGroupCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, brgc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(brgc.command.args[0], "")
	if err != nil {
		return err
	}

	brgc.blOption.bucketName = srcBucketUrL.bucket

	switch strMethod {
	case "put":
		err = brgc.PutBucketResourceGroup()
	case "get":
		err = brgc.GetBucketResourceGroup()
	}
	return err
}

func (brgc *BucketResourceGroupCommand) PutBucketResourceGroup() error {
	if len(brgc.command.args) < 2 {
		return fmt.Errorf("put bucket resource group need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := brgc.command.args[1]
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
	client, err := brgc.command.ossClient(brgc.blOption.bucketName)
	if err != nil {
		return err
	}

	options := []oss.Option{oss.AllowSameActionOverLap(true)}
	return client.PutBucketResourceGroupXml(brgc.blOption.bucketName, string(xmlBody), options...)
}

func (brgc *BucketResourceGroupCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket resource group: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (brgc *BucketResourceGroupCommand) GetBucketResourceGroup() error {
	client, err := brgc.command.ossClient(brgc.blOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketResourceGroupXml(brgc.blOption.bucketName)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(brgc.command.args) >= 2 {
		fileName := brgc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bContinue := brgc.confirm(fileName)
			if !bContinue {
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
