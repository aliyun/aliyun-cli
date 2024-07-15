package lib

import (
	"fmt"
	"strings"
)

var specChineseMkdir = SpecText{
	synopsisText: "创建一个目录,在oss中目录名字有后缀字符'/'",

	paramText: "dir_name [options]",

	syntaxText: ` 
	ossutil mkdir oss://bucket/dir_name
`,

	detailHelpText: ` 
    1) 如果输入的参数没有以后缀字符'/'结尾，工具会自动添加
    2) 如果目录已经存在,会提示报错
    3) 如果输入的参数包含多级目录,只会创建最后的那一个目录
`,

	sampleText: ` 
	1) 创建一个目录
      ossutil mkdir oss://bucket/dir
	
    2) 创建一个多级目录
      ossutil mkdir oss://bucket/dir1/dir2
`,
}

var specEnglishMkdir = SpecText{
	synopsisText: "Create a oss directory whose object name has the suffix character '/'",

	paramText: "dir_name [options]",

	syntaxText: ` 
	ossutil mkdir oss://bucket/dir_name
`,
	detailHelpText: ` 
    1) If the input parameter does not end with the suffix character '/', the tool will automatically add
    2) If the directory already exists, you will be prompted with an error.
    3) If the input parameter contains multiple levels of directories, only the last directory will be created.
`,
	sampleText: ` 
    1) create a diretory
       ossutil mkdir oss://bucket/dir

    2) create a multi-level directory
       ossutil mkdir oss://bucket/dir1/dir2
`,
}

type mkOptionType struct {
	encodingType string
}

type MkdirCommand struct {
	command  Command
	mkOption mkOptionType
}

var mkdirCommand = MkdirCommand{
	command: Command{
		name:        "mkdir",
		nameAlias:   []string{"mkdir"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseMkdir,
		specEnglish: specEnglishMkdir,
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
func (mkc *MkdirCommand) formatHelpForWhole() string {
	return mkc.command.formatHelpForWhole()
}

func (mkc *MkdirCommand) formatIndependHelp() string {
	return mkc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (mkc *MkdirCommand) Init(args []string, options OptionMapType) error {
	return mkc.command.Init(args, options, mkc)
}

// RunCommand simulate inheritance, and polymorphism
func (mkc *MkdirCommand) RunCommand() error {
	mkc.mkOption.encodingType, _ = GetString(OptionEncodingType, mkc.command.options)

	dirUrL, err := StorageURLFromString(mkc.command.args[0], mkc.mkOption.encodingType)
	if err != nil {
		return fmt.Errorf("StorageURLFromString error")
	}

	if !dirUrL.IsCloudURL() {
		return fmt.Errorf("parameter is not a cloud url,url is %s", dirUrL.ToString())
	}

	cloudUrl := dirUrL.(CloudURL)

	if cloudUrl.bucket == "" {
		return fmt.Errorf("bucket name is empty,url is %s", cloudUrl.ToString())
	}

	if cloudUrl.object == "" {
		return fmt.Errorf("object name is empty,url is %s", cloudUrl.ToString())
	}

	if !strings.HasSuffix(cloudUrl.object, "/") {
		cloudUrl.object += "/"
	}

	return mkc.MkBucketDir(cloudUrl)
}

func (mkc *MkdirCommand) MkBucketDir(dirUrl CloudURL) error {
	bucket, err := mkc.command.ossBucket(dirUrl.bucket)
	if err != nil {
		return err
	}

	bExist, err := bucket.IsObjectExist(dirUrl.object)
	if err != nil {
		return err
	}

	if bExist {
		return fmt.Errorf("%s already exists", dirUrl.object)
	}

	return bucket.PutObject(dirUrl.object, strings.NewReader(""))
}
