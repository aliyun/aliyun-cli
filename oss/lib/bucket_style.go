package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketStyle = SpecText{
	synopsisText: "添加、查询、删除或者列举bucket的图片样式",

	paramText: "bucket_url [local_xml_file] [style_name] [options]",

	syntaxText: ` 
    ossutil style --method put oss://bucket style_name local_xml_file [options]
    ossutil style --method get oss://bucket style_name [local_file] [options]
    ossutil style --method delete oss://bucket style_name [options]
    ossutil style --method list oss://bucket [local_file] [options]
`,
	detailHelpText: ` 
    style命令通过设置method选项值为put、get、delete、list,可以添加、查询、删除、列举bucket的图片样式

用法:
    该命令有四种用法:
	
    1) ossutil style --method put oss://bucket style_name local_xml_file [options]
        这个命令从配置文件local_xml_file中读取图片样式,然后添加一个bucket样式名城为style_name的图片样式
        配置文件是一个xml格式的文件,如果已经存在样式名称为style_name的配置,则覆盖
        下面是一个配置文件例子
   
        <?xml version="1.0" encoding="UTF-8"?>
        <Style>
         <Content>image/resize,p_50</Content>
        </Style>
      
    2) ossutil style --method get oss://bucket style_name [local_xml_file] [options]
        这个命令查询bucket样式名称为style_name的图片样式
        如果输入参数local_xml_file，图片样式将输出到该文件，否则输出到屏幕上
	
    3) ossutil style --method delete oss://bucket style_name [options]
        这个命令删除bucket样式名称为style_name的图片样式
    
    4) ossutil style --method list oss://bucket [local_file] [options]
        这个命令列举bucket的图片样式
`,
	sampleText: ` 
    1) 添加bucket的style图片样式
       ossutil style --method put oss://bucket style_name local_xml_file

    2) 查询bucket样式名称为style_name的图片样式，结果输出到标准输出
       ossutil style --method get oss://bucket style_name
	
    3) 删除bucket样式名称为style_name的图片样式
       ossutil style --method delete oss://bucket style_name
    
    4) 列举bucket的所有图片样式
       ossutil style --method list oss://bucket
`,
}

var specEnglishBucketStyle = SpecText{
	synopsisText: "Add, get, delete, or list bucket style configuration",

	paramText: "bucket_url [local_xml_file] [style_name] [options]",

	syntaxText: ` 
    ossutil style --method put oss://bucket style_name local_xml_file [options]
    ossutil style --method get oss://bucket style_name [local_file] [options]
    ossutil style --method delete oss://bucket style_name [options]
    ossutil style --method list oss://bucket [local_file] [options]
`,
	detailHelpText: ` 
    style command can add, get, delete or list the style configuration of the oss bucket by
    set method option value to put, get, delete, list

Usage:
    There are four usages for this command:
	
    1) ossutil style --method put oss://bucket style_name local_xml_file [options]
        The command adds the style of bucket name as style_name from local file local_xml_file
        the local_xml_file is xml format, if there is exist same style name of configuration, then overwrite.
        The following is an example xml file

        <?xml version="1.0" encoding="UTF-8"?>
        <Style>
         <Content>image/resize,p_50</Content>
        </Style>
        
    2) ossutil style --method get oss://bucket style_name [local_xml_file] [options]
       The command gets the style of bucket, The identifier of the style is style name
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3) ossutil style --method delete oss://bucket style_name [options]
       The command deletes the style configuration of bucket, The identifier of the style is style name
      
    4) ossutil style --method list oss://bucket [local_file] [options]
       List the bucket's all style configuration
`,
	sampleText: ` 
    1) add bucket style
       ossutil style --method put oss://bucket local_xml_file

    2) get style configuration to stdout, The identifier of the style is style name
       ossutil style --method get oss://bucket style_name
	
    3) delete style configuration, The identifier of the style is style name
       ossutil style --method delete oss://bucket style_name
    
    4) list the bucket's all style configuration
       ossutil style --method list oss://bucket
`,
}

type BucketStyleOptionType struct {
	bucketName string
	ruleCount  int
}

type BucketStyleCommand struct {
	command  Command
	bwOption BucketStyleOptionType
}

var bucketStyleCommand = BucketStyleCommand{
	command: Command{
		name:        "style",
		nameAlias:   []string{"style"},
		minArgc:     1,
		maxArgc:     3,
		specChinese: specChineseBucketStyle,
		specEnglish: specEnglishBucketStyle,
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
func (bsc *BucketStyleCommand) formatHelpForWhole() string {
	return bsc.command.formatHelpForWhole()
}

func (bsc *BucketStyleCommand) formatIndependHelp() string {
	return bsc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bsc *BucketStyleCommand) Init(args []string, options OptionMapType) error {
	return bsc.command.Init(args, options, bsc)
}

// RunCommand simulate inheritance, and polymorphism
func (bsc *BucketStyleCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bsc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" && strMethod != "list" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete|list")
	}

	srcBucketUrL, err := GetCloudUrl(bsc.command.args[0], "")
	if err != nil {
		return err
	}

	bsc.bwOption.bucketName = srcBucketUrL.bucket

	switch strMethod {
	case "put":
		err = bsc.PutBucketStyle()
	case "get":
		err = bsc.GetBucketStyle()
	case "list":
		err = bsc.ListBucketStyle()
	case "delete":
		err = bsc.DeleteBucketStyle()
	}
	return err
}

func (bsc *BucketStyleCommand) PutBucketStyle() error {
	if len(bsc.command.args) < 3 {
		return fmt.Errorf("put bucket style need at least 3 parameters,the parameter style name is empty or the local xml file is empty")
	}

	styleName := bsc.command.args[1]
	xmlFile := bsc.command.args[2]
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
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// put bucket style
	client, err := bsc.command.ossClient(bsc.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.PutBucketStyleXml(bsc.bwOption.bucketName, styleName, string(text))
}

func (bsc *BucketStyleCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket style: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bsc *BucketStyleCommand) GetBucketStyle() error {
	if len(bsc.command.args) < 2 {
		return fmt.Errorf("get bucket style need at least 2 parameters,the parameter style name is empty")
	}

	styleName := bsc.command.args[1]

	client, err := bsc.command.ossClient(bsc.bwOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketStyleXml(bsc.bwOption.bucketName, styleName)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bsc.command.args) >= 3 {
		fileName := bsc.command.args[2]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bsc.confirm(fileName)
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

func (bsc *BucketStyleCommand) DeleteBucketStyle() error {
	if len(bsc.command.args) < 2 {
		return fmt.Errorf("delete bucket style need at least 2 parameters,the parameter style name is empty")
	}

	styleName := bsc.command.args[1]

	// delete bucket style
	client, err := bsc.command.ossClient(bsc.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.DeleteBucketStyle(bsc.bwOption.bucketName, styleName)
}

func (bsc *BucketStyleCommand) ListBucketStyle() error {
	bsc.bwOption.ruleCount = 0
	client, err := bsc.command.ossClient(bsc.bwOption.bucketName)
	if err != nil {
		return err
	}

	xmlBody, err := client.ListBucketStyleXml(bsc.bwOption.bucketName)
	if err != nil {
		return err
	}

	var result oss.GetBucketListStyleResult
	err = xml.Unmarshal([]byte(xmlBody), &result)
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bsc.command.args) >= 2 {
		fileName := bsc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bsc.confirm(fileName)
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

	outFile.Write([]byte(xmlBody))
	return nil
}
