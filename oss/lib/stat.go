package lib

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseStat = SpecText{

	synopsisText: "显示bucket或者object的描述信息",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil stat oss://bucket[/object] [--encoding-type url] [--version-id versionId] [--payer requester] [-c file] 
`,

	detailHelpText: ` 
    该命令获取指定bucket或者object的描述信息。通过set-meta命令设置的object元信息，可以通过
    该命令查看。

用法：

    该命令有两种用法：

    1) ossutil stat oss://bucket [--encoding-type url]
        ossutil显示指定bucket的信息，包括创建时间，location，访问的外网域名，内网域名，拥
    有者，acl信息。

    2) ossutil stat oss://bucket/object [--encoding-type url] [--version-id versionId]
        ossutil显示指定object的元信息，包括文件大小，最新更新时间，etag，文件类型，acl，文
    件的自定义meta等信息。
`,

	sampleText: ` 
    ossutil stat oss://bucket1
    ossutil stat oss://bucket1/object  
    ossutil stat oss://bucket1/object --version-id versionId
    ossutil stat oss://bucket1/%e4%b8%ad%e6%96%87 --encoding-type url
    ossutil stat oss://bucket1/object --payer requester
`,
}

var specEnglishStat = SpecText{

	synopsisText: "Display meta information of bucket or objects",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil stat oss://bucket[/object] [--encoding-type url]  [--version-id versionId] [--payer requester] [-c file] 
`,

	detailHelpText: ` 
    The command display the meta information of bucket or objects. The object meta information 
    setted through set-meta command, can be check by the command.

Usage：

    There are three usages:    

    1) ossutil stat oss://bucket [--encoding-type url]
        ossutil display bucket meta info, include creation date, location, extranet endpoint, 
    intranet endpoint, Owner and acl info.

    2) ossutil stat oss://bucket/object [--encoding-type url] [--version-id versionId]
        ossutil display object meta info, include file size, last modify time, etag, content-type, 
    user meta etc.
`,

	sampleText: ` 
    ossutil stat oss://bucket1
    ossutil stat oss://bucket1/object
    ossutil stat oss://bucket1/object --version-id versionId  
    ossutil stat oss://bucket1/%e4%b8%ad%e6%96%87 --encoding-type url
    ossutil stat oss://bucket1/object --payer requester
`,
}

// StatCommand is the command get bucket's or objects' meta information
type StatCommand struct {
	command       Command
	versionId     string
	commonOptions []oss.Option
}

