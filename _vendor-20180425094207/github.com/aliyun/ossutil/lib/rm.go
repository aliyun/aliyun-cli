package lib

import (
	"fmt"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type uploadIdInfoType struct {
	key      string
	uploadId string
}

type removeOptionType struct {
	recursive bool
	force     bool
	typeSet   int64
}

var specChineseRemove = SpecText{

	synopsisText: "删除Bucket或Objects",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil rm oss://bucket[/prefix] [-r] [-b] [-f] [-c file] 
`,

	detailHelpText: ` 
    该命令删除Bucket或objects，在某些情况下可一并删除二者。请小心使用该命令！！
    在删除objects前确定objects可以删除，在删除bucket前确定整个bucket连同其下的所有
    objects都可以删除！

        （1）删除单个object，参考用法1)
        （2）删除bucket，不删除objects，参考用法2)
        （3）删除objects，不删除bucket，参考用法3)
        （4）删除bucket和objects，参考用法4)

        对bucket进行删除，都需要添加--bucket选项。
        如果指定了--force选项，则删除前不会进行询问提示。
        
        结果：显示命令耗时前未报错，则表示成功删除。

    默认情况下，删除object时，不包括以指定object名称进行的未complete的Multipart Upload
    事件。如果用户需要删除指定object名称下的所有未complete的Multipart Upload事件，需要
    指定--multipart选项（ossutil会删除所有匹配的Multipart Upload事件，但不支持删除特定
    的某个Multipart Upload事件）。

    如果要同时删除object和相应的Multipart Upload事件，需要指定--all-type选项。

    注意：删除未complete的Multipart Upload事件可能造成下次上传相同的UploadId失败，由于
    cp命令使用Multipart来进行断点续传，删除未complete的Multipart Upload事件可能造成cp
    命令断点续传失败（报错：NoSuchUpload），这种时候如果想要重新上传整个文件，请删除
    checkpoint目录中相应的文件。

用法：

    该命令有四种用法：

    1) ossutil rm oss://bucket/object [-m] [-a] 
        （删除单个object）
        如果未指定--recursive和--bucket选项，删除指定的单个object，此时请确保cloud_url
    精确指定了待删除的object，ossutil不会进行前缀匹配。无论是否指定--force选项，ossutil
    都不会进行询问提示。如果此时指定了--bucket选项，将会报错，单独删除bucket参考用法4)。
        如果指定了--multipart选项, 删除指定object下未complete的Multipart Upload事件。
        如果指定了--all-type选项, 删除指定object以及其下未complete的Multipart Upload事件。

    2) ossutil rm oss://bucket -b [-f]
        （删除bucket，不删除objects）
        如果指定了--bucket选项，未指定--recursive选项，ossutil删除指定的bucket，但并不去
    删除该bucket下的objects。此时请确保cloud_url精确匹配待删除的bucket，并且指定的bucket
    内容为空，否则会报错。如果指定了--force选项，则删除前不会进行询问提示。

    3) ossutil rm oss://bucket[/prefix] -r [-m] [-a] [-f]
        （删除objects，不删除bucket）
        如果指定了--recursive选项，未指定--bucket选项。则可以进行objects的批量删除。该用
    法查找与指定cloud_url前缀匹配的所有objects（prefix为空代表bucket下的所有objects），删
    除这些objects。由于未指定--bucket选项，则ossutil保留bucket。如果指定了--force选项，则
    删除前不会进行询问提示。
        如果指定了--multipart选项，删除以指定prefix开头的所有object下的未complete的Multipart 
    Upload任务。
        如果指定了--all-type，删除以指定prefix开头的所有object，以及其下的所有未complete
    的Multipart Upload任务。

    4) ossutil rm oss://bucket[/prefix] -r -b [-m] [-a] [-f]
        （删除bucket和objects）
        如果同时指定了--bucket和--recursive选项，ossutil进行批量删除后会尝试去一并删除
    bucket。当用户想要删除某个bucket连同其中的所有objects时，可采用该操作。如果指定了
    --force选项，则删除前不会进行询问提示。
        如果指定了--multipart选项，删除以指定prefix开头的所有object下的未complete的Multipart
    Upload任务。
        如果指定了--all-type, 删除以指定prefix开头的所有object，以及其下的所有未complete
    的Multipart Upload任务。

    
    不支持的用法：
    1) ossutil rm oss://bucket/object -m -b [-f]
        不能尝试删除一个指定object名称下未complete的Multipart Upload任务后紧接着删除该bucket。
    2) ossutil rm oss://bucket/object -a -b [-f]
        不能尝试删除一个指定的object和其下所有未complete的Multipart Upload任务后紧接着删除该bucket。
