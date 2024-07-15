package lib

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/syndtr/goleveldb/leveldb"
)

type batchOptionType struct {
	ctnu         bool
	reporter     *Reporter
	snapshotPath string
	snapshotldb  *leveldb.DB
}

var specChineseRestore = SpecText{

	synopsisText: "恢复冷冻状态的Objects为可读状态",

	paramText: "cloud_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil restore cloud_url [local_xml_file] [--encoding-type url] [-r] [-f] [--output-dir=odir] [--version-id versionId] [--payer requester] [-c file] [--object-file file] [--snapshot-path dir] [--disable-ignore-error]
`,

	detailHelpText: ` 
    该命令恢复处于冷冻状态的object进入可读状态，即操作对象object的存储类型为StorageArchive、StorageColdArchive

    一个Archive、StorageColdArchive类型的object初始时处于冷冻状态。

    针对处于冷冻状态的object调用restore命令，返回成功。object处于解冻中，服务端执行解
    冻，在此期间再次调用restore命令，同样成功，且不会延长object可读状态持续时间。

    待服务端执行完成解冻任务后，object就进入了解冻状态，此时用户可以读取object。

    解冻状态默认持续1天，对于解冻状态的object调用restore命令，会将object的解冻状态延长
    一天，最多可以延长到7天，之后object又回到初始时的冷冻状态。

    更多信息见官网文档：https://help.aliyun.com/document_detail/52930.html?spm=5176.doc31947.6.874.8GjVvu 


用法：

    该命令有两种用法：

    1) ossutil restore oss://bucket/object [--encoding-type url] [local_xml_file] 
        该用法恢复单个冷冻状态object为可读状态，当指定object不存在时，ossutil会提示错
    误，此时请确保指定的url精确匹配需要设置acl的object，并且不要指定--recursive选项（
    否则ossutil会进行前缀匹配，恢复多个冷冻状态的objects为可读状态）。无论--force选项
    是否指定，都不会进行询问提示。

    2) ossutil restore oss://bucket[/prefix] -r [--encoding-type url] [-f] [--output-dir=odir] [local_xml_file]
        该用法可批量恢复多个冷冻状态的objects为可读状态，此时必须输入--recursive选项，
    ossutil会查找所有前缀匹配url的objects，恢复它们为可读状态。当一个object操作出现错
    误时，会将出错object的错误信息记录到report文件，并继续操作其他object，成功操作的
    object信息将不会被记录到report文件中（更多信息见cp命令的帮助）。如果--force选项被
    指定，则不会进行询问提示。

    上面的local_xml_file是本地xml格式文件, 支持设置更多的restore参数, 举例如下
    <RestoreRequest>
        <Days>2</Days>
        <JobParameters>
            <Tier>Bulk</Tier>
        </JobParameters>
    </RestoreRequest>

    3) ossutil restore oss://bucket --object-file file [--snapshot-path dir] [--disable-ignore-error] [--output-dir=odir] [local_xml_file]
        该用法可批量恢复多个冷冻状态的objects为可读状态，与-r的区别在于，-r匹配所有符合
    指定前缀的objects，而--object-file指定对某些object进行处理。ossutil会读取文件中
    的objects，恢复它们为可读状态。当一个object操作出现错误时，会将出错object的错误信息
    记录到report文件，并继续操作其他object，成功操作的object信息将不会被记录到report
    文件中（更多信息见cp命令的帮助）。如果--force选项被指定，则不会进行询问提示。

    上面的file是本地文件, 里面罗列了所有需要处理的object，之间以"\n"隔开, 举例如下
    object1
    object2
    object3
`,

	sampleText: ` 
    1) ossutil restore oss://bucket-restore/object-store
    2) ossutil restore oss://bucket-restore/object-prefix -r
    3) ossutil restore oss://bucket-restore/object-prefix -r -f
    4) ossutil restore oss://bucket-restore/%e4%b8%ad%e6%96%87 --encoding-type url
    5) ossutil restore oss://bucket-restore/object-store --payer requester
    6) ossutil restore oss://bucket-restore/object-prefix -r -f local_xml_file
    7) ossutil restore oss://bucket-restore --object-file file -f local_xml_file
    8) ossutil restore oss://bucket-restore --object-file file --snapshot-path dir -f local_xml_file
`,
}

var specEnglishRestore = SpecText{

	synopsisText: "Restore Frozen State Object to Read Ready Status",

	paramText: "cloud_url [local_xml_file] [options]",

	syntaxText: ` 
    ossutil restore cloud_url [local_xml_file] [--encoding-type url] [-r] [-f] [--output-dir=odir] [--version-id versionId] [--payer requester] [-c file] [--object-file file] [--snapshot-path dir] [--disable-ignore-error]
