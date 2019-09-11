package lib

import (
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"sync/atomic"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseDu = SpecText{
	synopsisText: "获取bucket或者指定前缀(目录)所占的存储空间大小",

	paramText: "bucket_url [options]",

	syntaxText: ` 
	ossutil du oss://bucket[/prefix] [options]
`,

	detailHelpText: ` 
	该命令会获取bucket或者指定前缀(目录)所占的存储空间大小,包括未完成上传object的块大小
  
用法：

    该命令只有一种用法：

    1) ossutil du oss://bucket[/prefix] [options]
      查询bucket或者指定前缀(目录)所占存储空间大小
`,

	sampleText: ` 
	1) 查询bucket占用存储空间大小
       ossutil du oss://bucket
    
    2) 查询指定前缀(目录)占用存储空间大小
       ossutil du oss://bucket/prefix
`,
}

var specEnglishDu = SpecText{
	synopsisText: "Get the bucket or the specified prefix(directory) storage size",

	paramText: "bucket_url [options]",

	syntaxText: ` 
	ossutil du oss://bucket[/prefix] [options]
`,

	detailHelpText: ` 
	This command gets the bucket or the specified prefix(directory) storage size,including uncompleted part size

Usages：

    There is only one usage for this command:

    1) ossutil du oss://bucket[/prefix] [options]
       Gets the bucket or the specified prefix(directory) storage size
`,

	sampleText: ` 
	1) get the bucket storage size
       ossutil du oss://bucket
    
    2) get the prefix(directory) stroage size
       ossutil du oss://bucket/prefix
`,
}

type MultiPartObject struct {
	objectName string
	uploadId   string
}

type duSizeOptionType struct {
	bucketName       string
	object           string
	payer            string
	totalObjectCount int64
	sumObjectSize    int64
	totalPartCount   int64
	sumPartSize      int64
}

type DuCommand struct {
	command  Command
	duOption duSizeOptionType
}

var duSizeCommand = DuCommand{
	command: Command{
		name:        "du",
		nameAlias:   []string{"du"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseDu,
		specEnglish: specEnglishDu,
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
			OptionRequestPayer,
		},
	},
}

func (duc *DuCommand) GetCommand() *Command {
	return &duc.command
}

// function for FormatHelper interface
func (duc *DuCommand) formatHelpForWhole() string {
	return duc.command.formatHelpForWhole()
}

func (duc *DuCommand) formatIndependHelp() string {
	return duc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (duc *DuCommand) Init(args []string, options OptionMapType) error {
	return duc.command.Init(args, options, duc)
}

// RunCommand simulate inheritance, and polymorphism
func (duc *DuCommand) RunCommand() error {
	// clear for go tests
	duc.duOption.totalObjectCount = 0
	duc.duOption.sumObjectSize = 0
	duc.duOption.totalPartCount = 0
	duc.duOption.sumPartSize = 0

	encodingType, _ := GetString(OptionEncodingType, duc.command.options)
	srcBucketUrL, err := GetCloudUrl(duc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	payer, _ := GetString(OptionRequestPayer, duc.command.options)
	if payer != "" {
		if strings.ToLower(payer) != strings.ToLower(string(oss.Requester)) &&
			strings.ToLower(payer) != strings.ToLower(string(oss.BucketOwner)) {
			return fmt.Errorf("option payer value must be %s or %s",
				strings.ToLower(string(oss.Requester)), strings.ToLower(string(oss.BucketOwner)))
		}
	}

	duc.duOption.bucketName = srcBucketUrL.bucket
	duc.duOption.object = srcBucketUrL.object
	duc.duOption.payer = payer

	bucket, err := duc.command.ossBucket(duc.duOption.bucketName)
	if err != nil {
		return err
	}

	// first:get all object size
	err = duc.getAllObjectSize(bucket)
	if err != nil {
		return err
	}

	fmt.Printf("\robject count:%d\tobject sum size:%d\n", duc.duOption.totalObjectCount, duc.duOption.sumObjectSize)

	//second:get all part size
	err = duc.GetAllPartSize(bucket)
	if err != nil {
		return err
	}
	fmt.Printf("\rpart count:%d\tpart sum size:%d\n", atomic.LoadInt64(&duc.duOption.totalPartCount), atomic.LoadInt64(&duc.duOption.sumPartSize))

	fmt.Printf("total du size(byte):%d\n", duc.duOption.sumObjectSize+duc.duOption.sumPartSize)
	return nil
}

func (duc *DuCommand) getAllObjectSize(bucket *oss.Bucket) error {
	pre := oss.Prefix(duc.duOption.object)
	marker := oss.Marker("")
	listOptions := []oss.Option{pre, marker, oss.MaxKeys(1000)}
	if duc.duOption.payer != "" {
		listOptions = append(listOptions, oss.RequestPayer(oss.PayerType(duc.duOption.payer)))
	}

	for i := 1; ; i++ {
		lor, err := duc.command.ossListObjectsRetry(bucket, listOptions...)
		if err != nil {
			return err
		}

		duc.duOption.totalObjectCount += int64(len(lor.Objects))
		for _, object := range lor.Objects {
			duc.duOption.sumObjectSize += object.Size
		}

		fmt.Printf("\robject count:%d\tobject sum size:%d", duc.duOption.totalObjectCount, duc.duOption.sumObjectSize)

		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		listOptions = []oss.Option{pre, marker, oss.MaxKeys(1000)}
		if duc.duOption.payer != "" {
			listOptions = append(listOptions, oss.RequestPayer(oss.PayerType(duc.duOption.payer)))
		}

		if !lor.IsTruncated {
			break
		}
	}
	return nil
}

func (duc *DuCommand) GetAllPartSize(bucket *oss.Bucket) error {
	routineCount := runtime.NumCPU()
	chObjects := make(chan MultiPartObject, ChannelBuf)
	chError := make(chan error, routineCount)
	chListError := make(chan error, 1)

	go duc.uploadIdProducer(bucket, chObjects, chListError)
	for i := 0; i < routineCount; i++ {
		go duc.uploadIdConsumer(bucket, chObjects, chError)
	}
	return duc.waitRoutinueComplete(chError, chListError, routineCount)
}

func (duc *DuCommand) uploadIdConsumer(bucket *oss.Bucket, chObjects chan MultiPartObject, chError chan error) error {
	for object := range chObjects {
		err := duc.statPartSize(bucket, object)
		if err != nil {
			chError <- err
			return err
		}
	}
	chError <- nil
	return nil
}

func (duc *DuCommand) statPartSize(bucket *oss.Bucket, object MultiPartObject) error {
	var imur oss.InitiateMultipartUploadResult
	imur.Bucket = duc.duOption.bucketName
	imur.Key = object.objectName
	imur.UploadID = object.uploadId
	partNumberMarker := 0
	for i := 0; ; i++ {
		lpOptions := []oss.Option{}
		lpOptions = append(lpOptions, oss.MaxParts(1000))
		lpOptions = append(lpOptions, oss.PartNumberMarker(partNumberMarker))
		if duc.duOption.payer != "" {
			lpOptions = append(lpOptions, oss.RequestPayer(oss.PayerType(duc.duOption.payer)))
		}

		lpRes, err := bucket.ListUploadedParts(imur, lpOptions...)
		if err != nil {
			return err
		} else {
			atomic.AddInt64(&duc.duOption.totalPartCount, int64(len(lpRes.UploadedParts)))
		}

		for _, v := range lpRes.UploadedParts {
			atomic.AddInt64(&duc.duOption.sumPartSize, int64(v.Size))
		}

		fmt.Printf("\rpart count:%d\tpart sum size:%d", atomic.LoadInt64(&duc.duOption.totalPartCount), atomic.LoadInt64(&duc.duOption.sumPartSize))

		if lpRes.IsTruncated {
			partNumberMarker, _ = strconv.Atoi(lpRes.NextPartNumberMarker)
		} else {
			break
		}
	}

	return nil
}

func (duc *DuCommand) uploadIdProducer(bucket *oss.Bucket, chObjects chan MultiPartObject, chListError chan error) error {
	prefix := duc.duOption.object
	keyMarker := ""
	uploadIdMarker := ""
	for i := 0; ; i++ {
		lpOptions := []oss.Option{}
		lpOptions = append(lpOptions, oss.MaxParts(1000))
		lpOptions = append(lpOptions, oss.Prefix(prefix))
		lpOptions = append(lpOptions, oss.KeyMarker(keyMarker))
		lpOptions = append(lpOptions, oss.UploadIDMarker(uploadIdMarker))
		if duc.duOption.payer != "" {
			lpOptions = append(lpOptions, oss.RequestPayer(oss.PayerType(duc.duOption.payer)))
		}

		lpRes, err := bucket.ListMultipartUploads(lpOptions...)
		if err != nil {
			chListError <- err
			return err
		}

		for _, v := range lpRes.Uploads {
			var object MultiPartObject
			object.objectName = v.Key
			object.uploadId = v.UploadID
			chObjects <- object

		}

		if lpRes.IsTruncated {
			keyMarker = lpRes.NextKeyMarker
			uploadIdMarker = lpRes.NextUploadIDMarker
		} else {
			break
		}
	}

	defer close(chObjects)
	chListError <- nil
	return nil
}

func (duc *DuCommand) waitRoutinueComplete(chError, chListError <-chan error, routineCount int) error {
	completed := 0
	for completed <= routineCount {
		select {
		case err := <-chListError:
			if err != nil {
				return err
			}
			completed++
		case err := <-chError:
			if err != nil {
				return err
			}
			completed++
		}
	}
	return nil
}
