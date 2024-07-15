package lib

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseReadSymlink = SpecText{

	synopsisText: "读取符号链接文件的描述信息",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil read-symlink oss://bucket/object [--encoding-type url] [--version-id versionId] [--payer requester] [-c file] 
`,

	detailHelpText: ` 
    该命令获取指定符号链接object的描述信息，此操作要求用户对该符号链接有读权限。
    如果需要读取符合链接object的文件内容，请使用cp命令下载该object。
    
    返回的项中X-Oss-Symlink-Target表示符号链接的目标文件。

    如果object并非符号链接文件，该操作返回错误：NotSymlink。

    更多信息见官网API文档：https://help.aliyun.com/document_detail/45146.html?spm=5176.doc31968.6.871.24y1VX

用法：

    ossutil read-symlink oss://bucket/symlink-object [--version-id versionId] [--payer requester]
`,

	sampleText: ` 
    ossutil read-symlink oss://bucket1/object1 
        Etag                    : 455E20DBFFF1D588B67D092C46B16DB6
        Last-Modified           : 2017-04-17 14:49:42 +0800 CST
        X-Oss-Symlink-Target    : a
    
    ossutil read-symlink oss://bucket1/object --version-id versionId

    ossutil read-symlink oss://bucket1/object --payer requester
`,
}

var specEnglishReadSymlink = SpecText{

	synopsisText: "Display meta information of symlink object",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil read-symlink oss://bucket/object [--encoding-type url] [--payer requester] [-c file]
`,

	detailHelpText: ` 
    The command display the meta information of symlink object. The operation 
    requires that the user have read permission of the symlink object. If you 
    want to get the file data of symlink object, please use cp command to download 
    the symlink object. 

    The item X-Oss-Symlink-Target shows the target object of the symlink object.

    If the object is not symlink object, ossutil return error: NotSymlink.

    More information about symlink see: https://help.aliyun.com/document_detail/45146.html?spm=5176.doc31968.6.871.24y1VX 

Usage:

    ossutil read-symlink oss://bucket/symlink-object [--version-id versionId] [--payer requester]
`,

	sampleText: ` 
    ossutil read-symlink oss://bucket1/object1 
        Etag                    : 455E20DBFFF1D588B67D092C46B16DB6
        Last-Modified           : 2017-04-17 14:49:42 +0800 CST
        X-Oss-Symlink-Target    : a
    
    ossutil read-symlink oss://bucket1/object --version-id versionId

    ossutil read-symlink oss://bucket1/object --payer requester
`,
}

// ReadSymlinkCommand is the command list buckets or objects
type ReadSymlinkCommand struct {
	command       Command
	commonOptions []oss.Option
}

var readSymlinkCommand = ReadSymlinkCommand{
	command: Command{
		name:        "read-symlink",
		nameAlias:   []string{},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseReadSymlink,
		specEnglish: specEnglishReadSymlink,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionEncodingType,
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionRetryTimes,
			OptionLogLevel,
			OptionVersionId,
			OptionRequestPayer,
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
func (rc *ReadSymlinkCommand) formatHelpForWhole() string {
	return rc.command.formatHelpForWhole()
}

func (rc *ReadSymlinkCommand) formatIndependHelp() string {
	return rc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (rc *ReadSymlinkCommand) Init(args []string, options OptionMapType) error {
	return rc.command.Init(args, options, rc)
}

// RunCommand simulate inheritance, and polymorphism
func (rc *ReadSymlinkCommand) RunCommand() error {
	encodingType, _ := GetString(OptionEncodingType, rc.command.options)
	cloudURL, err := ObjectURLFromString(rc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	versionId, _ := GetString(OptionVersionId, rc.command.options)
	symlinkOptions := []oss.Option{}
	if len(versionId) > 0 {
		symlinkOptions = append(symlinkOptions, oss.VersionId(versionId))
	}

	payer, _ := GetString(OptionRequestPayer, rc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		rc.commonOptions = append(rc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	symlinkOptions = append(symlinkOptions, rc.commonOptions...)
	bucket, err := rc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	return rc.linkStat(bucket, cloudURL, symlinkOptions...)
}

func (rc *ReadSymlinkCommand) linkStat(bucket *oss.Bucket, cloudURL CloudURL, options ...oss.Option) error {
	// normal info
	props, err := rc.ossGetSymlinkRetry(bucket, cloudURL.object, options...)
	if err != nil {
		return err
	}

	sortNames := []string{}
	attrMap := map[string]string{}
	for name := range props {
		ln := strings.ToLower(name)
		if ln != strings.ToLower(oss.HTTPHeaderDate) &&
			ln != strings.ToLower(oss.HTTPHeaderOssRequestID) &&
			ln != strings.ToLower(oss.HTTPHeaderServer) &&
			ln != strings.ToLower(oss.HTTPHeaderContentLength) &&
			ln != "x-oss-server-time" &&
			ln != "connection" {
			sortNames = append(sortNames, name)
			attrMap[name] = props.Get(name)
		}
	}

	if lm, err := time.Parse(http.TimeFormat, attrMap[StatLastModified]); err == nil {
		attrMap[StatLastModified] = fmt.Sprintf("%s", utcToLocalTime(lm.UTC()))
	}

	sort.Strings(sortNames)

	for _, name := range sortNames {
		if strings.ToLower(name) != "etag" {
			fmt.Printf("%-24s: %s\n", name, attrMap[name])
		} else {
			fmt.Printf("%-24s: %s\n", name, strings.Trim(attrMap[name], "\""))
		}
	}
	return nil
}

func (rc *ReadSymlinkCommand) ossGetSymlinkRetry(bucket *oss.Bucket, symlinkObject string, options ...oss.Option) (http.Header, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		props, err := bucket.GetSymlink(symlinkObject, options...)
		if err == nil {
			return props, err
		}
		if int64(i) >= retryTimes {
			return props, ObjectError{err, bucket.BucketName, symlinkObject}
		}
	}
}
