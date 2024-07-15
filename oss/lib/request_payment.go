package lib

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseRequestPayment = SpecText{
	synopsisText: "设置、查询bucket的访问者付费配置",

	paramText: "bucket_url [payment_parameter] [options]",

	syntaxText: ` 
    ossutil request-payment --method put oss://bucket payment_parameter
    ossutil request-payment --method get oss://bucket 
`,
	detailHelpText: ` 
    request-payment命令通过设置method选项值为put、get, 可以设置、查询bucket的访问者付费配置
    选项--method为put时,参数只能为Requester, BucketOwner

用法:
    该命令有三种用法:
	
    1) ossutil request-payment --method put oss://bucket Requester
        这个命令设置由bucket的访问者付费
	
    2) ossutil request-payment --method put oss://bucket BucketOwner
        这个命令设置由bucket的拥有者付费
	
    3) ossutil request-payment --method get oss://bucket
        这个命令查询bucket的付费配置
`,
	sampleText: ` 
    1) 设置由bucket的访问者付费
       ossutil request-payment --method put oss://bucket Requester
    
    2) 设置由bucket的拥有者付费
       ossutil request-payment --method put oss://bucket BucketOwner
	
    3) 查询bucket的付费配置
       ossutil request-payment --method get oss://bucket
`,
}

var specEnglishRequestPayment = SpecText{
	synopsisText: "Set, get bucket request payment configuration",

	paramText: "bucket_url [payment_parameter] [options]",

	syntaxText: ` 
    ossutil request-payment --method put oss://bucket payment_parameter
    ossutil request-payment --method get oss://bucket 
`,
	detailHelpText: ` 
    request-payment command can set, get the bucket request payment configuration by set method option value to put, get
    If the --method option value is put, the parameter can only be Requester, BucketOwner
Usage:
    There are three usages for this command:
	
    1) ossutil request-payment --method put oss://bucket Requester
    This command sets that request is paid by the requester of the bucket

    2) ossutil request-payment --method put oss://bucket BucketOwner
    This command sets that request is paid by the owner of the bucket

    3) ossutil request-payment --method get oss://bucket
    This command query the bucket request payment configuration
`,
	sampleText: ` 
    1) setting request is paid by the requester of the bucket
       ossutil request-payment --method put oss://bucket Requester
    
    2) setting request is paid by the owner of the bucket
       ossutil request-payment --method put oss://bucket BucketOwner
	
    3) query the bucket request payment configuration 
       ossutil request-payment --method get oss://bucket
`,
}

type RequestPaymentCommand struct {
	command       Command
	bucketName    string
	paymentResult oss.RequestPaymentConfiguration
}

var requestPaymentCommand = RequestPaymentCommand{
	command: Command{
		name:        "request-payment",
		nameAlias:   []string{"request-payment"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseRequestPayment,
		specEnglish: specEnglishRequestPayment,
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
func (reqpc *RequestPaymentCommand) formatHelpForWhole() string {
	return reqpc.command.formatHelpForWhole()
}

func (reqpc *RequestPaymentCommand) formatIndependHelp() string {
	return reqpc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (reqpc *RequestPaymentCommand) Init(args []string, options OptionMapType) error {
	return reqpc.command.Init(args, options, reqpc)
}

// RunCommand simulate inheritance, and polymorphism
func (reqpc *RequestPaymentCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, reqpc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" {
		return fmt.Errorf("--method value is not in the optional value:put|get")
	}

	srcBucketUrL, err := GetCloudUrl(reqpc.command.args[0], "")
	if err != nil {
		return err
	}

	reqpc.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = reqpc.PutRequestPayment()
	} else if strMethod == "get" {
		err = reqpc.GetRequestPayment()
	}

	return err
}

func (reqpc *RequestPaymentCommand) PutRequestPayment() error {
	if len(reqpc.command.args) < 2 {
		return fmt.Errorf("missing parameter,payment parameter is empty")
	}

	strPayment := strings.ToLower(reqpc.command.args[1])

	if strPayment != strings.ToLower(string(oss.Requester)) &&
		strPayment != strings.ToLower(string(oss.BucketOwner)) {
		return fmt.Errorf("payment parameter must be %s or %s", string(oss.Requester), string(oss.BucketOwner))
	}

	// put bucket payment
	client, err := reqpc.command.ossClient(reqpc.bucketName)
	if err != nil {
		return err
	}

	var paymentConfig oss.RequestPaymentConfiguration
	if strPayment == strings.ToLower(string(oss.Requester)) {
		paymentConfig.Payer = string(oss.Requester)
	} else if strPayment == strings.ToLower(string(oss.BucketOwner)) {
		paymentConfig.Payer = string(oss.BucketOwner)

	}
	return client.SetBucketRequestPayment(reqpc.bucketName, paymentConfig)
}

func (reqpc *RequestPaymentCommand) GetRequestPayment() error {
	client, err := reqpc.command.ossClient(reqpc.bucketName)
	if err != nil {
		return err
	}

	reqpc.paymentResult, err = client.GetBucketRequestPayment(reqpc.bucketName)
	if err != nil {
		return err
	}

	fmt.Printf("\n%s\n", string(reqpc.paymentResult.Payer))

	return nil
}