`,

	detailHelpText: ` 
    The command restore frozen state object to read ready status, the object must be in the storage 
    class of StorageArchive、StorageColdArchive

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

    1) ossutil restore oss://bucket/object [--encoding-type url] [local_xml_file]
        If --recursive option is not specified, ossutil restore the specified frozen state object 
    to readable status. In the usage, please make sure url exactly specified the object you want to 
    restore, if object not exist, error occurs. No matter --force option is specified or not, ossutil 
    will not show prompt question. 

    2) ossutil restore oss://bucket[/prefix] -r [--encoding-type url] [-f] [--output-dir=odir] [local_xml_file]
        The usage restore the objects with the specified prefix and in frozen state to readable status. 
    --recursive option is required for the usage, and ossutil will search for prefix-matching objects 
    and restore those objects. When an error occurs when restore an object, ossutil will record the 
    error message to report file, and ossutil will continue to attempt to set acl on the remaining 
    objects(more information see help of cp command). If --force option is specified, ossutil will 
    not show prompt question. 

    The local_xml_file is a local XML format file, which supports setting more restore configurations. For example:
    <RestoreRequest>
        <Days>2</Days>
        <JobParameters>
            <Tier>Bulk</Tier>
        </JobParameters>
    </RestoreRequest>

    3) ossutil restore oss://bucket --object-file file [--snapshot-path dir] [--disable-ignore-error] [--output-dir=odir] [local_xml_file]
        The usage restore the objects with the specified prefix and in frozen state to readable status. 
    The difference with -r is that -r matches all objects that match the specified prefix, while 
    --object-file specifies some objects to be restored. Ossutil reads the objects in the file and 
    restores them to a readable state. When an error occurs when restore an object, ossutil will 
    record the error message to report file, and ossutil will continue to attempt to set acl on the 
    remaining objects(more information see help of cp command). If --force option is specified,
    ossutil will not show prompt question. 

    The file is a local file, which lists all the objects to be restored, separated by "\n". For example:
    object1
    object2
    object3
`,

	sampleText: ` 
    1) ossutil restore oss://bucket-restore/object-store
    2) ossutil restore oss://bucket-restore/object-prefix -r
    3) ossutil restore oss://bucket-restore/object-prefix -r -f
    4) ossutil restore oss://bucket-restore/%e4%b8%ad%e6%96%87 --encoding-type url
    5) ossutil restore oss://bucket-restore/object-store --payer requester
    6) ossutil restore oss://bucket-restore/object-prefix -r -f local_xml_file
    7) ossutil restore oss://bucket-restore --object-file file -f local_xml_file
    8) ossutil restore oss://bucket-restore --object-file file --snapshot-path dir -f local_xml_file
