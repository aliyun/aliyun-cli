package lib

import (
	"fmt"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseBucketVersioning = SpecText{
	synopsisText: "设置、查询bucket的versioning配置",

	paramText: "bucket_url [versioning_parameter] [options]",

	syntaxText: ` 
    ossutil bucket-versioning --method put oss://bucket versioning_parameter
    ossutil bucket-versioning --method get oss://bucket 
`,
	detailHelpText: ` 
    bucket-versioning命令通过设置method选项值为put、get、可以设置、查询bucket的versioning配置
    选项--method为put时,versioning状态参数只能为enabled、suspended

用法:
    该命令有三种用法:
	
    1) ossutil bucket-versioning --method put oss://bucket enabled
        这个命令开通bucket的versioning功能
	
    2) ossutil bucket-versioning --method put oss://bucket suspended
        这个命令关闭bucket的versioning功能
	
    3)  ossutil bucket-versioning --method get oss://bucket
        这个命令查询bucket的vesioning状态
`,
	sampleText: ` 
    1) 开通bucket的versioning功能
       ossutil bucket-versioning --method put oss://bucket enabled
    
    2) 关闭bucket的versioning功能
       ossutil bucket-versioning --method put oss://bucket suspended
	
    3) 查询bucket的versioning状态
       ossutil bucket-versioning --method get oss://bucket
`,
}

var specEnglishBucketVersioning = SpecText{
	synopsisText: "Set, get bucket versioning configuration",

	paramText: "bucket_url [versioning_parameter] [options]",

	syntaxText: ` 
    ossutil bucket-versioning --method put oss://bucket versioning_parameter
    ossutil bucket-versioning --method get oss://bucket 
`,
	detailHelpText: ` 
    bucket-versioning command can set, get the versioning configuration of the oss bucket by set method option value to put, get
    If the --method option value is put,the versioning status value can only be enabled, suspended, 
Usage:
    There are three usages for this command:
	
    1) ossutil bucket-versioning --method put oss://bucket enabled
    This command enables the bucket versioning

    2) ossutil bucket-versioning --method put oss://bucket suspended
    This command disables the bucket versioning

    3)  ossutil bucket-versioning --method get oss://bucket
    This command query the bucket versioning status
`,
	sampleText: ` 
    1) set bucket versioning enabled
       ossutil bucket-versioning --method put oss://bucket enabled
    
    2) set bucket versioning disable
       ossutil bucket-versioning --method put oss://bucket suspended
	
    3) get bucket versioning status
       ossutil bucket-versioning --method get oss://bucket
`,
}

type BucketVersioningCommand struct {
	command          Command
	bucketName       string
	versioningResult oss.GetBucketVersioningResult
}

var bucketVersioningCommand = BucketVersioningCommand{
	command: Command{
		name:        "bucket-versioning",
		nameAlias:   []string{"bucket-versioning"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseBucketVersioning,
		specEnglish: specEnglishBucketVersioning,
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
func (bvc *BucketVersioningCommand) formatHelpForWhole() string {
	return bvc.command.formatHelpForWhole()
}

func (bvc *BucketVersioningCommand) formatIndependHelp() string {
	return bvc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (bvc *BucketVersioningCommand) Init(args []string, options OptionMapType) error {
	return bvc.command.Init(args, options, bvc)
}

// RunCommand simulate inheritance, and polymorphism
func (bvc *BucketVersioningCommand) RunCommand() error {
	strMethod, _ := GetString(OptionMethod, bvc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" {
		return fmt.Errorf("--method value is not in the optional value:put|get")
	}

	srcBucketUrL, err := GetCloudUrl(bvc.command.args[0], "")
	if err != nil {
		return err
	}

	bvc.bucketName = srcBucketUrL.bucket

	if strMethod == "put" {
		err = bvc.PutBucketVersioning()
	} else if strMethod == "get" {
		err = bvc.GetBucketVersioning()
	}

	return err
}

func (bvc *BucketVersioningCommand) PutBucketVersioning() error {

	if len(bvc.command.args) < 2 {
		return fmt.Errorf("missing parameter,versioning status is empty")
	}

	strVersion := bvc.command.args[1]

	if strings.ToUpper(strVersion) != strings.ToUpper(string(oss.VersionEnabled)) &&
		strings.ToUpper(strVersion) != strings.ToUpper(string(oss.VersionSuspended)) {
		return fmt.Errorf("version status must be %s or %s", string(oss.VersionEnabled),
			string(oss.VersionSuspended))
	}

	// put bucket versioning
	client, err := bvc.command.ossClient(bvc.bucketName)
	if err != nil {
		return err
	}

	var versioningConfig oss.VersioningConfig
	if strings.ToUpper(strVersion) == strings.ToUpper(string(oss.VersionEnabled)) {
		versioningConfig.Status = string(oss.VersionEnabled)
	} else if strings.ToUpper(strVersion) == strings.ToUpper(string(oss.VersionSuspended)) {
		versioningConfig.Status = string(oss.VersionSuspended)

	}
	return client.SetBucketVersioning(bvc.bucketName, versioningConfig)
}

func (bvc *BucketVersioningCommand) GetBucketVersioning() error {
	client, err := bvc.command.ossClient(bvc.bucketName)
	if err != nil {
		return err
	}

	bvc.versioningResult, err = client.GetBucketVersioning(bvc.bucketName)
	if err != nil {
		return err
	}

	if bvc.versioningResult.Status == "" {
		bvc.versioningResult.Status = "null"
	}

	fmt.Printf("\nbucket versioning status:%s\n", bvc.versioningResult.Status)

	return nil
}
