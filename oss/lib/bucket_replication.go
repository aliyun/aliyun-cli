package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var specChineseReplication = SpecText{
	synopsisText: "bucket的跨区域同步配置管理",

	paramText: "bucket_url [local_xml_file|ruleID] [options]",

	syntaxText: ` 
    ossutil replication --method put oss://bucket local_xml_file [options]
    ossutil replication --method get oss://bucket [options]
    ossutil replication --method delete oss://bucket ruleID [options]
    ossutil replication --method get --item location oss://bucket [options]
    ossutil replication --method get --item progress oss://bucket [ruleID] [options]
    ossutil replication --method put --item rtc oss://bucket local_xml_file [options]
`,
	detailHelpText: `
    replication命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的跨区域复制规则;
    此外,当method选项为get时,可通过设置item选项选项值为location、progress,可以查询可复制到的目标bucket
    所在的地域或者bucket的跨区域复制进度信息

用法:
    该命令有六种用法:

    1) ossutil replication --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取跨区域复制的配置,然后设置bucket的跨区域复制规则,
        配置文件是一个xml格式的文件,举例如下
   
        <?xml version="1.0" encoding="UTF-8"?>
        <ReplicationConfiguration>
           <Rule>     
                <PrefixSet>
                    <Prefix>prefix_1</Prefix>
                    <Prefix>prefix_2</Prefix>
                </PrefixSet>
                <Action>ALL,PUT</Action>
                <Destination>
                    <Bucket>dest-bucket-name</Bucket>
                    <Location>oss-cn-hangzhou</Location>
                    <TransferType>oss_acc</TransferType>
                </Destination>
                <HistoricalObjectReplication>enabled</HistoricalObjectReplication>
           </Rule>
        </ReplicationConfiguration>

    2) ossutil replication --method get oss://bucket [options]
        这个命令查询bucket的跨区域复制规则,目前只支持输出到屏幕上

    3) ossutil replication --method delete oss://bucket ruleId [options]
        这个命令停止bucket的跨区域复制并删除bucket的复制配置
	
    4) ossutil replication --method get --item location oss://bucket [options]
        这个命令查询可复制到的目标bucket所在的地域,目前只支持输出到屏幕上
	
    5) ossutil replication --method get --item progress oss://bucket [ruleID] [options]
        这个命令查询bucket的跨区域复制进度,如果输入参数ruleID,则查询该ID对应的跨区域复制进度,否则查询bucket所有的跨区域复制进度,目前只支持输出到屏幕上
    
    6) ossutil replication --method put --item rtc oss://bucket [local_xml_file] [options]
        这个命令为bucket已有的跨区域复制规则开启或关闭数据复制时间控制
         配置文件是一个xml格式的文件,举例如下
           
        <?xml version="1.0" encoding="UTF-8"?>
        <ReplicationRule>
            <RTC>
                <Status>enabled or disabled</Status>
            </RTC>
            <ID>rule id</ID>
        </ReplicationRule>
`,

	sampleText: `
    1) 设置bucket的跨区域复制规则
       ossutil replication --method put oss://bucket local_xml_file

    2) 查询bucket的跨区域复制规则，结果输出到标准输出
       ossutil replication --method get oss://bucket
	
    3) 删除bucket的跨区域复制规则
       ossutil replication --method delete oss://bucket ruleID
	
    4) 查询可复制到的目标bucket所在的地域,结果输出到标准输出
       ossutil replication --method get --item location oss://bucket

    5) 查询bucket所有的跨区域复制进度,结果输出到标准输出
       ossutil replication --method get --item progress oss://bucket

    6) 查询bucket在指定ruleID下的跨区域复制进度,结果输出到标准输出
       ossutil replication --method get --item progress oss://bucket ruleID

    7) 为已有bucket的跨区域复制规则开启或关闭数据复制时间控制
       ossutil replication --method put --item rtc oss://bucket local_xml_file
`,
}

