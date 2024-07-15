package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var storageClassList = []string{
	StorageStandard,
	StorageIA,
	StorageArchive,
	StorageColdArchive,
}

func formatStorageClassString(sep string) string {
	return strings.Join(storageClassList, sep)
}

var specChineseMakeBucket = SpecText{

	synopsisText: "创建Bucket",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil mb oss://bucket [--acl acl] [--storage-class class] [-c file] [--meta meta]
`,

	detailHelpText: ` 
    该命令在指定的身份凭证下创建bucket，可以在创建的同时通过--acl选项指定bucket的acl，但是
    不支持指定region的功能，所创建的bucket属于哪个region由配置中的endpoint决定。如果该账户
    下已经存在该bucket，不会报错，但如果bucket属于他人，会报错bucket已存在。
    
    关于config中的endpoint，请参考help config。
    
    关于region的更多信息，请参考https://help.aliyun.com/document_detail/31827.html?spm=5176.doc31826.6.138.rhUAjo中的Region部分。

ACL：

    bucket的acl有三种：
        ` + formatACLString(bucketACL, "\n        ") + `


    关于acl的更多信息请参考help set-acl。

StorageClass:
    
    bucket的StorageClass有四种：
        ` + formatStorageClassString("\n        ") + `

    关于StorageClass的更多信息请参考：https://help.aliyun.com/document_detail/31959.html?spm=5176.doc31957.6.839.E1ifnh

RedundancyType:

    bucket的RedundancyType有两种: LRS和ZRS; LRS是缺省值,表示本地容灾; ZRS表示更高可用的同城容灾, 数据将会同时保存在同一地域(Region)的3个可用区

用法：

    ossutil mb oss://bucket [--acl=acl] [--storage-class class] [--redundancy-type type] [-c file] [--meta meta]
        当未指定--acl选项时，ossutil会在指定的身份凭证下创建指定bucket，所创建的bucket的acl
    为默认private。如果需要更改acl信息，可以使用set-acl命令。
        当未指定--storage-class选项时，ossutil创建的bucket的存储方式为默认存储方式：` + DefaultStorageClass + `。
        如果指定了--acl选项，ossutil会检查指定acl的合法性，如果acl非法，会进入交互模式，提
    示合法的acl输入，并询问acl信息。
        如果指定了--storage-class选项，ossutil会检查指定storage class的合法性。
        如果输入--meta选项，可以设置bucket的header信息
`,

	sampleText: ` 
    1)ossutil mb oss://bucket1
    2)ossutil mb oss://bucket1 --acl=public-read-write
    3)ossutil mb oss://bucket1 --storage-class IA 
    4)ossutil mb oss://bucket1 --acl=public-read-write --storage-class IA
    5)ossutil mb oss://bucket1 --redundancy-type ZRS
    6)ossutil mb oss://bucket1 --meta X-Oss-Server-Side-Encryption:KMS#X-Oss-Server-Side-Data-Encryption:SM4
`,
}

var specEnglishMakeBucket = SpecText{

	synopsisText: "Make Bucket",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil mb oss://bucket [--acl acl] [--storage-class class] [--redundancy-type type] [-c file] 
`,

	detailHelpText: ` 
    The command create bucket under the specified credentials. You can specify bucket acl 
    information through --acl option meanwhile. But you can not specify the region for the 
    bucket, the region is decided by endpoint in config file. If the bucket existed under 
    the credentials, no error occurs, but if the bucket belongs to others, error occurs.

    More information about endpoint in config, see help config.

    More information about region, see Region section in https://help.aliyun.com/document_detail/31827.html?spm=5176.doc31826.6.138.rhUAjo.

ACL:

    ossutil supports following bucket acls, shorthand versions in brackets:
        ` + formatACLString(bucketACL, "\n        ") + `

    More information about acl, see help set-acl.

StorageClass:

    There are four kinds of StorageClass:
        ` + formatStorageClassString("\n        ") + `

    More information about StorageClass see: https://help.aliyun.com/document_detail/31959.html?spm=5176.doc31957.6.839.E1ifnh

RedundancyType:

    There are two types of bucket redundancyType: LRS and ZRS; LRS is the default value, specifies locally redundant storage; ZRS specifies higher availability of redundancy storage, The data will be stored in the 3 availabe zones of the same region

Usage:

    ossutil mb oss://bucket [--acl=acl] [--storage-class class] [-c file]
        If you create bucket without --acl option, ossutil will create bucket under the 
    specified credentials and the bucket acl is private, if you want to change acl, please 
    use set-acl command. 
        If you create bucket without --storage-class option, the storage class of bucket will
     be the default one: ` + DefaultStorageClass + `. 
        If you create bucket with --acl option, ossutil will check the validity of acl, if 
    invalid, ossutil will enter interactive mode, prompt the valid acls and ask you for it. 
        If you create bucket with --storage-class option, ossutil will check the validity of 
    storage class. 
        If you create bucket with --meta option, you can set the header information of the bucket.
`,

	sampleText: ` 
    1)ossutil mb oss://bucket1
    2)ossutil mb oss://bucket1 --acl=public-read-write
    3)ossutil mb oss://bucket1 --storage-class IA 
    4)ossutil mb oss://bucket1 --acl=public-read-write --storage-class IA 
    5)ossutil mb oss://bucket1 --redundancy-type ZRS
    6)ossutil mb oss://bucket1 --meta X-Oss-Server-Side-Encryption:KMS#X-Oss-Server-Side-Data-Encryption:SM4
`,
}

