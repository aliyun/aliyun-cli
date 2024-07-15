package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

var specChineseBucketWebSite = SpecText{
	synopsisText: "设置、查询或者删除bucket的website配置",

	paramText: "bucket_url local_xml_file [options]",

	syntaxText: ` 
	ossutil website --method put oss://bucket local_xml_file [options]
    ossutil website --method get oss://bucket [local_file] [options]
    ossutil website --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    website命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的website配置

用法:
    该命令有三种用法:
	
    1) ossutil website --method put oss://bucket local_xml_file [options]
        这个命令从配置文件local_xml_file中读取website配置,然后设置bucket的website规则
        配置文件是一个xml格式的文件,可以选择只配置部分规则,下面是一个所有规则的例子
   
        <?xml version="1.0" encoding="UTF-8"?>
        <WebsiteConfiguration>
            <IndexDocument>
                <Suffix>index.html</Suffix>
            </IndexDocument>
            <ErrorDocument>
                <Key>error.html</Key>
            </ErrorDocument>
            <RoutingRules>
                <RoutingRule>
                    <RuleNumber>1</RuleNumber>
                    <Condition>
                        <KeyPrefixEquals>abc/</KeyPrefixEquals>
                        <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
                    </Condition>
                    <Redirect>
                        <RedirectType>Mirror</RedirectType>
                        <PassQueryString>true</PassQueryString>
                        <MirrorURL>http://www.test.com/</MirrorURL>
                        <MirrorPassQueryString>true</MirrorPassQueryString>
                        <MirrorFollowRedirect>true</MirrorFollowRedirect>
                        <MirrorCheckMd5>false</MirrorCheckMd5>
                        <MirrorHeaders>
                          <PassAll>true</PassAll>
                          <Pass>myheader-key1</Pass>
                          <Pass>myheader-key2</Pass>
                          <Remove>myheader-key3</Remove>
                          <Remove>myheader-key4</Remove>
                          <Set>
                            <Key>myheader-key5</Key>
                            <Value>myheader-value5</Value>
                          </Set>
                        </MirrorHeaders>
                    </Redirect>
                </RoutingRule>
                <RoutingRule>
                    <RuleNumber>2</RuleNumber>
                    <Condition>
                      <KeyPrefixEquals>abc/</KeyPrefixEquals>
                      <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
                      <IncludeHeader>
                        <Key>host</Key>
                        <Equals>test.oss-cn-beijing-internal.aliyuncs.com</Equals>
                      </IncludeHeader>
                    </Condition>
                    <Redirect>
                      <RedirectType>AliCDN</RedirectType>
                      <Protocol>http</Protocol>
                      <HostName>www.test.com</HostName>
                      <PassQueryString>false</PassQueryString>
                      <ReplaceKeyWith>prefix/${key}.suffix</ReplaceKeyWith>
                      <HttpRedirectCode>301</HttpRedirectCode>
                    </Redirect>
                </RoutingRule>
            </RoutingRules>
        </WebsiteConfiguration>

    2) ossutil website --method get oss://bucket  [local_xml_file] [options]
        这个命令查询bucket的website配置
        如果输入参数local_xml_file，website配置将输出到该文件，否则输出到屏幕上
	
    3) ossutil website --method delete oss://bucket [options]
        这个命令删除bucket的website配置
`,
	sampleText: ` 
    1) 设置bucket的website配置
       ossutil website --method put oss://bucket local_xml_file

    2) 查询bucket的website配置，结果输出到标准输出
       ossutil website --method get oss://bucket
	
    3) 查询bucket的website配置，结果输出到本地文件
       ossutil website --method get oss://bucket local_xml_file
	
    4) 删除bucket的website配置
       ossutil website --method delete oss://bucket
`,
}