`,

	sampleText: ` 
    ossutil rm oss://bucket1/obj1
    ossutil rm oss://bucket1/obj1 -m
    ossutil rm oss://bucket1/obj1 -a
    ossutil rm oss://bucket1/objdir -r 
    ossutil rm oss://bucket1/multidir -m -r 
    ossutil rm oss://bucket1/dir -a -r 
    ossutil rm oss://bucket1 -b
    ossutil rm oss://bucket2 -r -b -f
    ossutil rm oss://bucket2 -a -r -b -f
    ossutil rm oss://bucket2/%e4%b8%ad%e6%96%87 --encoding-type url
`,
}

var specEnglishRemove = SpecText{

	synopsisText: "Remove Bucket or Objects",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil rm oss://bucket[/prefix] [-r] [-b] [-f] [-c file]
`,

	detailHelpText: ` 
    The command remove bucket or objects, in some case remove both. Please use the 
    command carefully!! 
    Make sure the objects can be removed before useing the command to remove objects! 
    Make sure the bucket and objects inside can be removed before useing the command 
    to remove bucket!

        (1) Remove single object, see usage 1)
        (2) Remove bucket, don't remove objects inside, see usage 2)
        (3) Batch remove many objects, reserve bucket, see usage 3)
        (4) Remove bucket and objects inside, see usage 4)

        When remove bucket, the --bucket option must be specified.
        If --force option is specified, remove silently without asking user to confirm the 
        operation.  

        Result: if no error displayed before show elasped time, then the target is removed 
        successfully.

    By default, when remove object, ossutil will reserve the uncompleted multipart upload 
    tasks whose object name match the specified cloud_url, if you want to remove those multipart 
    upload tasks, please specify --multipart option. Note: ossutil will remove all the multipart 
    upload tasks of the specified cloud_url, remove a special single multipart upload task 
    is unsupported. 

    If you need to remove object and the multipart upload tasks whose object name match the 
    specified cloud_url meanwhile, please use --all-type option.

    Note: remove the multipart upload tasks uncompleted will cause upload the part fail next 
    time. Because cp command use multipart upload to realize resume upload/download/copy, so 
    remove the multipart upload tasks uncompleted may cause resume upload/download/copy fail 
    the next time(Error: NoSuchUpload). If you want to reupload/download/copy the entire file 
    again, please remove the checkpoint file in checkpoint directory. 

Usage:

    There are four usages:

    1) ossutil rm oss://bucket/object
        (Remove single object)
        If you remove without --recursive and --bucket option, ossutil remove the single 
    object specified in cloud_url. In the usage, please make sure cloud_url exactly specified 
    the object you want to remove, ossutil will not treat object as prefix and remove prefix 
    matching objects. No matter --force is specified or not, ossutil will not show prompt 
    question.
        If --multipart option is specified, ossutil will remove the multipart upload tasks 
    of the specified object.
        If --all-type option is specified, ossutil will remove the specified object along 
    with the multipart upload tasks of the specified object. 

    2) ossutil rm oss://bucket -b [-f]
        (Remove bucket, don't remove objects inside)
        If you remove with --bucket option, without --recursive option, ossutil try to 
    remove the bucket, if the bucket is not empty, error occurs. In the usage, please make 
    sure cloud_url exactly specified the bucket you want to remove, or error occurs. If --force 
    option is specified, ossutil will not show prompt question. 

    3) ossutil rm oss://bucket[/prefix] -r [-m] [-a] [-f]
        (Remove objects, reserve bucket)
        If you remove with --recursive option, without --bucket option, ossutil remove all 
    the objects that prefix-matching the cloud_url you specified(empty prefix means all 
    objects in the bucket), bucket will be reserved because of missing --bucket option.
        If --multipart option is specified, ossutil will remove the multipart upload tasks 
    whose object name start with the specified prefix.
        If --all-type option is specified, ossutil will remove the objects with the specified 
    prefix along with the multipart upload tasks whose object name start with the specified 
    prefix. 

    4) ossutil rm oss://bucket[/prefix] -r -b [-a] [-f] 
        (Remove bucket and objects inside)
        If you remove with both --recursive and --bucket option, after ossutil removed all 
    the prefix-matching objects, ossutil will try to remove the bucket together. If user want 
    to remove bucket and objects inside, the usage is recommended. If --force option is 
    specified, ossutil will not show prompt question. 
        If --multipart option is specified, ossutil will remove the multipart upload tasks 
    whose object name start with the specified prefix.
        If --all-type option is specified, ossutil will remove the objects with the specified 
    prefix along with the multipart upload tasks whose object name start with the specified 
    prefix. 


	Invalid Usage: 
    1) ossutil rm oss://bucket/object -m -b [-f]
		It's invalid to remove the bucket right after remove uncompleted upload tasks of single 
    object.
    2) ossutil rm oss://bucket/object -a -b [-f]
        It's invalid to remove the bucket right after remove the object and uncompleted upload 
    tasks of the single object you specified.
`,

	sampleText: ` 
    ossutil rm oss://bucket1/obj1
    ossutil rm oss://bucket1/obj1 -m
    ossutil rm oss://bucket1/obj1 -a
    ossutil rm oss://bucket1/objdir -r 
    ossutil rm oss://bucket1/multidir -m -r 
    ossutil rm oss://bucket1/dir -a -r 
    ossutil rm oss://bucket1 -b
    ossutil rm oss://bucket2 -r -b -f
    ossutil rm oss://bucket2 -a -r -b -f
    ossutil rm oss://bucket2/%e4%b8%ad%e6%96%87 --encoding-type url
`,
}

