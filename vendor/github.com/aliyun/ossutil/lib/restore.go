package lib

import (
	"fmt"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

type batchOptionType struct {
	ctnu     bool
	reporter *Reporter
}

var specChineseRestore = SpecText{

	synopsisText: "恢复冷冻状态的Objects为可读状态",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil restore cloud_url [--encoding-type url] [-r] [-f] [--output-dir=odir] [-c file] 
`,

	detailHelpText: ` 
    该命令恢复处于冷冻状态的object进入可读状态，即操作对象object必须为` + StorageArchive + `存储类型
    的object。

    一个Archive类型的object初始时处于冷冻状态。

    针对处于冷冻状态的object调用restore命令，返回成功。object处于解冻中，服务端执行解
    冻，在此期间再次调用restore命令，同样成功，且不会延长object可读状态持续时间。

    待服务端执行完成解冻任务后，object就进入了解冻状态，此时用户可以读取object。

    解冻状态默认持续1天，对于解冻状态的object调用restore命令，会将object的解冻状态延长
    一天，最多可以延长到7天，之后object又回到初始时的冷冻状态。

    更多信息见官网文档：https://help.aliyun.com/document_detail/52930.html?spm=5176.doc31947.6.874.8GjVvu 


用法：

    该命令有两种用法：

    1) ossutil restore oss://bucket/object [--encoding-type url] 
        该用法恢复单个冷冻状态object为可读状态，当指定object不存在时，ossutil会提示错
    误，此时请确保指定的url精确匹配需要设置acl的object，并且不要指定--recursive选项（
    否则ossutil会进行前缀匹配，恢复多个冷冻状态的objects为可读状态）。无论--force选项
    是否指定，都不会进行询问提示。

    2) ossutil restore oss://bucket[/prefix] -r [--encoding-type url] [-f] [--output-dir=odir]
        该用法可批量恢复多个冷冻状态的objects为可读状态，此时必须输入--recursive选项，
    ossutil会查找所有前缀匹配url的objects，恢复它们为可读状态。当一个object操作出现错
    误时，会将出错object的错误信息记录到report文件，并继续操作其他object，成功操作的
    object信息将不会被记录到report文件中（更多信息见cp命令的帮助）。如果--force选项被
    指定，则不会进行询问提示。
`,

	sampleText: ` 
    1) ossutil restore oss://bucket-restore/object-store
    2) ossutil restore oss://bucket-restore/object-prefix -r
    3) ossutil restore oss://bucket-restore/object-prefix -r -f
    4) ossutil restore oss://bucket-restore/%e4%b8%ad%e6%96%87 --encoding-type url
`,
}

var specEnglishRestore = SpecText{

	synopsisText: "Restore Frozen State Object to Read Ready Status",

	paramText: "cloud_url [options]",

	syntaxText: ` 
    ossutil restore cloud_url [--encoding-type url] [-r] [-f] [--output-dir=odir] [-c file] 
`,

	detailHelpText: ` 
    The command restore frozen state object to read ready status, the object must be in the storage 
    class of ` + StorageArchive + `. 

    An object of Archive storage class will be in frozen state at first.

    If user restore a frozen state object, the operation will success, and the object will be in 
    restroing status, oss will thaw the object. In this period, if user restore the object again, 
    the operation will success, but the time that the object can be downloaded will not be extended.

    When oss has finished restoring the object, the object can be downloaded.

    The time that an restored object can be downloaded is one day in default, if user restore the 
    object again during the time, the time that the object can be downloaded will be extended for 
    one day, the time can be at most extended to seven days. 

    More information about restore see: https://help.aliyun.com/document_detail/52930.html?spm=5176.doc31947.6.874.8GjVvu  


Usage:

    There are two usages:

    1) ossutil restore oss://bucket/object [--encoding-type url] 
        If --recursive option is not specified, ossutil restore the specified frozen state object 
    to readable status. In the usage, please make sure url exactly specified the object you want to 
    restore, if object not exist, error occurs. No matter --force option is specified or not, ossutil 
    will not show prompt question. 

    2) ossutil restore oss://bucket[/prefix] -r [--encoding-type url] [-f] [--output-dir=odir]
        The usage restore the objects with the specified prefix and in frozen state to readable status. 
    --recursive option is required for the usage, and ossutil will search for prefix-matching objects 
    and restore those objects. When an error occurs when restore an object, ossutil will record the 
    error message to report file, and ossutil will continue to attempt to set acl on the remaining 
    objects(more information see help of cp command). If --force option is specified, ossutil will 
    not show prompt question. 
`,

	sampleText: ` 
    1) ossutil restore oss://bucket-restore/object-store
    2) ossutil restore oss://bucket-restore/object-prefix -r
    3) ossutil restore oss://bucket-restore/object-prefix -r -f
    4) ossutil restore oss://bucket-restore/%e4%b8%ad%e6%96%87 --encoding-type url
