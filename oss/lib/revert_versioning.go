package lib

import (
	"fmt"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseRevert = SpecText{
	synopsisText: "将object从删除状态恢复成最近的多版本状态",

	paramText: "cloud_url [options]",

	syntaxText: ` 
	ossutil revert-versioning oss://bucket[/prefix] [--encoding-type encodeType] [-r] [--start-time startTime] [--end-time endTime]  [--include include-pattern] [--exclude exclude-pattern] [--payer requester]
`,

	detailHelpText: ` 
    该命令通过删除最新的删除标记,使object从删除状态恢复成最近的多版本状态
    
--recursive选项
    如果输入--recursive或者-r,表示批量操作匹配prefix的所有objects, 否则只操作key为prefix的单个object

--start-time
    时间戳, 既从1970年1月1日(UTC/GMT的午夜)开始所经过的秒数
    如果输入这个选项, object的删除时间小于该时间戳将被忽略

--end-time
    时间戳, 既从1970年1月1日(UTC/GMT的午夜)开始所经过的秒数
    如果输入这个选项, object的删除时间大于该时间戳将被忽略
  
用法：

    该命令只有一种用法：

    1) ossutil revert-versioning oss://bucket[/prefix] [--encoding-type encodeType] [-r] [--start-time startTime] [--end-time endTime]  [--include include-pattern] [--exclude exclude-pattern] [--payer requester]
       恢复bucket下面满足前缀为prefix的object为多版本状态
`,

	sampleText: ` 
	1) 恢复整个bucket的处于删除状态的objects为最近的多版本状态
       ossutil revert-versioning oss://bucket -r
    
    2) 恢复单个处于删除状态的object为最近多版本状态
       ossutil revert-versioning oss://bucket/object
    
    3) 恢复处于删除状态的objects为最近的多版本状态, key的后缀满足输入的过滤条件
       ossutil revert-versioning oss://bucket/prefix -r --include *.jpg --exclude *.txt
    
    4) 恢复处于删除状态的objects为最近的多版本状态, object的最后删除时间必须在输入范围内
       起始时间为北京时间2020/6/16 16:22:58, 结束时间为北京时间2020/6/16 16:39:38
       ossutil revert-versioning oss://bucket/prefix -r --start-time 1592295778 --end-time 1592296778
    
    5) 访问者付费模式
       ossutil revert-versioning oss://bucket/prefix -r --payer requester
`,
}

var specEnglishRevert = SpecText{
	synopsisText: "Revert the deleted object to the latest versioning state",

	paramText: "cloud_url [options]",

	syntaxText: ` 
	ossutil revert-versioning oss://bucket[/prefix] [--encoding-type encodeType] [-r] [--start-time startTime] [--end-time endTime]  [--include include-pattern] [--exclude exclude-pattern] [--payer requester]
`,

	detailHelpText: ` 
	This command revert the object from the deleted state to the latest versioning state by deleting the latest delete mark

Usages：

    There is only one usage for this command:

    1) ossutil revert-versioning oss://bucket[/prefix] [--encoding-type encodeType] [-r] [--start-time startTime] [--end-time endTime]  [--include include-pattern] [--exclude exclude-pattern] [--payer requester]
       Revert the bucket's objects whose prefix are "prefix" to the versioning state
`,

	sampleText: ` 
	1) Revert the bucket's deleted objects to the latest versioning state
       ossutil revert-versioning oss://bucket -r
    
    2) Revert a single deleted object to the latest versioning state
       ossutil revert-versioning oss://bucket/object
    
    3) Revert deleted objects to the latest versioning state, the key suffix meets the input filter conditions
       ossutil revert-versioning oss://bucket/prefix -r --include *.jpg --exclude *.txt
    
    4) Revert deleted objects to the latest versioning state, the last deletion time of objects must be within the input range
       The start time is Beijing time 2020/6/16 16:22:58, and the end time is Beijing time 2020/6/16 16:39:38
       ossutil revert-versioning oss://bucket/prefix -r --start-time 1592295778 --end-time 1592296778
    
    5) Use requester to pay mode
       ossutil revert-versioning oss://bucket/prefix -r --payer requester
`,
}

type revertOptionType struct {
	bucketName  string
	object      string
	startTime   int64
	endTime     int64
	payer       string
	filters     []filterOptionType
	options     []oss.Option
	recursive   bool
	revertCount int64
}

type RevertCommand struct {
	command      Command
	revertOption revertOptionType
}

var revertCommand = RevertCommand{
	command: Command{
		name:        "revert-versioning",
		nameAlias:   []string{"revert-versioning"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseRevert,
		specEnglish: specEnglishRevert,
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
			OptionRecursion,
			OptionRequestPayer,
			OptionStartTime,
			OptionEndTime,
			OptionInclude,
			OptionExclude,
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
func (revert *RevertCommand) formatHelpForWhole() string {
	return revert.command.formatHelpForWhole()
}

func (revert *RevertCommand) formatIndependHelp() string {
	return revert.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (revert *RevertCommand) Init(args []string, options OptionMapType) error {
	return revert.command.Init(args, options, revert)
}

// RunCommand simulate inheritance, and polymorphism
func (revert *RevertCommand) RunCommand() error {
	encodingType, _ := GetString(OptionEncodingType, revert.command.options)
	srcBucketUrL, err := GetCloudUrl(revert.command.args[0], encodingType)
	if err != nil {
		return err
	}
	revert.revertOption.bucketName = srcBucketUrL.bucket
	revert.revertOption.object = srcBucketUrL.object
	revert.revertOption.recursive, _ = GetBool(OptionRecursion, revert.command.options)
	if !revert.revertOption.recursive && revert.revertOption.object == "" {
		return fmt.Errorf("please input object key when option recursive is false")
	}

	revert.revertOption.startTime, _ = GetInt(OptionStartTime, revert.command.options)
	revert.revertOption.endTime, _ = GetInt(OptionEndTime, revert.command.options)
	if revert.revertOption.endTime > 0 && revert.revertOption.startTime > revert.revertOption.endTime {
		return fmt.Errorf("start time %d is larger than end time %d", revert.revertOption.startTime, revert.revertOption.endTime)
	}

	revert.revertOption.payer, _ = GetString(OptionRequestPayer, revert.command.options)
	if revert.revertOption.payer != "" {
		if strings.ToLower(revert.revertOption.payer) != strings.ToLower(string(oss.Requester)) &&
			strings.ToLower(revert.revertOption.payer) != strings.ToLower(string(oss.BucketOwner)) {
			return fmt.Errorf("option payer value must be %s or %s",
				strings.ToLower(string(oss.Requester)), strings.ToLower(string(oss.BucketOwner)))
		}
		revert.revertOption.options = append(revert.revertOption.options, oss.RequestPayer(oss.PayerType(revert.revertOption.payer)))
	}

	var res bool
	res, revert.revertOption.filters = getFilter(os.Args)
	if !res {
		return fmt.Errorf("--include or --exclude does not support format containing dir info")
	}

	bucket, err := revert.command.ossBucket(revert.revertOption.bucketName)
	if err != nil {
		return err
	}
	return revert.revertObjects(bucket)
}

func (revert *RevertCommand) revertObjects(bucket *oss.Bucket) error {
	pre := oss.Prefix(revert.revertOption.object)
	keyMarker := oss.KeyMarker("")
	versionIdMarker := oss.VersionIdMarker("")
	listOptions := []oss.Option{pre, keyMarker, versionIdMarker, oss.MaxKeys(1000)}
	if revert.revertOption.payer != "" {
		listOptions = append(listOptions, oss.RequestPayer(oss.PayerType(revert.revertOption.payer)))
	}

	bStopped := false
	batchCount := 0
	for {
		if bStopped {
			break
		}
		batchCount++
		lor, err := bucket.ListObjectVersions(listOptions...)
		if err != nil {
			return err
		}
		var objectVersions []oss.DeleteObject
		for _, deleteMarker := range lor.ObjectDeleteMarkers {
			if !revert.revertOption.recursive && deleteMarker.Key != revert.revertOption.object {
				bStopped = true
				break
			}

			if deleteMarker.IsLatest && revert.filterDeleteMarker(&deleteMarker) {
				objectVersions = append(objectVersions, oss.DeleteObject{
					Key:       deleteMarker.Key,
					VersionId: deleteMarker.VersionId,
				})
			}
		}

		if len(objectVersions) > 0 {
			deleteOptions := append(revert.revertOption.options, oss.DeleteObjectsQuiet(true))
			delRes, err := bucket.DeleteObjectVersions(objectVersions, deleteOptions...)
			if err != nil {
				return err
			}

			if len(delRes.DeletedObjectsDetail) > 0 {
				fmt.Printf("\n")
				for _, object := range delRes.DeletedObjectsDetail {
					fmt.Printf("delete deleteMarker failure, key:%s,version:%s\n", object.Key, object.VersionId)
				}
				return fmt.Errorf("delete deleteMarker failure")
			}
			revert.revertOption.revertCount += int64(len(objectVersions))
			for _, object := range objectVersions {
				LogInfo("revert %s %s\n", object.Key, object.VersionId)
			}
		}
		keyMarker = oss.KeyMarker(lor.NextKeyMarker)
		versionIdMarker := oss.VersionIdMarker(lor.NextVersionIdMarker)
		listOptions = []oss.Option{pre, keyMarker, versionIdMarker, oss.MaxKeys(1000)}
		if revert.revertOption.payer != "" {
			listOptions = append(listOptions, oss.RequestPayer(oss.PayerType(revert.revertOption.payer)))
		}
		fmt.Printf("\rrevert versioning object count is %d, batch list count is %d", revert.revertOption.revertCount, batchCount)
		if !lor.IsTruncated {
			break
		}
	}
	fmt.Printf("\n")
	return nil
}

func (revert *RevertCommand) filterDeleteMarker(deleteMarker *oss.ObjectDeleteMarkerProperties) bool {
	if !doesSingleObjectMatchPatterns(deleteMarker.Key, revert.revertOption.filters) {
		return false
	}

	if (revert.revertOption.startTime > 0 && deleteMarker.LastModified.Unix() < revert.revertOption.startTime) ||
		(revert.revertOption.endTime > 0 && deleteMarker.LastModified.Unix() > revert.revertOption.endTime) {
		return false
	}
	return true
}