// RemoveCommand is the command remove bucket or objects
type RemoveCommand struct {
	monitor  RMMonitor //Put first for atomic op on some fileds
	command  Command
	rmOption removeOptionType
}

var removeCommand = RemoveCommand{
	command: Command{
		name:        "rm",
		nameAlias:   []string{"remove", "delete", "del"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseRemove,
		specEnglish: specEnglishRemove,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionRecursion,
			OptionBucket,
			OptionForce,
			OptionMultipart,
			OptionAllType,
			OptionEncodingType,
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionRetryTimes,
		},
	},
}

// function for FormatHelper interface
func (rc *RemoveCommand) formatHelpForWhole() string {
	return rc.command.formatHelpForWhole()
}

func (rc *RemoveCommand) formatIndependHelp() string {
	return rc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (rc *RemoveCommand) Init(args []string, options OptionMapType) error {
	return rc.command.Init(args, options, rc)
}

// RunCommand simulate inheritance, and polymorphism
func (rc *RemoveCommand) RunCommand() error {
	rc.monitor.init()

	encodingType, _ := GetString(OptionEncodingType, rc.command.options)
	cloudURL, err := CloudURLFromString(rc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", rc.command.args[0])
	}

	bucket, err := rc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	// assembleOption
	if err := rc.assembleOption(cloudURL); err != nil {
		return err
	}

	// confirm remove objects/multiparts/allTypes before statistic
	if !rc.confirmRemoveObject(cloudURL) {
		return nil
	}

	// start progressbar
	go rc.entryStatistic(bucket, cloudURL)

	exitStat := normalExit
	if err = rc.removeEntry(bucket, cloudURL); err != nil {
		exitStat = errExit
	}
	fmt.Printf(rc.monitor.progressBar(true, exitStat))
	return err
}

func (rc *RemoveCommand) assembleOption(cloudURL CloudURL) error {
	rc.rmOption.recursive, _ = GetBool(OptionRecursion, rc.command.options)
	rc.rmOption.force, _ = GetBool(OptionForce, rc.command.options)
	isMultipart, _ := GetBool(OptionMultipart, rc.command.options)
	isAllType, _ := GetBool(OptionAllType, rc.command.options)
	toBucket, _ := GetBool(OptionBucket, rc.command.options)

	if err := rc.checkOption(cloudURL, isMultipart, isAllType, toBucket); err != nil {
		return err
	}

	rc.rmOption.typeSet = 0
	if isMultipart {
		rc.rmOption.typeSet |= multipartType
	}
	if isAllType {
		rc.rmOption.typeSet |= allType
	}
	if toBucket {
		rc.rmOption.typeSet |= bucketType
	}
	if !rc.rmOption.recursive {
		if rc.rmOption.typeSet == 0 {
			rc.rmOption.typeSet |= objectType
		}
	} else {
		if rc.rmOption.typeSet&allType == 0 {
			rc.rmOption.typeSet |= objectType
		}
	}

	return nil
}

func (rc *RemoveCommand) checkOption(cloudURL CloudURL, isMultipart, isAllType, toBucket bool) error {
	if !rc.rmOption.recursive {
		if !toBucket {
			// "rm -a/m" miss object, invalid
			if cloudURL.object == "" {
				return fmt.Errorf("remove bucket, miss --bucket option, if you mean remove object, invalid url: %s, miss object", rc.command.args[0])
			}
		} else {
			if isMultipart || isAllType {
				// "rm -mb" and "rm -ab", with or without object, both invalid
				if cloudURL.object == "" {
					return fmt.Errorf("remove bucket redundant option: --multipart or --all-type, if you mean remove all objects and the bucket meanwhile, you should add --recursive option")
				} else {
					return fmt.Errorf("remove object redundant option: --bucket, remove bucket after remove single object is not supported")
				}
			} else if cloudURL.object != "" {
				// "rm -b" with object, invalid
				return fmt.Errorf("remove bucket invalid url: %s, object not empty, if you mean remove object, you should not use --bucket option", rc.command.args[0])
			}
		}
	}
	return nil
}

func (rc *RemoveCommand) confirmRemoveObject(cloudURL CloudURL) bool {
	if !rc.rmOption.force && rc.rmOption.recursive && rc.rmOption.typeSet&allType != 0 {
		stringList := []string{}
		if rc.rmOption.typeSet&objectType != 0 {
			stringList = append(stringList, "objects")
		}
		if rc.rmOption.typeSet&multipartType != 0 {
			stringList = append(stringList, "multipart uploadIds")
		}
		var val string
		fmt.Printf("Do you really mean to remove recursively %s of %s(y or N)? ", strings.Join(stringList, " and "), rc.command.args[0])
		if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
			fmt.Println("operation is canceled.")
			return false
		}
		return true
	}
	return true
}