`,
}

// RestoreCommand is the command list buckets or objects
type RestoreCommand struct {
	monitor  Monitor //Put first for atomic op on some fileds
	command  Command
	reOption batchOptionType
}

var restoreCommand = RestoreCommand{
	command: Command{
		name:        "restore",
		nameAlias:   []string{},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseRestore,
		specEnglish: specEnglishRestore,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionRecursion,
			OptionForce,
			OptionEncodingType,
			OptionConfigFile,
			OptionEndpoint,
			OptionAccessKeyID,
			OptionAccessKeySecret,
			OptionSTSToken,
			OptionRetryTimes,
			OptionRoutines,
			OptionOutputDir,
		},
	},
}

// function for FormatHelper interface
func (rc *RestoreCommand) formatHelpForWhole() string {
	return rc.command.formatHelpForWhole()
}

func (rc *RestoreCommand) formatIndependHelp() string {
	return rc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (rc *RestoreCommand) Init(args []string, options OptionMapType) error {
	return rc.command.Init(args, options, rc)
}

// RunCommand simulate inheritance, and polymorphism
func (rc *RestoreCommand) RunCommand() error {
	rc.monitor.init("Restored")

	encodingType, _ := GetString(OptionEncodingType, rc.command.options)
	recursive, _ := GetBool(OptionRecursion, rc.command.options)

	cloudURL, err := CloudURLFromString(rc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	if err = rc.checkArgs(cloudURL, recursive); err != nil {
		return err
	}

	bucket, err := rc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	if !recursive {
		return rc.ossRestoreObject(bucket, cloudURL.object)
	}
	return rc.batchRestoreObjects(bucket, cloudURL)
}

func (rc *RestoreCommand) checkArgs(cloudURL CloudURL, recursive bool) error {
	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", rc.command.args[0])
	}
	if !recursive && cloudURL.object == "" {
		return fmt.Errorf("restore object invalid cloud url: %s, object empty. Restore bucket is not supported, if you mean batch restore objects, please use --recursive", rc.command.args[0])
	}
	return nil
}

func (rc *RestoreCommand) ossRestoreObject(bucket *oss.Bucket, object string) error {
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)
	for i := 1; ; i++ {
		err := bucket.RestoreObject(object)
		if err == nil {
			return err
		}

		switch err.(type) {
		case oss.ServiceError:
			if err.(oss.ServiceError).StatusCode == 409 && err.(oss.ServiceError).Code == "RestoreAlreadyInProgress" {
				return nil
			}
		}

		if int64(i) >= retryTimes {
			return ObjectError{err, bucket.BucketName, object}
		}
	}
}

func (rc *RestoreCommand) batchRestoreObjects(bucket *oss.Bucket, cloudURL CloudURL) error {
	force, _ := GetBool(OptionForce, rc.command.options)
	if !force {
		var val string
		fmt.Printf("Do you really mean to recursivlly restore objects of %s(y or N)? ", rc.command.args[0])
		if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
			fmt.Println("operation is canceled.")
			return nil
		}
	}

	rc.reOption.ctnu = true
	outputDir, _ := GetString(OptionOutputDir, rc.command.options)

	// init reporter
	var err error
	if rc.reOption.reporter, err = GetReporter(rc.reOption.ctnu, outputDir, commandLine); err != nil {
		return err
	}
	defer rc.reOption.reporter.Clear()

	return rc.restoreObjects(bucket, cloudURL)
}

func (rc *RestoreCommand) restoreObjects(bucket *oss.Bucket, cloudURL CloudURL) error {
	routines, _ := GetInt(OptionRoutines, rc.command.options)

	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go rc.command.objectStatistic(bucket, cloudURL, &rc.monitor, []filterOptionType{})
	go rc.command.objectProducer(bucket, cloudURL, chObjects, chListError, []filterOptionType{})
	for i := 0; int64(i) < routines; i++ {
		go rc.restoreConsumer(bucket, cloudURL, chObjects, chError)
	}

	return rc.waitRoutinueComplete(chError, chListError, routines)
}

func (rc *RestoreCommand) restoreConsumer(bucket *oss.Bucket, cloudURL CloudURL, chObjects <-chan string, chError chan<- error) {
	for object := range chObjects {
		err := rc.restoreObjectWithReport(bucket, object)
		if err != nil {
			chError <- err
			if !rc.reOption.ctnu {
				return
			}
			continue
		}
	}

	chError <- nil
}

func (rc *RestoreCommand) restoreObjectWithReport(bucket *oss.Bucket, object string) error {
	err := rc.ossRestoreObject(bucket, object)
	rc.command.updateMonitor(err, &rc.monitor)
	msg := fmt.Sprintf("restore %s", CloudURLToString(bucket.BucketName, object))
	rc.command.report(msg, err, &rc.reOption)
	return err
}

func (rc *RestoreCommand) waitRoutinueComplete(chError, chListError <-chan error, routines int64) error {
	completed := 0
	var ferr error
	for int64(completed) <= routines {
		select {
		case err := <-chListError:
			if err != nil {
				return err
			}
			completed++
		case err := <-chError:
			if err == nil {
				completed++
			} else {
				ferr = err
				if !rc.reOption.ctnu {
					fmt.Printf(rc.monitor.progressBar(true, errExit))
					return err
				}
			}
		}
	}
	return rc.formatResultPrompt(ferr)
}

func (rc *RestoreCommand) formatResultPrompt(err error) error {
	fmt.Printf(rc.monitor.progressBar(true, normalExit))
	if err != nil && rc.reOption.ctnu {
		return nil
	}
	return err
}