// MakeBucketCommand is the command create bucket
type MakeBucketCommand struct {
	command  Command
	mcOption bucketOptionType
}

type bucketOptionType struct {
	ossMeta string
}

var makeBucketCommand = MakeBucketCommand{
	command: Command{
		name:        "mb",
		nameAlias:   []string{"cb", "pb"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseMakeBucket,
		specEnglish: specEnglishMakeBucket,
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
			OptionRetryTimes,
			OptionLanguage,
			OptionACL,
			OptionStorageClass,
			OptionLogLevel,
			OptionRedundancyType,
			OptionPassword,
			OptionMode,
			OptionMeta,
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
func (mc *MakeBucketCommand) formatHelpForWhole() string {
	return mc.command.formatHelpForWhole()
}

func (mc *MakeBucketCommand) formatIndependHelp() string {
	return mc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (mc *MakeBucketCommand) Init(args []string, options OptionMapType) error {
	return mc.command.Init(args, options, mc)
}

// RunCommand simulate inheritance, and polymorphism
func (mc *MakeBucketCommand) RunCommand() error {

	cloudURL, err := CloudURLFromString(mc.command.args[0], "")
	if err != nil {
		return err
	}

	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", mc.command.args[0])
	}

	if cloudURL.object != "" {
		return fmt.Errorf("invalid cloud url: %s, object not empty, upload object please use \"cp\" command", mc.command.args[0])
	}

	client, err := mc.command.ossClient(cloudURL.bucket)
	if err != nil {
		return err
	}
	var op []oss.Option
	mc.mcOption.ossMeta, _ = GetString(OptionMeta, mc.command.options)
	if mc.mcOption.ossMeta != "" {
		metas, err := mc.command.parseHeaders(mc.mcOption.ossMeta, false)
		if err != nil {
			return err
		}
		options, err := mc.command.getOSSOptions(headerOptionMap, metas)
		if err != nil {
			return err
		}
		op = append(op, options...)
	}
	if len(mc.command.args) >= 2 {
		return mc.createBucketXmlFile(client, cloudURL.bucket, mc.command.args[1], op)
	}

	aclStr, _ := GetString(OptionACL, mc.command.options)
	language, _ := GetString(OptionLanguage, mc.command.options)
	language = strings.ToLower(language)
	strRedundancy, _ := GetString(OptionRedundancyType, mc.command.options)

	if aclStr != "" {
		acl, err := mc.getACL(aclStr, language)
		if err != nil {
			return err
		}
		op = append(op, oss.ACL(acl))
	}

	if strRedundancy != "" {
		if strings.ToUpper(strRedundancy) != string(oss.RedundancyLRS) && strings.ToUpper(strRedundancy) != string(oss.RedundancyZRS) {
			return fmt.Errorf("--redundancy-type muse be %s or %s", string(oss.RedundancyLRS), string(oss.RedundancyZRS))
		}
		redundancyType := oss.DataRedundancyType(strings.ToUpper(strRedundancy))
		op = append(op, oss.RedundancyType(redundancyType))
	}

	return mc.ossCreateBucketRetry(client, cloudURL.bucket, op...)
}

func (mc *MakeBucketCommand) createBucketXmlFile(client *oss.Client, bucketName string, fileName string, options []oss.Option) error {
	var op []oss.Option
	// parsing the xml file
	file, err := os.Open(fileName)
	if err != nil {
		return err
	}
	defer file.Close()
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}
	if len(options) > 0 {
		op = append(op, options...)
	}
	return client.CreateBucketXml(bucketName, string(text), op...)
}

func (mc *MakeBucketCommand) getACL(aclStr, language string) (oss.ACLType, error) {
	acl, err := mc.command.checkACL(aclStr, bucketACL)
	if err != nil {
		if language == LEnglishLanguage {
			fmt.Printf("Invalid acl: %s\n", aclStr)
			fmt.Printf("Acceptable acls:\n\t%s\nPlease enter the acl you want to set on the bucket:", formatACLString(bucketACL, "\n\t"))
		} else {
			fmt.Printf("acl: %s非法\n", aclStr)
			fmt.Printf("合法的acl有:\n\t%s\n请输入你想设置的acl：", formatACLString(bucketACL, "\n\t"))
		}
		if _, err := fmt.Scanln(&aclStr); err != nil {
			return "", fmt.Errorf("invalid acl: %s, please check", aclStr)
		}
		acl, err = mc.command.checkACL(aclStr, bucketACL)
	}
	return acl, err
}

func (mc *MakeBucketCommand) ossCreateBucketRetry(client *oss.Client, bucket string, options ...oss.Option) error {
	storageClass := mc.getStorageClass()
	if storageClass != oss.StorageStandard {
		options = append(options, oss.StorageClass(storageClass))
	}
	retryTimes, _ := GetInt(OptionRetryTimes, mc.command.options)
	for i := 1; ; i++ {
		err := client.CreateBucket(bucket, options...)
		if err == nil {
			return err
		}
		if int64(i) >= retryTimes {
			return BucketError{err, bucket}
		}
	}
}

func (mc *MakeBucketCommand) getStorageClass() oss.StorageClassType {
	storageClass, _ := GetString(OptionStorageClass, mc.command.options)
	if strings.EqualFold(storageClass, StorageIA) {
		return oss.StorageIA
	}
	if strings.EqualFold(storageClass, StorageArchive) {
		return oss.StorageArchive
	}
	if strings.EqualFold(storageClass, StorageColdArchive) {
		return oss.StorageColdArchive
	}
	return oss.StorageStandard
}