func (rc *RemoveCommand) entryStatistic(bucket *oss.Bucket, cloudURL CloudURL) {
	if rc.rmOption.typeSet&objectType != 0 {
		rc.objectStatistic(bucket, cloudURL)
	}
	if rc.rmOption.typeSet&multipartType != 0 {
		rc.multipartUploadsStatistic(bucket, cloudURL)
	}
	rc.monitor.setScanEnd()
}

func (rc *RemoveCommand) objectStatistic(bucket *oss.Bucket, cloudURL CloudURL) error {
	// single object statistic before remove
	if rc.rmOption.recursive {
		return rc.batchObjectStatistic(bucket, cloudURL)
	}
	return nil
}

func (rc *RemoveCommand) touchObject(bucket *oss.Bucket, cloudURL CloudURL) (bool, error) {
	exist, err := rc.ossIsObjectExistRetry(bucket, cloudURL.object)
	if err != nil {
		rc.monitor.setScanError(err)
	} else if exist {
		rc.monitor.updateScanNum(1)
	}
	return exist, err
}

func (rc *RemoveCommand) ossIsObjectExistRetry(bucket *oss.Bucket, object string) (bool, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		exist, err := bucket.IsObjectExist(object)
		if err == nil {
			return exist, err
		}
		if int64(i) >= retryTimes {
			return false, ObjectError{err, bucket.BucketName, object}
		}
	}
}

func (rc *RemoveCommand) batchObjectStatistic(bucket *oss.Bucket, cloudURL CloudURL) error {
	pre := oss.Prefix(cloudURL.object)
	marker := oss.Marker("")
	for {
		lor, err := rc.command.ossListObjectsRetry(bucket, marker, pre)
		if err != nil {
			rc.monitor.setScanError(err)
			return err
		}

		rc.monitor.updateScanNum(int64(len(lor.Objects)))

		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}
	return nil
}

func (rc *RemoveCommand) multipartUploadsStatistic(bucket *oss.Bucket, cloudURL CloudURL) error {
	pre := oss.Prefix(cloudURL.object)
	keyMarker := oss.KeyMarker("")
	uploadIdMarker := oss.UploadIDMarker("")
	for {
		lmr, err := rc.command.ossListMultipartUploadsRetry(bucket, keyMarker, uploadIdMarker, pre)
		if err != nil {
			rc.monitor.setScanError(err)
			return err
		}

		if rc.rmOption.recursive {
			rc.monitor.updateScanUploadIdNum(int64(len(lmr.Uploads)))
		} else {
			for _, uploadId := range lmr.Uploads {
				if uploadId.Key == cloudURL.object {
					rc.monitor.updateScanUploadIdNum(1)
				} else {
					break
				}
			}
		}

		pre = oss.Prefix(lmr.Prefix)
		keyMarker = oss.KeyMarker(lmr.NextKeyMarker)
		uploadIdMarker = oss.UploadIDMarker(lmr.NextUploadIDMarker)
		if !lmr.IsTruncated {
			break
		}
	}
	return nil
}