var specEnglishreplication = SpecText{
	synopsisText: "manage bucket's replication configuration",

	paramText: "bucket_url [local_xml_file|ruleID] [options]",

	syntaxText: ` 
    ossutil replication --method put oss://bucket local_xml_file [options]
    ossutil replication --method get oss://bucket [options]
    ossutil replication --method delete oss://bucket ruleID [options]
    ossutil replication --method get --item location oss://bucket [options]
    ossutil replication --method get --item progress oss://bucket [ruleID] [options]
    ossutil replication --method put --item rtc oss://bucket local_xml_file [options]
`,
	detailHelpText: ` 
    replication command can set, get and delete cross region replication rules of 
    the oss bucket by setting method option value to put, get and delete; in addition, 
    when the method option is get, you can get the region where the target bucket can 
    be copied to or the cross region replication progress of the bucket by setting item 
    option value to location and progress

Usage:
    There are six usages for this command:
	
    1) ossutil replication --method put oss://bucket local_xml_file [options]
        The command sets the cross region replication rules of bucket from local file local_xml_file
        The local_xml_file is xml format,for example

        <?xml version="1.0" encoding="UTF-8"?>
        <ReplicationConfiguration>
            <Rule>     
                <PrefixSet>
                    <Prefix>prefix_1</Prefix>
                    <Prefix>prefix_2</Prefix>
                </PrefixSet>
                <Action>ALL,PUT</Action>
                <Destination>
                    <Bucket>dest-bucket-name</Bucket>
                    <Location>oss-cn-hangzhou</Location>
                    <TransferType>oss_acc</TransferType>
                </Destination>
                <HistoricalObjectReplication>enabled</HistoricalObjectReplication>
            </Rule>
        </ReplicationConfiguration>
        
    2) ossutil replication --method get oss://bucket [options]
       The command gets the cross region replication rules of bucket
       At present, it only supports output to stdout
	
    3) ossutil replication --method delete oss://bucket ruleID
       The command stops the cross region replication of bucket and
	   removes the replication configuration of bucket

    4) ossutil replication --method get --item location oss://bucket [options]
       The command gets the regions of the target bucket that can be copied to
       At present, it only supports output to stdout

    5) ossutil replication --method get --item progress oss://bucket [ruleID] [options]
       The command gets the cross region replication progress of bucket
       If you input the parameter ruleID, the cross region replication progress corresponding to the ID will be gotten
       If you don't input the parameter ruleID, all cross region replication progress will be gotten
       At present, it only supports output to stdout

    6) ossutil replication --method put --item rtc oss://bucket [local_xml_file] [options]
        This command enables or disables data replication time control for the existing cross-region replication rules of the bucket
        The local_xml_file is xml format,for example
           
        <?xml version="1.0" encoding="UTF-8"?>
        <ReplicationRule>
            <RTC>
                <Status>enabled or disabled</Status>
            </RTC>
            <ID>rule id</ID>
        </ReplicationRule>
`,
	sampleText: ` 
    1) put bucket cross region replication rules
       ossutil replication --method put oss://bucket local_xml_file

    2) get bucket cross region replication rules, output to stdout
       ossutil replication --method get oss://bucket
	
    3) delete bucket cross region replication rules
       ossutil replication --method delete oss://bucket ruleID
	
    4) get regions where the target bucket can be copied to, output to stdout
       ossutil replication --method get --item location oss://bucket

    5) get bucket all the cross region replication progress, output to stdout
       ossutil replication --method get --item progress oss://bucket

    6) get bucket the cross region replication progress with the specified ruleID, output to stdout
       ossutil replication --method get --item progress oss://bucket ruleID

    7) enable or disable data replication time control for the existing cross region replication rule
       ossutil replication --method get --item rtc oss://bucket local_xml_file
`,
}

type ReplicationCommand struct {
	command    Command
	bucketName string
}

var replicationCommand = ReplicationCommand{
	command: Command{
		name:        "replication",
		nameAlias:   []string{"replication"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseReplication,
		specEnglish: specEnglishreplication,
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
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionReadTimeout,
			OptionConnectTimeout,
			OptionSTSRegion,
			OptionMethod,
			OptionItem,
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
func (replicationc *ReplicationCommand) formatHelpForWhole() string {
	return replicationc.command.formatHelpForWhole()
}

func (replicationc *ReplicationCommand) formatIndependHelp() string {
	return replicationc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (replicationc *ReplicationCommand) Init(args []string, options OptionMapType) error {
	return replicationc.command.Init(args, options, replicationc)
}

// RunCommand simulate inheritance, and polymorphism
func (replicationc *ReplicationCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, replicationc.command.options)
	strItem, _ := GetString(OptionItem, replicationc.command.options)

	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	strItem = strings.ToLower(strItem)
	if strMethod == "get" && strItem != "" && strItem != "location" && strItem != "progress" {
		return fmt.Errorf("--item value is not in the optional value:location|progress")
	}

	srcBucketUrL, err := GetCloudUrl(replicationc.command.args[0], "")
	if err != nil {
		return err
	}

	replicationc.bucketName = srcBucketUrL.bucket
	switch strMethod {
	case "put":
		switch strItem {
		case "":
			err = replicationc.PutBucketReplication()
		case "rtc":
			err = replicationc.PutBucketRTC()
		}
	case "get":
		switch strItem {
		case "":
			err = replicationc.GetBucketReplication()
		case "location":
			err = replicationc.GetBucketReplicationLocation()
		case "progress":
			err = replicationc.GetBucketReplicationProgress()
		}
	case "delete":
		err = replicationc.DeleteBucketReplication()
	}
	return err
}

func (replicationc *ReplicationCommand) PutBucketReplication() error {
	if len(replicationc.command.args) < 2 {
		return fmt.Errorf("put bucket replication need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := replicationc.command.args[1]
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

	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}
	return client.PutBucketReplication(replicationc.bucketName, string(text))
}

func (replicationc *ReplicationCommand) GetBucketReplication() error {
	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}
	data, err := client.GetBucketReplication(replicationc.bucketName)
	if err == nil {
		fmt.Printf("%s\n", data)
	}
	return err
}

func (replicationc *ReplicationCommand) DeleteBucketReplication() error {
	if len(replicationc.command.args) < 2 {
		return fmt.Errorf("delete bucket replication need at least 2 parameters,the rule ID is empty")
	}

	ruleID := replicationc.command.args[1]
	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}
	return client.DeleteBucketReplication(replicationc.bucketName, ruleID)
}

func (replicationc *ReplicationCommand) GetBucketReplicationLocation() error {
	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}

	data, err := client.GetBucketReplicationLocation(replicationc.bucketName)
	if err == nil {
		fmt.Printf("%s\n", data)
	}
	return err
}

func (replicationc *ReplicationCommand) GetBucketReplicationProgress() error {
	ruleID := ""

	if len(replicationc.command.args) >= 2 {
		ruleID = replicationc.command.args[1]
	}

	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}

	data, err := client.GetBucketReplicationProgress(replicationc.bucketName, ruleID)
	if err == nil {
		fmt.Printf("%s\n", data)
	}
	return err
}

func (replicationc *ReplicationCommand) PutBucketRTC() error {
	if len(replicationc.command.args) < 2 {
		return fmt.Errorf("put bucket rtc need at least 2 parameters,the local xml file is empty")
	}
	xmlFile := replicationc.command.args[1]
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

	client, err := replicationc.command.ossClient(replicationc.bucketName)
	if err != nil {
		return err
	}

	return client.PutBucketRTCXml(replicationc.bucketName, string(text))
}