var statCommand = StatCommand{
	command: Command{
		name:        "stat",
		nameAlias:   []string{"meta", "info"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseStat,
		specEnglish: specEnglishStat,
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
func (sc *StatCommand) formatHelpForWhole() string {
	return sc.command.formatHelpForWhole()
}

func (sc *StatCommand) formatIndependHelp() string {
	return sc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (sc *StatCommand) Init(args []string, options OptionMapType) error {
	return sc.command.Init(args, options, sc)
}

// RunCommand simulate inheritance, and polymorphism
func (sc *StatCommand) RunCommand() error {
	sc.versionId, _ = GetString(OptionVersionId, sc.command.options)
	encodingType, _ := GetString(OptionEncodingType, sc.command.options)
	cloudURL, err := CloudURLFromString(sc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", sc.command.args[0])
	}

	payer, _ := GetString(OptionRequestPayer, sc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		sc.commonOptions = append(sc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	bucket, err := sc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	if cloudURL.object == "" {
		return sc.bucketStat(bucket, cloudURL)
	}
	return sc.objectStat(bucket, cloudURL)
}

func (sc *StatCommand) bucketStat(bucket *oss.Bucket, cloudURL CloudURL) error {
	// TODO: go sdk should implement GetBucketInfo
	gbar, err := sc.ossGetBucketStatRetry(bucket)
	if err != nil {
		return err
	}

	fmt.Printf("%-22s: %s\n", StatName, gbar.BucketInfo.Name)
	fmt.Printf("%-22s: %s\n", StatLocation, gbar.BucketInfo.Location)
	fmt.Printf("%-22s: %s\n", StatCreationDate, utcToLocalTime(gbar.BucketInfo.CreationDate))
	fmt.Printf("%-22s: %s\n", StatExtranetEndpoint, gbar.BucketInfo.ExtranetEndpoint)
	fmt.Printf("%-22s: %s\n", StatIntranetEndpoint, gbar.BucketInfo.IntranetEndpoint)
	fmt.Printf("%-22s: %s\n", StatACL, gbar.BucketInfo.ACL)
	fmt.Printf("%-22s: %s\n", StatOwner, gbar.BucketInfo.Owner.ID)
	fmt.Printf("%-22s: %s\n", StatStorageClass, gbar.BucketInfo.StorageClass)
	if len(gbar.BucketInfo.RedundancyType) > 0 {
		fmt.Printf("%-22s: %s\n", StatRedundancyType, gbar.BucketInfo.RedundancyType)
	}
	if len(gbar.BucketInfo.SseRule.SSEAlgorithm) > 0 {
		fmt.Printf("%-22s: %s\n", StatSSEAlgorithm, gbar.BucketInfo.SseRule.SSEAlgorithm)
	}
	if len(gbar.BucketInfo.SseRule.KMSMasterKeyID) > 0 {
		fmt.Printf("%-22s: %s\n", StatKMSMasterKeyID, gbar.BucketInfo.SseRule.KMSMasterKeyID)
	}
	if len(gbar.BucketInfo.SseRule.KMSDataEncryption) > 0 {
		fmt.Printf("%-22s: %s\n", StatKMSDataEncryption, gbar.BucketInfo.SseRule.KMSDataEncryption)
	}
	fmt.Printf("%-22s: %s\n", StatTransferAcceleration, gbar.BucketInfo.TransferAcceleration)
	fmt.Printf("%-22s: %s\n", StatCrossRegionReplication, gbar.BucketInfo.CrossRegionReplication)
	if len(gbar.BucketInfo.AccessMonitor) > 0 {
		fmt.Printf("%-22s: %s\n", StatAccessMonitor, gbar.BucketInfo.AccessMonitor)
	}

	return nil
}

func (sc *StatCommand) ossGetBucketStatRetry(bucket *oss.Bucket) (oss.GetBucketInfoResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, sc.command.options)
	for i := 1; ; i++ {
		gbar, err := bucket.Client.GetBucketInfo(bucket.BucketName, sc.commonOptions...)
		if err == nil {
			return gbar, err
		}
		if int64(i) >= retryTimes {
			return gbar, BucketError{err, bucket.BucketName}
		}
	}
}

func (sc *StatCommand) objectStat(bucket *oss.Bucket, cloudURL CloudURL) error {
	// acl info
	goar, err := sc.ossGetObjectACLRetry(bucket, cloudURL.object)
	if err != nil {
		return err
	}

	// normal info
	statOptions := []oss.Option{}
	if len(sc.versionId) > 0 {
		statOptions = append(statOptions, oss.VersionId(sc.versionId))
	}
	statOptions = append(statOptions, sc.commonOptions...)

	props, err := sc.command.ossGetObjectStatRetry(bucket, cloudURL.object, statOptions...)
	if err != nil {
		return err
	}

	sortNames := []string{}
	attrMap := map[string]string{}
	maxNameLen := 0
	for name := range props {
		if len(name) > maxNameLen {
			maxNameLen = len(name)
		}

		ln := strings.ToLower(name)
		if ln != strings.ToLower(oss.HTTPHeaderDate) &&
			ln != strings.ToLower(oss.HTTPHeaderOssRequestID) &&
			ln != strings.ToLower(oss.HTTPHeaderServer) &&
			ln != "x-oss-server-time" &&
			ln != "connection" {
			sortNames = append(sortNames, name)
			attrMap[name] = props.Get(name)
		}
	}

	sortNames = append(sortNames, "Owner")
	sortNames = append(sortNames, "ACL")
	attrMap[StatOwner] = goar.Owner.ID
	attrMap[StatACL] = goar.ACL
	if lm, err := time.Parse(http.TimeFormat, attrMap[StatLastModified]); err == nil {
		attrMap[StatLastModified] = fmt.Sprintf("%s", utcToLocalTime(lm.UTC()))
	}

	sort.Strings(sortNames)
	for _, name := range sortNames {
		if strings.ToLower(name) != "etag" {
			fmt.Printf("%-[1]*s: %s\n", maxNameLen+2, name, attrMap[name])
		} else {
			fmt.Printf("%-[1]*s: %s\n", maxNameLen+2, name, strings.Trim(attrMap[name], "\""))
		}
	}
	return nil
}

func (sc *StatCommand) ossGetObjectACLRetry(bucket *oss.Bucket, object string) (oss.GetObjectACLResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, sc.command.options)
	aclOptions := []oss.Option{}
	if len(sc.versionId) > 0 {
		aclOptions = append(aclOptions, oss.VersionId(sc.versionId))
	}
	aclOptions = append(aclOptions, sc.commonOptions...)

	for i := 1; ; i++ {
		goar, err := bucket.GetObjectACL(object, aclOptions...)
		if err == nil {
			return goar, err
		}
		if int64(i) >= retryTimes {
			return goar, ObjectError{err, bucket.BucketName, object}
		}
	}
}