func (rc *RemoveCommand) removeEntry(bucket *oss.Bucket, cloudURL CloudURL) error {
	// op control whether to show progress bar of the type,
	// but do not control whether to record the ok/error num of the type,
	// so the show and record can be separated.
	rc.monitor.updateOP(rc.rmOption.typeSet & allType)

	if rc.rmOption.typeSet&objectType != 0 {
		if err := rc.removeObjectEntry(bucket, cloudURL); err != nil {
			return err
		}
	}
	if rc.rmOption.typeSet&multipartType != 0 {
		if err := rc.removeMultipartUploadsEntry(bucket, cloudURL); err != nil {
			return err
		}
	}
	if rc.rmOption.typeSet&bucketType != 0 {
		return rc.removeBucket(bucket, cloudURL)
	}
	return nil
}

func (rc *RemoveCommand) removeObjectEntry(bucket *oss.Bucket, cloudURL CloudURL) error {
	if !rc.rmOption.recursive {
		return rc.removeObject(bucket, cloudURL)
	} else {
		return rc.batchDeleteObjects(bucket, cloudURL)
	}
}

func (rc *RemoveCommand) removeObject(bucket *oss.Bucket, cloudURL CloudURL) error {
	// single object statistic before remove to avoid inconsistency
	exist, err := rc.touchObject(bucket, cloudURL)
	if err != nil || exist {
		err = rc.deleteObjectWithMonitor(bucket, cloudURL.object)
		if err != nil && rc.monitor.op == objectType {
			// remove single object error, return error information, do not print progressbar
			rc.monitor.setOP(0)
		}
		return err
	}
	return nil
}

func (rc *RemoveCommand) deleteObjectWithMonitor(bucket *oss.Bucket, object string) error {
	err := rc.ossDeleteObjectRetry(bucket, object)
	if err == nil {
		rc.updateObjectMonitor(1, 0)
	} else {
		rc.updateObjectMonitor(0, 1)
	}
	return err
}

func (rc *RemoveCommand) ossDeleteObjectRetry(bucket *oss.Bucket, object string) error {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		err := bucket.DeleteObject(object)
		if err == nil {
			return err
		}
		if int64(i) >= retryTimes {
			return ObjectError{err, bucket.BucketName, object}
		}
	}
}

func (rc *RemoveCommand) updateObjectMonitor(okNum, errNum int64) {
	rc.monitor.updateObjectNum(okNum)
	rc.monitor.updateErrObjectNum(errNum)
	fmt.Printf(rc.monitor.progressBar(false, normalExit))
}

func (rc *RemoveCommand) batchDeleteObjects(bucket *oss.Bucket, cloudURL CloudURL) error {
	// list objects
	pre := oss.Prefix(cloudURL.object)
	marker := oss.Marker("")
	for {
		lor, err := rc.command.ossListObjectsRetry(bucket, marker, pre)
		if err != nil {
			return err
		}

		// batch delete
		delNum, err := rc.ossBatchDeleteObjectsRetry(bucket, rc.getObjectsFromListResult(lor))
		rc.updateObjectMonitor(int64(delNum), int64(len(lor.Objects)-delNum))
		if err != nil {
			return err
		}
		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		if !lor.IsTruncated {
			break
		}
	}
	return nil
}

func (rc *RemoveCommand) ossBatchDeleteObjectsRetry(bucket *oss.Bucket, objects []string) (int, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	num := len(objects)
	if num <= 0 {
		return 0, nil
	}

	for i := 1; ; i++ {
		delRes, err := bucket.DeleteObjects(objects, oss.DeleteObjectsQuiet(true))
		if err == nil && len(delRes.DeletedObjects) == 0 {
			return num, nil
		}
		if int64(i) >= retryTimes {
			if err != nil {
				return num - len(objects), err
			}
			return num - len(delRes.DeletedObjects), fmt.Errorf("delete objects: %s failed", delRes.DeletedObjects)
		}
		objects = delRes.DeletedObjects
	}
}

