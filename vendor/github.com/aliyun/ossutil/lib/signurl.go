package lib

import (
	"fmt"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseSignurl = SpecText{

	synopsisText: "生成object下载链接",

	paramText: "cloud_url [meta] [options]",

	syntaxText: ` 
    ossutil sign cloud_url [--timeout t]
`,

	detailHelpText: ` 
    该命令签名用户指定的cloud_url，生成经过签名的url可供第三方用户访问object，其中cloud_url
    必须为形如：oss://bucket/object的cloud_url，bucket和object不可缺少。通过--timeout选项指
    定url的过期时间，默认为60s。

用法：

    ossutil sign oss://bucket/object [--timeout t]
`,

	sampleText: ` 
    ossutil sign oss://bucket1/object1
        生成oss://bucket1/object1的签名url，超时时间60s

    ossutil sign oss://bucket1/object1 --timeout 300
        生成oss://bucket1/object1的签名url，超时时间300s

    ossutil sign oss://tempb1/test%20a%2Bb' --encoding-type url
        生成oss://tempb1/'test a+b'的签名url，超时时间60s
`,
}

var specEnglishSignurl = SpecText{

	synopsisText: "Generate download link for object",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil sign cloud_url [--timeout t]
`,

	detailHelpText: ` 
    The command will generate signature for user specified cloud_url. This signed url can
    be used by third-party to access the object. 
    Where, cloud_url must like: oss://bucket/object
    Use --timeout to specify the expire time of url, the default is 60s.

Usage:

    ossutil sign oss://bucket/object [--timeout t]
`,

	sampleText: ` 
    ossutil sign oss://bucket1/object1
        Generate the signature of oss://bucket1/object1 with expire time 60s

    ossutil sign oss://bucket1/object1 --timeout 300
        Generate the signature of oss://bucket1/object1 with expire time 300s

    ossutil sign oss://tempb1/test%20a%2Bb' --encoding-type url
        Generate the signature of oss://tempb1/'test a+b' with expire time 60s
`,
}

// SignurlCommand definition
type SignurlCommand struct {
	command Command
}

var signURLCommand = SignurlCommand{
	command: Command{
		name:        "sign",
		nameAlias:   []string{},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseSignurl,
		specEnglish: specEnglishSignurl,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionTimeout,
			OptionEncodingType,
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
		},
	},
}

// function for FormatHelper interface
func (sc *SignurlCommand) formatHelpForWhole() string {
	return sc.command.formatHelpForWhole()
}

func (sc *SignurlCommand) formatIndependHelp() string {
	return sc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (sc *SignurlCommand) Init(args []string, options OptionMapType) error {
	return sc.command.Init(args, options, sc)
}

// RunCommand simulate inheritance, and polymorphism
func (sc *SignurlCommand) RunCommand() error {
	encodingType, _ := GetString(OptionEncodingType, sc.command.options)
	cloudURL, err := ObjectURLFromString(sc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	timeout, _ := GetInt(OptionTimeout, sc.command.options)

	bucket, err := sc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	str, err := sc.ossSign(bucket, cloudURL.object, timeout)
	if err != nil {
		return err
	}

	fmt.Println(str)
	return nil
}

func (sc *SignurlCommand) ossSign(bucket *oss.Bucket, object string, timeout int64, options ...oss.Option) (string, error) {
	str, err := bucket.SignURL(object, oss.HTTPMethod(DefaultMethod), timeout, options...)
	if err == nil {
		return str, nil
	}

	return str, ObjectError{err, bucket.BucketName, object}
}