var specEnglishBucketWebSite = SpecText{
	synopsisText: "Set, get or delete bucket website configuration",

	paramText: "bucket_url website [options]",

	syntaxText: ` 
	ossutil website --method put oss://bucket local_xml_file [options]
    ossutil website --method get oss://bucket [local_xml_file] [options]
    ossutil website --method delete oss://bucket [options]
`,
	detailHelpText: ` 
    website command can set, get and delete the website configuration of the oss bucket by
    set method option value to put, get, delete

Usage:
    There are three usages for this command:
	
    1) ossutil website --method put oss://bucket local_xml_file [options]
        The command sets the website configuration of bucket from local file local_xml_file
        the local_xml_file is xml format,you can choose to configure only some rules
        The following is an example of all rules:

        <?xml version="1.0" encoding="UTF-8"?>
        <WebsiteConfiguration>
            <IndexDocument>
                <Suffix>index.html</Suffix>
            </IndexDocument>
            <ErrorDocument>
                <Key>error.html</Key>
            </ErrorDocument>
            <RoutingRules>
                <RoutingRule>
                    <RuleNumber>1</RuleNumber>
                    <Condition>
                        <KeyPrefixEquals>abc/</KeyPrefixEquals>
                        <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
                    </Condition>
                    <Redirect>
                        <RedirectType>Mirror</RedirectType>
                        <PassQueryString>true</PassQueryString>
                        <MirrorURL>http://www.test.com/</MirrorURL>
                        <MirrorPassQueryString>true</MirrorPassQueryString>
                        <MirrorFollowRedirect>true</MirrorFollowRedirect>
                        <MirrorCheckMd5>false</MirrorCheckMd5>
                        <MirrorHeaders>
                          <PassAll>true</PassAll>
                          <Pass>myheader-key1</Pass>
                          <Pass>myheader-key2</Pass>
                          <Remove>myheader-key3</Remove>
                          <Remove>myheader-key4</Remove>
                          <Set>
                            <Key>myheader-key5</Key>
                            <Value>myheader-value5</Value>
                          </Set>
                        </MirrorHeaders>
                    </Redirect>
                </RoutingRule>
                <RoutingRule>
                    <RuleNumber>2</RuleNumber>
                    <Condition>
                      <KeyPrefixEquals>abc/</KeyPrefixEquals>
                      <HttpErrorCodeReturnedEquals>404</HttpErrorCodeReturnedEquals>
                      <IncludeHeader>
                        <Key>host</Key>
                        <Equals>test.oss-cn-beijing-internal.aliyuncs.com</Equals>
                      </IncludeHeader>
                    </Condition>
                    <Redirect>
                      <RedirectType>AliCDN</RedirectType>
                      <Protocol>http</Protocol>
                      <HostName>www.test.com</HostName>
                      <PassQueryString>false</PassQueryString>
                      <ReplaceKeyWith>prefix/${key}.suffix</ReplaceKeyWith>
                      <HttpRedirectCode>301</HttpRedirectCode>
                    </Redirect>
                </RoutingRule>
            </RoutingRules>
        </WebsiteConfiguration>
        
    2) ossutil website --method get oss://bucket  [local_xml_file] [options]
       The command gets the website configuration of bucket
       If you input parameter local_xml_file,the configuration will be output to local_xml_file
       If you don't input parameter local_xml_file,the configuration will be output to stdout
	
    3) ossutil website --method delete oss://bucket [options]
       The command deletes the website configuration of bucket
`,
	sampleText: ` 
    1) put bucket website
       ossutil website --method put oss://bucket local_xml_file

    2) get website configuration to stdout
       ossutil website --method get oss://bucket
	
    3) get website configuration to local file
       ossutil website --method get oss://bucket local_xml_file
	
    4) delete website configuration
       ossutil website --method delete oss://bucket
`,
}

type bucketWebSiteOptionType struct {
	bucketName string
}

type BucketWebSiteCommand struct {
	command  Command
	bwOption bucketWebSiteOptionType
}

var bucketWebsiteCommand = BucketWebSiteCommand{
	command: Command{
		name:        "website",
		nameAlias:   []string{"website"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketWebSite,
		specEnglish: specEnglishBucketWebSite,
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
func (bwc *BucketWebSiteCommand) formatHelpForWhole() string {
	return bwc.command.formatHelpForWhole()
}

func (bwc *BucketWebSiteCommand) formatIndependHelp() string {
	return bwc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bwc *BucketWebSiteCommand) Init(args []string, options OptionMapType) error {
	return bwc.command.Init(args, options, bwc)
}

// RunCommand simulate inheritance, and polymorphism
func (bwc *BucketWebSiteCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bwc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(bwc.command.args[0], "")
	if err != nil {
		return err
	}

	bwc.bwOption.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bwc.PutBucketWebsite()
	} else if strMethod == "get" {
		err = bwc.GetBucketWebsite()
	} else if strMethod == "delete" {
		err = bwc.DeleteBucketWebsite()
	}
	return err
}

func (bwc *BucketWebSiteCommand) PutBucketWebsite() error {
	if len(bwc.command.args) < 2 {
		return fmt.Errorf("put bucket website need at least 2 parameters,the local xml file is empty")
	}

	xmlFile := bwc.command.args[1]
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

	// put bucket website
	client, err := bwc.command.ossClient(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketWebsiteXml(bwc.bwOption.bucketName, string(text))
}

func (bwc *BucketWebSiteCommand) confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("bucket website: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (bwc *BucketWebSiteCommand) GetBucketWebsite() error {
	client, err := bwc.command.ossClient(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	output, err := client.GetBucketWebsiteXml(bwc.bwOption.bucketName)
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

func (bwc *BucketWebSiteCommand) DeleteBucketWebsite() error {
	// delete bucket website
	client, err := bwc.command.ossClient(bwc.bwOption.bucketName)
	if err != nil {
		return err
	}

	return client.DeleteBucketWebsite(bwc.bwOption.bucketName)
}