func (rc *RemoveCommand) getObjectsFromListResult(lor oss.ListObjectsResult) []string {
	objects := []string{}
	for _, object := range lor.Objects {
		objects = append(objects, object.Key)
	}
	return objects
}

func (rc *RemoveCommand) removeMultipartUploadsEntry(bucket *oss.Bucket, cloudURL CloudURL) error {
	routines := 1
	chUploadIds := make(chan uploadIdInfoType, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go rc.multipartUploadsProducer(bucket, cloudURL, chUploadIds, chListError)
	for i := 0; i < routines; i++ {
		go rc.abortMultipartUploadConsumer(bucket, chUploadIds, chError)
	}

	completed := 0
	for completed <= routines {
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

func (rc *RemoveCommand) multipartUploadsProducer(bucket *oss.Bucket, cloudURL CloudURL, chUploadIds chan<- uploadIdInfoType, chListError chan<- error) {
	pre := oss.Prefix(cloudURL.object)
	keyMarker := oss.KeyMarker("")
	uploadIdMarker := oss.UploadIDMarker("")
	for {
		lmr, err := rc.command.ossListMultipartUploadsRetry(bucket, keyMarker, uploadIdMarker, pre)
		if err != nil {
			chListError <- err
			break
		}

		for _, uploadId := range lmr.Uploads {
			if !rc.rmOption.recursive && uploadId.Key != cloudURL.object {
				break
			}
			chUploadIds <- uploadIdInfoType{uploadId.Key, uploadId.UploadID}
		}

		pre = oss.Prefix(lmr.Prefix)
		keyMarker = oss.KeyMarker(lmr.NextKeyMarker)
		uploadIdMarker = oss.UploadIDMarker(lmr.NextUploadIDMarker)
		if !lmr.IsTruncated {
			break
		}
	}
	defer close(chUploadIds)
	chListError <- nil
}

func (rc *RemoveCommand) abortMultipartUploadConsumer(bucket *oss.Bucket, chUploadIds <-chan uploadIdInfoType, chError chan<- error) {
	for uploadIdInfo := range chUploadIds {
		err := rc.ossAbortMultipartUploadRetry(bucket, uploadIdInfo.key, uploadIdInfo.uploadId)
		rc.updateUploadIdMonitor(err)
		if err != nil {
			chError <- err
			return
		}
	}

	chError <- nil
}

func (rc *RemoveCommand) updateUploadIdMonitor(err error) {
	if err == nil {
		rc.monitor.updateUploadIdNum(1)
	} else {
		rc.monitor.updateErrUploadIdNum(1)
	}
	fmt.Printf(rc.monitor.progressBar(false, normalExit))
}

func (rc *RemoveCommand) ossAbortMultipartUploadRetry(bucket *oss.Bucket, key, uploadId string) error {
	var imur = oss.InitiateMultipartUploadResult{Bucket: bucket.BucketName, Key: key, UploadID: uploadId}
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		err := bucket.AbortMultipartUpload(imur)

		if err == nil {
			return err
		}

		switch err.(type) {
		case oss.ServiceError:
			if err.(oss.ServiceError).Code == "NoSuchUpload" {
				return nil
			}
		}

		if int64(i) >= retryTimes {
			return ObjectError{err, bucket.BucketName, key}
		}
	}
}

func (rc *RemoveCommand) removeBucket(bucket *oss.Bucket, cloudURL CloudURL) error {
	if !rc.confirmRemoveBucket(cloudURL) {
		return nil
	}

	rc.monitor.updateOP(bucketType)
	err := rc.ossDeleteBucketRetry(&bucket.Client, cloudURL.bucket)
	if err == nil {
		rc.monitor.updateRemovedBucket(cloudURL.bucket)
	}
	return err
}

func (rc *RemoveCommand) confirmRemoveBucket(cloudURL CloudURL) bool {
	if !rc.rmOption.force {
		var val string
		fmt.Printf(getClearStr(fmt.Sprintf("Do you really mean to remove the Bucket: %s(y or N)? ", cloudURL.bucket)))
		if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
			fmt.Println("operation is canceled.")
			return false
		}
		return true
	}
	return true
}

func (rc *RemoveCommand) ossDeleteBucketRetry(client *oss.Client, bucket string) error {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		err := client.DeleteBucket(bucket)
		if err == nil {
			return err
		}
		if int64(i) >= retryTimes {
			return BucketError{err, bucket}
		}
	}
}
