package lib

import (
	"fmt"
	"io"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseCat = SpecText{
	synopsisText: "将文件内容输出到标准输出",

	paramText: "object [options]",

	syntaxText: ` 
	ossutil cat oss://bucket/object [--payer requester] [--version-id versionId]
`,
	detailHelpText: ` 
    cat命令可以将oss的object内容输出到标准输出,object内容最好是文本格式

用法:
    该命令仅有一种用法:
	
    1) ossutil cat oss://bucket/object [--version-id versionId] [--payer requester]
       将object内容输出到标准输出
`,
	sampleText: ` 
    1) 将object内容输出到标准输出
       ossutil cat oss://bucket/object
    
    2) 将object指定版本内容输出到标准输出
       ossutil cat oss://bucket/object --version-id versionId
    
    3) 访问者付费模式
       ossutil cat oss://bucket/object --payer requester
`,
}

var specEnglishCat = SpecText{
	synopsisText: "Output object content to standard output",

	paramText: "object [options]",

	syntaxText: ` 
	ossutil cat oss://bucket/object [--payer requester] [--version-id versionId]
`,
	detailHelpText: ` 
	The cat command can output the object content of oss to standard output
    The object content is preferably text format

Usage:
    There is only one usage for this command:
	
    1) ossutil cat oss://bucket/object [--version-id versionId] [--payer requester]
       The command output object content to standard output
`,
	sampleText: ` 
    1) output object content to standard output
       ossutil cat oss://bucket/object
    
    2) output the object's specified version content to standard output
       ossutil cat oss://bucket/object --version-id versionId
    
    3) output object content with requester payment
       ossutil cat oss://bucket/object --payer requester
`,
}

type catOptionType struct {
	bucketName   string
	objectName   string
	encodingType string
}

type CatCommand struct {
	command       Command
	catOption     catOptionType
	commonOptions []oss.Option
}

var catCommand = CatCommand{
	command: Command{
		name:        "cat",
		nameAlias:   []string{"cat"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseCat,
		specEnglish: specEnglishCat,
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
			OptionEncodingType,
			OptionLogLevel,
			OptionVersionId,
			OptionRequestPayer,
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
func (catc *CatCommand) formatHelpForWhole() string {
	return catc.command.formatHelpForWhole()
}

func (catc *CatCommand) formatIndependHelp() string {
	return catc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (catc *CatCommand) Init(args []string, options OptionMapType) error {
	return catc.command.Init(args, options, catc)
}

// RunCommand simulate inheritance, and polymorphism
func (catc *CatCommand) RunCommand() error {
	catc.catOption.encodingType, _ = GetString(OptionEncodingType, catc.command.options)
	srcBucketUrL, err := GetCloudUrl(catc.command.args[0], catc.catOption.encodingType)
	if err != nil {
		return err
	}

	if srcBucketUrL.object == "" {
		return fmt.Errorf("object key is empty")
	}

	catc.catOption.bucketName = srcBucketUrL.bucket
	catc.catOption.objectName = srcBucketUrL.object

	// check object exist or not
	client, err := catc.command.ossClient(catc.catOption.bucketName)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(catc.catOption.bucketName)
	if err != nil {
		return err
	}

	payer, _ := GetString(OptionRequestPayer, catc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		catc.commonOptions = append(catc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	var options []oss.Option
	options = append(options, catc.commonOptions...)

	versionId, _ := GetString(OptionVersionId, catc.command.options)
	if len(versionId) > 0 {
		options = append(options, oss.VersionId(versionId))
	}

	body, err := bucket.GetObject(catc.catOption.objectName, options...)
	if err != nil {
		return err
	}

	defer body.Close()
	io.Copy(os.Stdout, body)
	fmt.Printf("\n")

	return err
}
