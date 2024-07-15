package lib

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketQos = SpecText{
	synopsisText: "设置、查询或者删除bucket的qos配置",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
	ossutil bucket-qos --method put oss://bucket local_xml_file [options]
    ossutil bucket-qos --method get oss://bucket [local_file] [options]
    ossutil bucket-qos --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    bucket-qos命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的qos配置

用法:
    该命令有三种用法:
	
    1) ossutil bucket-qos --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取qos配置,然后设置bucket的qos规则
        配置文件是一个xml格式的文件,举例如下
   
        <?xml version="1.0" encoding="UTF-8"?>
        <QoSConfiguration>
          <TotalUploadBandwidth>10</TotalUploadBandwidth>
          <IntranetUploadBandwidth>-1</IntranetUploadBandwidth>
          <ExtranetUploadBandwidth>-1</ExtranetUploadBandwidth>
          <TotalDownloadBandwidth>10</TotalDownloadBandwidth>
          <IntranetDownloadBandwidth>-1</IntranetDownloadBandwidth>
          <ExtranetDownloadBandwidth>-1</ExtranetDownloadBandwidth>
          <TotalQps>1000</TotalQps>
          <IntranetQps>-1</IntranetQps>
          <ExtranetQps>-1</ExtranetQps>
        </QoSConfiguration>

    2) ossutil bucket-qos --method get oss://bucket [local_xml_file] [options]
        这个命令查询bucket的qos配置,如果输入参数local_xml_file,qos配置将输出到该文件,否则输出到屏幕上
	
    3) ossutil bucket-qos --method delete oss://bucket [options]
        这个命令删除bucket的qos配置
`,
	sampleText: ` 
    1) 设置bucket的qos配置
       ossutil bucket-qos --method put oss://bucket local_xml_file

    2) 查询bucket的qos配置，结果输出到标准输出
       ossutil bucket-qos --method get oss://bucket
	
    3) 查询bucket的qos配置，结果输出到本地文件
       ossutil bucket-qos --method get oss://bucket local_xml_file
	
    4) 删除bucket的qos配置
       ossutil bucket-qos --method delete oss://bucket
`,
}

var specEnglishBucketQos = SpecText{
	synopsisText: "Set, get or delete bucket qos configuration",

	paramText: "bucket_url [local_xml_file] [options]",

	syntaxText: ` 
	ossutil bucket-qos --method put oss://bucket local_xml_file [options]
    ossutil bucket-qos --method get oss://bucket [local_xml_file] [options]
    ossutil bucket-qos --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    bucket-qos command can set, get and delete the qos configuration of the oss bucket by
    set method option value to put, get, delete

Usage:
    There are three usages for this command:
	
    1) ossutil bucket-qos --method put oss://bucket local_xml_file [options]
        The command sets the qos configuration of bucket from local file local_xml_file
        the local_xml_file is xml format,for example

        <?xml version="1.0" encoding="UTF-8"?>
        <QoSConfiguration>
          <TotalUploadBandwidth>10</TotalUploadBandwidth>
          <IntranetUploadBandwidth>-1</IntranetUploadBandwidth>
          <ExtranetUploadBandwidth>-1</ExtranetUploadBandwidth>
          <TotalDownloadBandwidth>10</TotalDownloadBandwidth>
          <IntranetDownloadBandwidth>-1</IntranetDownloadBandwidth>
          <ExtranetDownloadBandwidth>-1</ExtranetDownloadBandwidth>
          <TotalQps>1000</TotalQps>
          <IntranetQps>-1</IntranetQps>
          <ExtranetQps>-1</ExtranetQps>
        </QoSConfiguration>
        
    2) ossutil bucket-qos --method get oss://bucket  [local_xml_file] [options]
       The command gets the qos configuration of bucket
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3) ossutil bucket-qos --method delete oss://bucket [options]
       The command deletes the qos configuration of bucket
`,
	sampleText: ` 
    1) put bucket qos
       ossutil bucket-qos --method put oss://bucket local_xml_file

    2) get bucket qos configuration to stdout
       ossutil bucket-qos --method get oss://bucket
	
    3) get bucket qos configuration to local file
       ossutil bucket-qos --method get oss://bucket local_xml_file
	
    4) delete bucket qos configuration
       ossutil bucket-qos --method delete oss://bucket
`,
}

type bucketQosOptionType struct {
	bucketName string
}

type BucketQosCommand struct {
	command  Command
	bqOption bucketQosOptionType
}

var bucketQosCommand = BucketQosCommand{
	command: Command{
		name:        "bucket-qos",
		nameAlias:   []string{"bucket-qos"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketQos,
		specEnglish: specEnglishBucketQos,
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
func (bqc *BucketQosCommand) formatHelpForWhole() string {
	return bqc.command.formatHelpForWhole()
}

func (bqc *BucketQosCommand) formatIndependHelp() string {
	return bqc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bqc *BucketQosCommand) Init(args []string, options OptionMapType) error {
	return bqc.command.Init(args, options, bqc)
}

// RunCommand simulate inheritance, and polymorphism
func (bqc *BucketQosCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bqc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(bqc.command.args[0], "")
	if err != nil {
		return err
	}

	bqc.bqOption.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bqc.PutBucketQos()
	} else if strMethod == "get" {
		err = bqc.GetBucketQos()
	} else if strMethod == "delete" {
		err = bqc.DeleteBucketQos()
	}
	return err
}

func (bqc *BucketQosCommand) PutBucketQos() error {
	if len(bqc.command.args) < 2 {
		return fmt.Errorf("put bucket qos need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := bqc.command.args[1]
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

	qosConfig := oss.BucketQoSConfiguration{}
	err = xml.Unmarshal(text, &qosConfig)
	if err != nil {
		return err
	}

	// put bucket qos
	client, err := bqc.command.ossClient(bqc.bqOption.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketQoSInfo(bqc.bqOption.bucketName, qosConfig)
}

func (bqc *BucketQosCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket qos: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bqc *BucketQosCommand) GetBucketQos() error {
	client, err := bqc.command.ossClient(bqc.bqOption.bucketName)
	if err != nil {
		return err
	}

	qosRes, err := client.GetBucketQosInfo(bqc.bqOption.bucketName)
	if err != nil {
		return err
	}

	output, err := xml.MarshalIndent(qosRes, "  ", "    ")
	if err != nil {
		return err
	}

	var outFile *os.File
	if len(bqc.command.args) >= 2 {
		fileName := bqc.command.args[1]
		_, err = os.Stat(fileName)
		if err == nil {
			bConitnue := bqc.confirm(fileName)
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

func (bqc *BucketQosCommand) DeleteBucketQos() error {
	// delete bucket qos
	client, err := bqc.command.ossClient(bqc.bqOption.bucketName)
	if err != nil {
		return err
	}

	return client.DeleteBucketQosInfo(bqc.bqOption.bucketName)
}
