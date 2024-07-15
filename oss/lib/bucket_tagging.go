package lib

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketTag = SpecText{
	synopsisText: "设置、查询或者删除bucket的tag配置",

	paramText: "bucket_url [tag_parameter] [options]",

	syntaxText: ` 
    ossutil bucket-tagging --method put oss://bucket key#value
    ossutil bucket-tagging --method get oss://bucket 
    ossutil bucket-tagging --method delete oss://bucket
`,
	detailHelpText: ` 
    bucket-tagging命令通过设置method选项值为put、get、delete,可以设置、查询或者删除bucket的tag配置
    每个tag的key和value必须以字符'#'分隔,最多可以连续输入10个tag信息

用法:
    该命令有三种用法:
	
    1) ossutil bucket-tagging --method put oss://bucket  tagkey#tagvalue
        这个命令设置bucket的tag配置,key和value分别为tagkey、tagvalue
	
    2) ossutil bucket-tagging --method get oss://bucket 
        这个命令查询bucket的tag配置
	
    3)  ossutil bucket-tagging --method delete oss://bucket
        这个命令删除bucket的tag配置
`,
	sampleText: ` 
    1) 设置bucket的tag配置
       ossutil bucket-tagging --method put oss://bucket  tagkey#tagvalue
    
    2) 设置bucket的多个tag配置
       ossutil bucket-tagging --method put oss://bucket  tagkey1#tagvalue1 tagkey2#tagvalue2
	
    3) 查询bucket的tag配置
       ossutil bucket-tagging --method get oss://bucket
	
    4) 删除bucket的tag配置
       ossutil bucket-tagging --method delete oss://bucket
`,
}

var specEnglishBucketTag = SpecText{
	synopsisText: "Set, get or delete bucket tag configuration",

	paramText: "bucket_url [tag_parameter] [options]",

	syntaxText: ` 
    ossutil bucket-tagging --method put oss://bucket key#value
    ossutil bucket-tagging --method get oss://bucket 
    ossutil bucket-tagging --method delete oss://bucket
`,
	detailHelpText: ` 
    bucket-tagging command can set, get and delete the tag configuration of the oss bucket by set method option value to put, get, delete
    the key and value of each tag must be separated by the character '#', you can enter up to 10 tag parameters.
Usage:
    There are three usages for this command:
	
    1) ossutil bucket-tagging --method put oss://bucket tagkey#tagvalue
        The command sets the tag configuration of the bucket. The key and value are tagkey and tagvalue
	
    2) ossutil bucket-tagging --method get oss://bucket 
        The command gets the tag configuration of bucket

    3) ossutil bucket-tagging --method delete oss://bucket
        The command deletes the tag configuration of bucket
`,
	sampleText: ` 
    1) set bucket tag configuration with one tag   
       ossutil bucket-tagging --method put oss://bucket tagkey#tagvalue
    
    2) set bucket tag configuration with serveral tags
       ossutil bucket-tagging --method put oss://bucket tagkey1#tagvalue1 tagkey2#tagvalue2 

    3) get bucket tag configuration
       ossutil bucket-tagging --method get oss://bucket
	
    4) delete bucket tag configuration
       ossutil bucket-tagging --method delete oss://bucket
`,
}

type BucketTagCommand struct {
	command    Command
	bucketName string
	tagResult  oss.GetBucketTaggingResult
}

var bucketTagCommand = BucketTagCommand{
	command: Command{
		name:        "bucket-tagging",
		nameAlias:   []string{"bucket-tagging"},
		minArgc:     1,
		maxArgc:     11,
		specChinese: specChineseBucketTag,
		specEnglish: specEnglishBucketTag,
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
func (btc *BucketTagCommand) formatHelpForWhole() string {
	return btc.command.formatHelpForWhole()
}

func (btc *BucketTagCommand) formatIndependHelp() string {
	return btc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (btc *BucketTagCommand) Init(args []string, options OptionMapType) error {
	return btc.command.Init(args, options, btc)
}

// RunCommand simulate inheritance, and polymorphism
func (btc *BucketTagCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, btc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}

	srcBucketUrL, err := GetCloudUrl(btc.command.args[0], "")
	if err != nil {
		return err
	}

	btc.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = btc.PutBucketTag()
	} else if strMethod == "get" {
		err = btc.GetBucketTag()
	} else if strMethod == "delete" {
		err = btc.DeleteBucketTag()
	}
	return err
}

func (btc *BucketTagCommand) PutBucketTag() error {
	if len(btc.command.args) < 2 {
		return fmt.Errorf("missing parameter,the tag value is empty")
	}

	var tagging oss.Tagging
	tagList := btc.command.args[1:len(btc.command.args)]
	for _, tag := range tagList {
		pSlice := strings.Split(tag, "#")
		if len(pSlice) != 2 {
			return fmt.Errorf("%s error,tag name and tag value must be separated by #", tag)
		}
		tagging.Tags = append(tagging.Tags, oss.Tag{Key: pSlice[0], Value: pSlice[1]})
	}

	// put bucket tag
	client, err := btc.command.ossClient(btc.bucketName)
	if err != nil {
		return err
	}

	return client.SetBucketTagging(btc.bucketName, tagging)
}

func (btc *BucketTagCommand) GetBucketTag() error {
	client, err := btc.command.ossClient(btc.bucketName)
	if err != nil {
		return err
	}

	btc.tagResult, err = client.GetBucketTagging(btc.bucketName)
	if err != nil {
		return err
	}

	if len(btc.tagResult.Tags) > 0 {
		fmt.Printf("%-10s%s\t%s\n", "index", "tag key", "tag value")
		fmt.Printf("---------------------------------------------------\n")
	}

	for index, tag := range btc.tagResult.Tags {
		fmt.Printf("%-10d\"%s\"\t\"%s\"\n", index, tag.Key, tag.Value)
	}

	fmt.Printf("\n\n")

	return nil
}

func (btc *BucketTagCommand) DeleteBucketTag() error {
	client, err := btc.command.ossClient(btc.bucketName)
	if err != nil {
		return err
	}
	return client.DeleteBucketTagging(btc.bucketName)
}
