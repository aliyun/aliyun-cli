package lib

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketEncryption = SpecText{
	synopsisText: "设置、查询或者删除bucket的encryption配置",

	paramText: "bucket_url [options]",

	syntaxText: ` 
    ossutil bucket-encryption --method put oss://bucket --sse-algorithm algorithmName [--kms-masterkey-id  keyid] [--kms-data-encryption SM4]
    ossutil bucket-encryption --method get oss://bucket 
    ossutil bucket-encryption --method delete oss://bucket
`,
	detailHelpText: ` 
    bucket-encryption命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的encryption配置
    选项--sse-algorithm值只能是KMS、AES256、SM4
    当--sse-algorithm选项值为AES256时，不能输入选项--kms-masterkey-id
    当--sse-algorithm取值为KMS时, --kms-data-encryption可以取值SM4, 指定KMS服务使用SM4加密算法加密
    

用法:
    该命令有三种用法:
	
    1) ossutil bucket-encryption --method put oss://bucket --sse-algorithm algorithmName --kms-masterkey-id  keyid
        这个命令设置bucket的encryption配置,算法名为algorithmName，KMSMasterKeyID为keyid
	
    2) ossutil bucket-encryption --method get oss://bucket 
        这个命令查询bucket的encryption配置
	
    3) ossutil bucket-encryption --method delete oss://bucket
        这个命令删除bucket的encryption配置
`,
	sampleText: ` 
    1) 设置bucket的encryption配置，算法名为AES256
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm AES256
    
    2) 设置bucket的encryption配置，算法名为KMS，KMSMasterKeyID为123
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm KMS --kms-masterkey-id  123
    
    3) 设置bucket的encryption配置，算法名为SM4
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm SM4
	
    4) 查询bucket的encryption配置
       ossutil bucket-encryption --method get oss://bucket
	
    5) 删除bucket的encryption配置
       ossutil bucket-encryption --method delete oss://bucket
    
    6) 使用kms服务加密,加密算法为SM4
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm KMS --kms-data-encryption SM4
`,
}

var specEnglishBucketEncryption = SpecText{
	synopsisText: "Set, get or delete bucket encryption configuration",

	paramText: "bucket_url [options]",

	syntaxText: ` 
    ossutil bucket-encryption --method put oss://bucket --sse-algorithm algorithmName [--kms-masterkey-id  keyid] [--kms-data-encryption SM4]
    ossutil bucket-encryption --method get oss://bucket 
    ossutil bucket-encryption --method delete oss://bucket
`,
	detailHelpText: ` 
    bucket-encryption command can set, get and delete the encryption configuration of the oss bucket by set method option value to put, get, delete
    The option --sse-algorithm value can only be KMS, AES256, SM4.
    If the --sse-algorithm option value is AES256, you cannot input the option --kms-masterkey-id
    If the --sse-algorithm is kms, the value of --kms-data-encryption can be SM4, specifying that the KMS service uses SM4 encryption algorithm to encrypt
Usage:
    There are three usages for this command:
	
    1) ossutil bucket-encryption --method put oss://bucket --sse-algorithm algorithmName --kms-masterkey-id  keyid
        The command sets the encryption configuration of the bucket, the algorithm name is algorithmName and KMSMasterKeyID is keyid.
	
    2) ossutil bucket-encryption --method get oss://bucket 
        The command gets the encryption configuration of bucket

    3) ossutil bucket-encryption --method delete oss://bucket
        The command deletes the encryption configuration of bucket
`,
	sampleText: ` 
    1) set the encryption configuration of the bucket. The algorithm name is AES256.
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm AES256
    
    2) set the encryption configuration of the bucket. The algorithm name is KMS and the KMSMasterKeyID is 123.
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm KMS --kms-masterkey-id 123
    
    3) set the encryption configuration of the bucket. The algorithm name is SM4
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm SM4
	
    4) get bucket encryption configuration
       ossutil bucket-encryption --method get oss://bucket
	
    5) delete bucket encryption configuration
       ossutil bucket-encryption --method delete oss://bucket
    
    6) Using kms service encryption, the encryption algorithm is SM4
       ossutil bucket-encryption --method put oss://bucket --sse-algorithm KMS --kms-data-encryption SM4
`,
}

