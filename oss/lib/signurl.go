package lib

import (
	"fmt"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseSignurl = SpecText{

	synopsisText: "生成object下载链接",

	paramText: "cloud_url [meta] [options]",

	syntaxText: ` 
    ossutil sign cloud_url [--timeout t] [--version-id versionId] [--trafic-limit limitSpeed] [--disable-encode-slash] [--payer requester] [--query-param key:value]
`,

	detailHelpText: ` 
    该命令签名用户指定的cloud_url，生成经过签名的url可供第三方用户访问object，其中cloud_url
    必须为形如：oss://bucket/object的cloud_url，bucket和object不可缺少。通过--timeout选项指
    定url的过期时间，默认为60s。通过--version-id选项指定版本号。

用法：

    ossutil sign oss://bucket/object [--timeout t] [--version-id versionId] [--trafic-limit limitSpeed] [--disable-encode-slash] [--payer requester] [--query-param key:value]
`,

	sampleText: ` 
    ossutil sign oss://bucket1/object1
        生成oss://bucket1/object1的签名url，超时时间60s

    ossutil sign oss://bucket1/object1 --timeout 300
        生成oss://bucket1/object1的签名url，超时时间300s

    ossutil sign oss://tempb1/test%20a%2Bb' --encoding-type url
        生成oss://tempb1/'test a+b'的签名url，超时时间60s

    ossutil sign oss://bucket1/object1 --version-id versionId
        生成指定版本的 oss://bucket1/object1的签名url，超时时间60s
    
    ossutil sign oss://bucket1/object1 --trafic-limit 8388608
        生成oss://bucket1/object1的签名url, http限速为8388608(bit/s)
    
    ossutil sign oss://bucket1/dir/object1 --disable-encode-slash
        生成oss://bucket1/dir/object1的签名url, 对path中的'/'不进行编码
    
    ossutil sign oss://bucket1/object1  --payer requester
        生成oss://bucket1/dir/object1的签名url, 使用访问者付费模式

    ossutil sign oss://bucket1/object1.jpg  --query-param x-oss-process:image/resize,m_fixed,w_100,h_100/rotate,90
        生成处理过的图片 oss://bucket1/dir/object1.jpg的签名url 
`,
}

var specEnglishSignurl = SpecText{

	synopsisText: "Generate download link for object",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil sign cloud_url [--timeout t] [--version-id versionId] [--trafic-limit limitSpeed] [--disable-encode-slash] [--payer requester] [--query-param key:value]
`,

	detailHelpText: ` 
    The command will generate signature for user specified cloud_url. This signed url can
    be used by third-party to access the object. 
    Where, cloud_url must like: oss://bucket/object
    Use --timeout to specify the expire time of url, the default is 60s.
    Use --version-id to specify the version.
    Use --trafic-limit to specify the trafic speed
    use --disable-encode-slash to specify not encoding of '/' in url path section
    use --payer to specify request payment
    use --query-param to specify the query parameters, can be passed multiple times.

Usage:

    ossutil sign oss://bucket/object [--timeout t] [--version-id versionId] [--trafic-limit limitSpeed] [--disable-encode-slash] [--payer requester] [--query-param key:value]
`,

	sampleText: ` 
    ossutil sign oss://bucket1/object1
        Generate the signature of oss://bucket1/object1 with expire time 60s

    ossutil sign oss://bucket1/object1 --timeout 300
        Generate the signature of oss://bucket1/object1 with expire time 300s

    ossutil sign oss://tempb1/test%20a%2Bb' --encoding-type url
        Generate the signature of oss://tempb1/'test a+b' with expire time 60s

    ossutil sign oss://bucket1/object1 --version-id versionId
        Generate the signature of a specific version of oss://bucket1/object1 with  expire time 60s
    
    ossutil sign oss://bucket1/object1  --trafic-limit 8388608
        Generate the signature of oss://bucket1/object1, http limit speed is 8388608(bit/s)
    
    ossutil sign oss://bucket1/dir/object1 --disable-encode-slash
        Generate the signature of oss://bucket1/dir/object1,no encoding of '/' in url path section
    
    ossutil sign oss://bucket1/object1  --payer requester
        Generate the signature of oss://bucket1/object1, use requester payment

    ossutil sign oss://bucket1/object1.jpg  --query-param x-oss-process:image/resize,m_fixed,w_100,h_100/rotate,90
		Generate the signature of processed picture oss://bucket1/dir/object1.jpg
`,
}

// SignurlCommand definition
type SignurlCommand struct {
	command Command
	signUrl string
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
			OptionLogLevel,
			OptionVersionId,
			OptionTrafficLimit,
			OptionDisableEncodeSlash,
			OptionRequestPayer,
			//General options
			OptionPassword,
			OptionMode,
			OptionECSRoleName,
			OptionTokenTimeout,
			OptionRamRoleArn,
			OptionRoleSessionName,
			OptionSTSRegion,
			OptionUserAgent,
			OptionQueryParam,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
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
	versionId, _ := GetString(OptionVersionId, sc.command.options)
	trafficLimit, getErr := GetInt(OptionTrafficLimit, sc.command.options)
	if getErr == nil && trafficLimit < 0 {
		return fmt.Errorf("Option value of --trafic-limit must be greater than 0")
	}

	payer, _ := GetString(OptionRequestPayer, sc.command.options)
	if payer != "" && payer != strings.ToLower(string(oss.Requester)) {
		return fmt.Errorf("invalid request payer: %s, please check", payer)
	}
	query, _ := GetStrings(OptionQueryParam, sc.command.options)

	bucket, err := sc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	var options []oss.Option
	if len(versionId) > 0 {
		options = append(options, oss.VersionId(versionId))
	}

	if trafficLimit > 0 {
		options = append(options, oss.TrafficLimitParam(trafficLimit))
	}

	if payer != "" {
		options = append(options, oss.RequestPayerParam(oss.PayerType(payer)))
	}

	if len(query) > 0 {
		options, err = AddStringsToOption(query, options)
		if err != nil {
			return err
		}
	}

	str, err := sc.ossSign(bucket, cloudURL.object, timeout, options...)
	if err != nil {
		return err
	}
	sc.signUrl = str

	fmt.Println(str)
	return nil
}

func (sc *SignurlCommand) ossSign(bucket *oss.Bucket, object string, timeout int64, options ...oss.Option) (string, error) {
	str, err := bucket.SignURL(object, oss.HTTPMethod(DefaultMethod), timeout, options...)
	if err != nil {
		return str, ObjectError{err, bucket.BucketName, object}
	}

	disableEncodeSlash, _ := GetBool(OptionDisableEncodeSlash, sc.command.options)
	if !disableEncodeSlash {
		return str, nil
	}

	// replace %2F with /
	urlSlice := strings.SplitN(str, "?", 2)
	headStr := strings.Replace(urlSlice[0], "%2F", "/", -1)
	if len(urlSlice) == 2 {
		str = headStr + "?" + urlSlice[1]
	}
	return str, nil
}
