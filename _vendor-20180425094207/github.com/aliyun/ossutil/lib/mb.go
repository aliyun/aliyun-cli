package lib

import (
	"fmt"
	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"strings"
)

var storageClassList = []string{
	StorageStandard,
	StorageIA,
	StorageArchive,
}

func formatStorageClassString(sep string) string {
	return strings.Join(storageClassList, sep)
}

var specChineseMakeBucket = SpecText{

	synopsisText: "创建Bucket",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil mb oss://bucket [--acl acl] [--storage-class class] [-c file] 
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
    
    bucket的StorageClass有三种：
        ` + formatStorageClassString("\n        ") + `

    关于StorageClass的更多信息请参考：https://help.aliyun.com/document_detail/31959.html?spm=5176.doc31957.6.839.E1ifnh

用法：

    ossutil mb oss://bucket [--acl=acl] [--storage-class class] [-c file]
        当未指定--acl选项时，ossutil会在指定的身份凭证下创建指定bucket，所创建的bucket的acl
    为默认private。如果需要更改acl信息，可以使用set-acl命令。
        当未指定--storage-class选项时，ossutil创建的bucket的存储方式为默认存储方式：` + DefaultStorageClass + `。
        如果指定了--acl选项，ossutil会检查指定acl的合法性，如果acl非法，会进入交互模式，提
    示合法的acl输入，并询问acl信息。
        如果指定了--storage-class选项，ossutil会检查指定storage class的合法性。
`,

	sampleText: ` 
    1)ossutil mb oss://bucket1
    2)ossutil mb oss://bucket1 --acl=public-read-write
    3)ossutil mb oss://bucket1 --storage-class IA 
    4)ossutil mb oss://bucket1 --acl=public-read-write --storage-class IA 
`,
}

var specEnglishMakeBucket = SpecText{

	synopsisText: "Make Bucket",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil mb oss://bucket [--acl acl] [--storage-class class] [-c file] 
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

    There are three kinds of StorageClass:
        ` + formatStorageClassString("\n        ") + `

    More information about StorageClass see: https://help.aliyun.com/document_detail/31959.html?spm=5176.doc31957.6.839.E1ifnh

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
`,

	sampleText: ` 
    1)ossutil mb oss://bucket1
    2)ossutil mb oss://bucket1 --acl=public-read-write
    3)ossutil mb oss://bucket1 --storage-class IA 
    4)ossutil mb oss://bucket1 --acl=public-read-write --storage-class IA 
`,
}

// MakeBucketCommand is the command create bucket
type MakeBucketCommand struct {
	command Command
}

var makeBucketCommand = MakeBucketCommand{
	command: Command{
		name:        "mb",
		nameAlias:   []string{"cb", "pb"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseMakeBucket,
		specEnglish: specEnglishMakeBucket,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionACL,
			OptionStorageClass,
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionRetryTimes,
			OptionLanguage,
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

	aclStr, _ := GetString(OptionACL, mc.command.options)
	language, _ := GetString(OptionLanguage, mc.command.options)
	language = strings.ToLower(language)
	var op oss.Option
	if aclStr != "" {
		acl, err := mc.getACL(aclStr, language)
		if err != nil {
			return err
		}
		op = oss.ACL(acl)
	}

	return mc.ossCreateBucketRetry(client, cloudURL.bucket, op)
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
	options = append(options, oss.StorageClass(mc.getStorageClass()))
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
	return oss.StorageStandard
}