type BucketEncryptionCommand struct {
	command          Command
	bucketName       string
	encryptionResult oss.GetBucketEncryptionResult
}

var bucketEncryptionCommand = BucketEncryptionCommand{
	command: Command{
		name:        "bucket-encryption",
		nameAlias:   []string{"bucket-encryption"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseBucketEncryption,
		specEnglish: specEnglishBucketEncryption,
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
			OptionSSEAlgorithm,
			OptionKMSMasterKeyID,
			OptionKMSDataEncryption,
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
func (bec *BucketEncryptionCommand) formatHelpForWhole() string {
	return bec.command.formatHelpForWhole()
}

func (bec *BucketEncryptionCommand) formatIndependHelp() string {
	return bec.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bec *BucketEncryptionCommand) Init(args []string, options OptionMapType) error {
	return bec.command.Init(args, options, bec)
}

// RunCommand simulate inheritance, and polymorphism
func (bec *BucketEncryptionCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bec.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(bec.command.args[0], "")
	if err != nil {
		return err
	}

	bec.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bec.PutBucketEncryption()
	} else if strMethod == "get" {
		err = bec.GetBucketEncryption()
	} else if strMethod == "delete" {
		err = bec.DeleteBucketEncryption()
	}
	return err
}

func (bec *BucketEncryptionCommand) PutBucketEncryption() error {
	strAlgorithm, _ := GetString(OptionSSEAlgorithm, bec.command.options)
	strKeyId, _ := GetString(OptionKMSMasterKeyID, bec.command.options)
	strKmsDataEncryption, _ := GetString(OptionKMSDataEncryption, bec.command.options)

	// support sm4 algorithm
	//if strAlgorithm != string(oss.KMSAlgorithm) && strAlgorithm != string(oss.AESAlgorithm) {
	//	return fmt.Errorf("value of option --sse-algorithm must be KMS or AES256")
	//}

	if strAlgorithm == string(oss.AESAlgorithm) && len(strKeyId) > 0 {
		return fmt.Errorf("value of option --kms-masterkey-id must be empty if value of option --sse-algorithm is AES256")
	}

	var encryptionRule oss.ServerEncryptionRule
	encryptionRule.SSEDefault.SSEAlgorithm = strAlgorithm
	encryptionRule.SSEDefault.KMSMasterKeyID = strKeyId
	encryptionRule.SSEDefault.KMSDataEncryption = strKmsDataEncryption

	// put bucket encryption
	client, err := bec.command.ossClient(bec.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketEncryption(bec.bucketName, encryptionRule)
}

func (bec *BucketEncryptionCommand) GetBucketEncryption() error {
	client, err := bec.command.ossClient(bec.bucketName)
	if err != nil {
		return err
	}

	bec.encryptionResult, err = client.GetBucketEncryption(bec.bucketName)
	if err != nil {
		fmt.Printf("GetBucketEncryption error,info:%s\n", err.Error())
		return err
	}

	fmt.Printf("SSEAlgorithm:%s\n", bec.encryptionResult.SSEDefault.SSEAlgorithm)
	if bec.encryptionResult.SSEDefault.SSEAlgorithm == string(oss.KMSAlgorithm) {
		fmt.Printf("KMSMasterKeyID:%s\n", bec.encryptionResult.SSEDefault.KMSMasterKeyID)
		fmt.Printf("KMSDataEncryption:%s\n", bec.encryptionResult.SSEDefault.KMSDataEncryption)
	}

	fmt.Printf("\n\n")

	return nil
}

func (bec *BucketEncryptionCommand) DeleteBucketEncryption() error {
	client, err := bec.command.ossClient(bec.bucketName)
	if err != nil {
		return err
	}
	return client.DeleteBucketEncryption(bec.bucketName)
}
