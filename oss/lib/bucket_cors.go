package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseCors = SpecText{
	synopsisText: "设置、查询或者删除bucket的cors配置",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil cors --method put oss://bucket  local_xml_file
    ossutil cors --method get oss://bucket  [local_xml_file]
    ossutil cors --method delete oss://bucket
`,
	detailHelpText: ` 
    cors命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的cors配置

用法:
    该命令有三种用法:
	
    1) ossutil cors --method put oss://bucket  local_xml_file
        这个命令从配置文件local_xml_file中读取cors配置，然后设置bucket的cors规则
        配置文件是一个xml格式的文件，举例如下
	   
        <?xml version="1.0" encoding="UTF-8"?>
          <CORSConfiguration>
            <CORSRule>
                <AllowedOrigin>www.aliyun.com</AllowedOrigin>
                <AllowedMethod>PUT</AllowedMethod>
                <MaxAgeSeconds>10000</MaxAgeSeconds>
            </CORSRule>
        </CORSConfiguration>
	
    2) ossutil cors --method get oss://bucket  [local_xml_file]
        这个命令查询bucket的cors配置
        如果输入参数local_xml_file，cors配置将输出到该文件，否则输出到屏幕上
	
    3)  ossutil cors --method delete oss://bucket
        这个命令删除bucket的cors配置
`,
	sampleText: ` 
    1) 设置bucket的cors配置
       ossutil cors --method put oss://bucket  local_xml_file

    2) 查询bucket的cors配置，结果输出到标准输出
       ossutil cors --method get oss://bucket
	
    3) 查询bucket的cors配置，结果输出到本地文件
       ossutil cors --method get oss://bucket  local_xml_file
	
    4) 删除bucket的cors配置
       ossutil cors --method delete oss://bucket
`,
}

var specEnglishCors = SpecText{
	synopsisText: "Set, get or delete the cors configuration of the oss bucket",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil cors --method put oss://bucket  local_xml_file
    ossutil cors --method get oss://bucket  [local_xml_file]
    ossutil cors --method delete oss://bucket
`,
	detailHelpText: ` 
    cors command can set、get and delete the cors configuration of the oss bucket by
    set method option value to put, get,delete

Usage:
    There are three usages for this command:
	
    1) ossutil cors --method put oss://bucket  local_xml_file
	   
        The command sets the cors configuration of bucket from local file local_xml_file
    the local_xml_file is xml format
        The following is an example of the contents of local_xml_file
	   
        <?xml version="1.0" encoding="UTF-8"?>
          <CORSConfiguration>
            <CORSRule>
                <AllowedOrigin>www.aliyun.com</AllowedOrigin>
                <AllowedMethod>PUT</AllowedMethod>
                <MaxAgeSeconds>10000</MaxAgeSeconds>
            </CORSRule>
        </CORSConfiguration>
	
    2) ossutil cors --method get oss://bucket  [local_xml_file]
        The command gets the cors configuration of bucket
        if you input parameter local_xml_file,the configuration will be output to local_xml_file
        if you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3)  ossutil cors --method delete oss://bucket
        The command deletes the cors configuration of bucket
`,
	sampleText: ` 
    1) put cors configuration
       ossutil cors --method put oss://bucket  local_xml_file

    2) get cors configuration to stdout
       ossutil cors --method get oss://bucket
	
    3) get cors configuration to local file
       ossutil cors --method get oss://bucket  local_xml_file
	
    4) delete cors configuration
       ossutil cors --method delete oss://bucket
`,
}

type corsOptionType struct {
	bucketName string
}

type CorsCommand struct {
	command  Command
	csOption corsOptionType
}

var corsCommand = CorsCommand{
	command: Command{
		name:        "cors",
		nameAlias:   []string{"cors"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseCors,
		specEnglish: specEnglishCors,
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
func (corsc *CorsCommand) formatHelpForWhole() string {
	return corsc.command.formatHelpForWhole()
}

func (corsc *CorsCommand) formatIndependHelp() string {
	return corsc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (corsc *CorsCommand) Init(args []string, options OptionMapType) error {
	return corsc.command.Init(args, options, corsc)
}

// RunCommand simulate inheritance, and polymorphism
func (corsc *CorsCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, corsc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}
	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	bucketUrL, err := StorageURLFromString(corsc.command.args[0], "")
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

	corsc.csOption.bucketName = cloudUrl.bucket

	if strMethod == "put" {
		err = corsc.PutBucketCors()
	} else if strMethod == "get" {
		err = corsc.GetBucketCors()
	} else if strMethod == "delete" {
		err = corsc.DeleteBucketCors()
	}
	if err != nil {
		fmt.Printf("error:%s\n", err.Error())
	}
	return err
}

func (corsc *CorsCommand) PutBucketCors() error {
	if len(corsc.command.args) < 2 {
		return fmt.Errorf("missing parameters,the local cors config file is empty")
	}

	corsFile := corsc.command.args[1]
	fileInfo, err := os.Stat(corsFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", corsFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", corsFile)
	}

	// parsing the xml file
	file, err := os.Open(corsFile)
	if err != nil {
		return err
	}
	defer file.Close()
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	rulesConfig := oss.CORSXML{}
	err = xml.Unmarshal(text, &rulesConfig)
	if err != nil {
		return err
	}

	// put bucket cors
	client, err := corsc.command.ossClient(corsc.csOption.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketCORS(corsc.csOption.bucketName, rulesConfig.CORSRules)
}

func (corsc *CorsCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("cors: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (corsc *CorsCommand) GetBucketCors() error {
	client, err := corsc.command.ossClient(corsc.csOption.bucketName)
	if err != nil {
		return err
	}

	corsRes, err := client.GetBucketCORS(corsc.csOption.bucketName)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(corsRes, "  ", "    ")
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(corsc.command.args) >= 2 {
		fileName := corsc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := corsc.confirm(fileName)
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

func (corsc *CorsCommand) DeleteBucketCors() error {
	client, err := corsc.command.ossClient(corsc.csOption.bucketName)
	if err != nil {
		return err
	}
	return client.DeleteBucketCORS(corsc.csOption.bucketName)
}
