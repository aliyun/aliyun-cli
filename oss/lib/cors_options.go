package lib

import (
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseOptions = SpecText{
	synopsisText: "向oss发送http options请求,用于CORS检测",

	paramText: "oss_url [options]",

	syntaxText: ` 
    ossutil cors-options --acr-method <value> --origin <value> --acr-headers <value> oss://bucket/[object] [options]
`,
	detailHelpText: ` 
    cors-options命令向oss发送http options请求
    --acr-method、--origin、--acr-headers分别对应http header:Access-Control-Request-Method、Origin、Access-Control-Request-Headers
    --acr-headers如果有多个取值,各个header用逗号分隔,再加上双引号,比如  --acr-headers "header1,header2,header3"

用法:
    该命令有一种用法:
	
    1) ossutil cors-options --acr-method PUT --origin "www.aliyuncs.com" --acr-header x-oss-meta-author oss://bucket/ [options]
       向oss发送options请求,Origin、Access-Control-Request-Method、Access-Control-Request-Headers分别为www.aliyuncs.com、PUT、x-oss-meta-author
`,
	sampleText: ` 
    1) 发送options请求,Access-Control-Request-Method为PUT
       ossutil cors-options --acr-method PUT --origin "www.aliyuncs.com" --acr-header x-oss-meta-author oss://bucket/
    
    2) 发送options请求,有多个header参数,Access-Control-Request-Method为GET
       ossutil cors-options --acr-method GET --origin "www.aliyuncs.com" --acr-header "x-oss-meta-author1,x-oss-meta-author2" oss://bucket/
`,
}

var specEnglishOptions = SpecText{
	synopsisText: "Send http options request to oss for CORS detection",

	paramText: "oss_url [options]",

	syntaxText: ` 
    ossutil cors-options --acr-method <value> --origin <value> --acr-headers <value> oss://bucket/[object] [options]
`,
	detailHelpText: ` 
    The cors-options command sends an http options request to oss
    --acr-method, --origin, --acr-headers correspond to http header:Access-Control-Request-Method, Origin, Access-Control-Request-Headers
    If --acr-headers have multiple values, each header is separated by a comma, followed by double quotes, for example: --acr-headers "header1,header2,header3"

Usage:
    There are one usage for this command:
	
    1) ossutil cors-options --acr-method PUT --origin "www.aliyuncs.com" --acr-header x-oss-meta-author oss://bucket/ [options]
       sends an http options request to oss,Origin、Access-Control-Request-Method、Access-Control-Request-Headers values are www.aliyuncs.com、PUT、x-oss-meta-author
`,
	sampleText: ` 
    1) sends an http options request,Access-Control-Request-Method value is PUT
    ossutil cors-options --acr-method PUT --origin "www.aliyuncs.com" --acr-header x-oss-meta-author oss://bucket/
 
    2) sends an http options request,there are multipule values for --acr-header,Access-Control-Request-Method value is GET
    ossutil cors-options --acr-method GET --origin "www.aliyuncs.com" --acr-header "x-oss-meta-author1,x-oss-meta-author2" oss://bucket/
`,
}

type OptionsCommand struct {
	command Command
}

var corsOptionsCommand = OptionsCommand{
	command: Command{
		name:        "cors-options",
		nameAlias:   []string{"cors-options"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseOptions,
		specEnglish: specEnglishOptions,
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
			OptionEncodingType,
			OptionOrigin,
			OptionAcrMethod,
			OptionAcrHeaders,
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
func (opsc *OptionsCommand) formatHelpForWhole() string {
	return opsc.command.formatHelpForWhole()
}

func (opsc *OptionsCommand) formatIndependHelp() string {
	return opsc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (opsc *OptionsCommand) Init(args []string, options OptionMapType) error {
	return opsc.command.Init(args, options, opsc)
}

// RunCommand simulate inheritance, and polymorphism
func (opsc *OptionsCommand) RunCommand() error {
	strOrigin, _ := GetString(OptionOrigin, opsc.command.options)
	strMethod, _ := GetString(OptionAcrMethod, opsc.command.options)
	strAcrHeaders, _ := GetString(OptionAcrHeaders, opsc.command.options)

	strEncodingType, _ := GetString(OptionEncodingType, opsc.command.options)
	srcBucketUrL, err := GetCloudUrl(opsc.command.args[0], strEncodingType)
	if err != nil {
		return err
	}

	options := []oss.Option{}

	if len(strOrigin) > 0 {
		options = append(options, oss.Origin(strOrigin))
	}
	if len(strMethod) > 0 {
		options = append(options, oss.ACReqMethod(strings.ToUpper(strMethod)))
	}
	if len(strAcrHeaders) > 0 {
		options = append(options, oss.ACReqHeaders(strAcrHeaders))
	}

	objectName := srcBucketUrL.object

	client, err := opsc.command.ossClient(srcBucketUrL.bucket)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(srcBucketUrL.bucket)
	if err != nil {
		return err
	}

	respHeader, err := bucket.OptionsMethod(objectName, options...)
	if err != nil {
		return err
	}

	exclude := map[string]bool{}
	exclude["Connection"] = true
	exclude["ontent-Length"] = true
	exclude["Date"] = true
	exclude["Server"] = true
	exclude["X-Oss-Request-Id"] = true
	exclude["X-Oss-Server-Time"] = true
	exclude["Content-Length"] = true

	respHeader.WriteSubset(os.Stdout, exclude)

	return nil
}