`,
}

// RestoreCommand is the command list buckets or objects
type RestoreCommand struct {
	monitor       Monitor //Put first for atomic op on some fileds
	command       Command
	reOption      batchOptionType
	commonOptions []oss.Option
	restoreConfig oss.RestoreConfiguration
	hasConfig     bool
	configXml     string
	hasObjFile    bool
	objFilePath   string
}

var restoreCommand = RestoreCommand{
	command: Command{
		name:        "restore",
		nameAlias:   []string{},
		minArgc:     1,
		maxArgc:     2,
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
			OptionProxyHost,
			OptionProxyUser,
			OptionProxyPwd,
			OptionRetryTimes,
			OptionRoutines,
			OptionOutputDir,
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
			OptionObjectFile,
			OptionSnapshotPath,
			OptionDisableIgnoreError,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,
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
	rc.monitor.init("Restore")
	encodingType, _ := GetString(OptionEncodingType, rc.command.options)
	recursive, _ := GetBool(OptionRecursion, rc.command.options)
	versionid, _ := GetString(OptionVersionId, rc.command.options)
	force, _ := GetBool(OptionForce, rc.command.options)
	objFileXml, _ := GetString(OptionObjectFile, rc.command.options)
	snapshotPath, _ := GetString(OptionSnapshotPath, rc.command.options)

	payer, _ := GetString(OptionRequestPayer, rc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		rc.commonOptions = append(rc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	var err error
	// load snapshot
	rc.reOption.snapshotPath = snapshotPath
	if rc.reOption.snapshotPath != "" {
		if rc.reOption.snapshotldb, err = leveldb.OpenFile(rc.reOption.snapshotPath, nil); err != nil {
			return fmt.Errorf("load snapshot error, reason: %s", err.Error())
		}
		defer rc.reOption.snapshotldb.Close()
	}

	cloudURL, err := CloudURLFromString(rc.command.args[0], encodingType)
	if err != nil {
		return err
	}
	if err = rc.checkOptions(cloudURL, recursive, force, versionid, objFileXml); err != nil {
		return err
	}
	bucket, err := rc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	// parse restore config
	if len(rc.command.args) > 1 {
		restoreConf := rc.command.args[1]
		err := rc.parseRestoreConf(restoreConf)
		if err != nil {
			return err
		}
	} else {
		rc.hasConfig = false
	}

	if objFileXml != "" {
		// check objFileXml and parse it
		if err := rc.checkObjectFile(objFileXml); err != nil {
			return err
		}
		recursive = true
		return rc.batchRestoreObjects(bucket, cloudURL, recursive)
	} else {
		if !recursive {
			return rc.ossRestoreObject(bucket, cloudURL.object, versionid, false)
		}

		return rc.batchRestoreObjects(bucket, cloudURL, recursive)
	}
}

func (rc *RestoreCommand) checkOptions(cloudURL CloudURL, recursive, force bool, versionid, objectFile string) error {
	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", cloudURL.urlStr)
	}
	if cloudURL.object == "" {
		if !recursive && objectFile == "" {
			return fmt.Errorf("restore object invalid cloud url: %s, object empty. Restore bucket is not supported, if you mean batch restore objects, please use --recursive or --object-file", rc.command.args[0])
		}
	} else {
		if objectFile != "" {
			return fmt.Errorf("the first arg of `ossutil restore` only support oss://bucket when set option --object-file")
		}
	}

	if (recursive && len(versionid) > 0) || (objectFile != "" && len(versionid) > 0) {
		return fmt.Errorf("restore bucket dose not support the --version-id=%s argument.", versionid)
	}

	if !force {
		var val string
		if !recursive && objectFile == "" {
			return nil
		}
		fmt.Printf("Do you really mean to recursivlly restore objects of %s(y or N)? ", rc.command.args[0])
		if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
			fmt.Println("operation is canceled.")
			return nil
		}
	}

	return nil
}

func (rc *RestoreCommand) ossRestoreObject(bucket *oss.Bucket, object, versionid string, batchOperate bool) error {
	nowt := time.Now().Unix()
	spath := ""
	msg := "restore"

	if batchOperate && rc.reOption.snapshotPath != "" {
		spath = rc.formatSnapshotKey(bucket.BucketName, object, msg)
		if skip := rc.skipRestore(spath); skip {
			rc.updateSkip(1)
			LogInfo("restore obj skip: %s\n", object)
			return nil
		}
	}

	var options []oss.Option
	if len(versionid) > 0 {
		options = append(options, oss.VersionId(versionid))
	}
	options = append(options, rc.commonOptions...)

	err := rc.ossRestoreObjectRetry(bucket, object, options...)
	if batchOperate && rc.reOption.snapshotPath != "" {
		if err != nil {
			_ = rc.updateSnapshot(err, spath, nowt)
			return err
		} else {
			err = rc.updateSnapshot(err, spath, nowt)
			if err != nil {
				return err
			}
		}
	} else {
		return err
	}

	return nil
}

func (rc *RestoreCommand) ossRestoreObjectRetry(bucket *oss.Bucket, object string, options ...oss.Option) error {
	var err error
	retryTimes, _ := GetInt(OptionRetryTimes, rc.command.options)

	for i := 1; ; i++ {
		if rc.hasConfig {
			err = bucket.RestoreObjectXML(object, rc.configXml, options...)
		} else {
			err = bucket.RestoreObject(object, options...)
		}

		if err == nil {
			return nil
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

func (rc *RestoreCommand) batchRestoreObjects(bucket *oss.Bucket, cloudURL CloudURL, recursive bool) error {
	rc.reOption.ctnu = true
	if recursive || rc.hasObjFile {
		disableIgnoreError, _ := GetBool(OptionDisableIgnoreError, rc.command.options)
		rc.reOption.ctnu = !disableIgnoreError
	}
	outputDir, _ := GetString(OptionOutputDir, rc.command.options)

	// init reporter
	var err error
	if rc.reOption.reporter, err = GetReporter(rc.reOption.ctnu, outputDir, commandLine); err != nil {
		return err
	}
	defer rc.reOption.reporter.Clear()

	if rc.hasObjFile {
		return rc.restoreObjectsFromFile(bucket, cloudURL, rc.objFilePath)
	}
	return rc.restoreObjects(bucket, cloudURL)
}

func (rc *RestoreCommand) restoreObjects(bucket *oss.Bucket, cloudURL CloudURL) error {
	routines, _ := GetInt(OptionRoutines, rc.command.options)
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go rc.command.objectStatistic(bucket, cloudURL, &rc.monitor, []filterOptionType{}, rc.commonOptions...)
	go rc.command.objectProducer(bucket, cloudURL, chObjects, chListError, []filterOptionType{}, rc.commonOptions...)
	for i := 0; int64(i) < routines; i++ {
		go rc.restoreConsumer(bucket, cloudURL, chObjects, chError)
	}

	return rc.waitRoutinueComplete(chError, chListError, routines)
}

func (rc *RestoreCommand) restoreObjectsFromFile(bucket *oss.Bucket, cloudURL CloudURL, objectFile string) error {
	routines, _ := GetInt(OptionRoutines, rc.command.options)

	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go rc.restoreStatistic(objectFile, &rc.monitor, []filterOptionType{}, rc.commonOptions...)
	go rc.restoreProducer(objectFile, chObjects, chListError, []filterOptionType{}, rc.commonOptions...)
	for i := 0; int64(i) < routines; i++ {
		go rc.restoreConsumer(bucket, cloudURL, chObjects, chError)
	}
	return rc.waitRoutinueComplete(chError, chListError, routines)
}

func (rc *RestoreCommand) restoreStatistic(objectFile string, monitor Monitorer, filters []filterOptionType, options ...oss.Option) {
	if monitor == nil {
		return
	}

	file, err := os.Open(objectFile)
	if err != nil {
		monitor.setScanError(err)
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		object := scanner.Text()
		object = strings.Trim(object, " ")
		if object == "" {
			monitor.setScanError(fmt.Errorf("object can't be '' in --object-file"))
			return
		}
		monitor.updateScanNum(1)
	}

	monitor.setScanEnd()
}

func (rc *RestoreCommand) restoreProducer(objectFile string, chObjects chan<- string, chError chan<- error, filters []filterOptionType, options ...oss.Option) {
	defer close(chObjects)
	file, err := os.Open(objectFile)
	if err != nil {
		chError <- err
		return
	}
	defer file.Close()
	encodingType, _ := GetString(OptionEncodingType, rc.command.options)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		object := scanner.Text()
		object = strings.Trim(object, " ")
		if object == "" {
			chError <- fmt.Errorf("object can't be '' in --object-file")
			return
		}
		if encodingType == URLEncodingType {
			oldObject := object
			if object, err = url.QueryUnescape(oldObject); err != nil {
				chError <- fmt.Errorf("invalid object url: %s, object name is not url encoded, %s", oldObject, err.Error())
				return
			}
		}
		chObjects <- object
	}
	chError <- nil
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
	err := rc.ossRestoreObject(bucket, object, "", true)
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

func (rc *RestoreCommand) parseRestoreConf(xmlFile string) error {
	fileInfo, err := os.Stat(xmlFile)
	if err != nil {
		return err
	}

	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir,not the expected file", xmlFile)
	}

	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", xmlFile)
	}

	// parsing the xml file
	file, err := os.Open(xmlFile)
	if err != nil {
		return err
	}
	defer file.Close()
	text, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	rc.hasConfig = true
	rc.configXml = string(text)
	return nil
}

func (rc *RestoreCommand) formatSnapshotKey(bucket, object, msg string) string {
	return CloudURLToString(bucket, object) + SnapshotConnector + msg
}

func (rc *RestoreCommand) skipRestore(spath string) bool {
	if rc.reOption.snapshotPath != "" {
		_, err := rc.reOption.snapshotldb.Get([]byte(spath), nil)
		if err == nil {
			return true
		}
	}
	return false
}

func (rc *RestoreCommand) updateSnapshot(err error, spath string, srct int64) error {
	if rc.reOption.snapshotPath != "" && err == nil {
		srctstr := fmt.Sprintf("%d", srct)
		err := rc.reOption.snapshotldb.Put([]byte(spath), []byte(srctstr), nil)
		if err != nil {
			return fmt.Errorf("dump snapshot error: %s", err.Error())
		}
	}
	return nil
}

func (rc *RestoreCommand) checkObjectFile(objFileXml string) error {
	// check file if exists
	fileInfo, err := os.Stat(objFileXml)
	if err != nil {
		return err
	}
	if fileInfo.IsDir() {
		return fmt.Errorf("%s is dir, not the expected file", objFileXml)
	}
	if fileInfo.Size() == 0 {
		return fmt.Errorf("%s is empty file", objFileXml)
	}

	rc.hasObjFile = true
	rc.objFilePath = objFileXml
	return nil
}

func (rc *RestoreCommand) updateSkip(num int64) {
	atomic.AddInt64(&rc.monitor.skipNum, num)
}
