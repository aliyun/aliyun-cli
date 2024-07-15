package lib

import (
	"fmt"
	"strings"
	"sync"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseObjectTag = SpecText{
	synopsisText: "设置、查询或者删除object的tag配置",

	paramText: "cloud_url [tag_parameter] [options]",

	syntaxText: ` 
    ossutil object-tagging --method put oss://bucket[/prefix] key#value [--encoding-type url] [-r] [--payer requester] [--version-id versionId] [-c file] 
    ossutil object-tagging --method get oss://bucket[/prefix] [--encoding-type url] [-r]  [--payer requester] [--version-id versionId] [-c file] 
    ossutil object-tagging --method delete oss://bucket[/prefix] [--encoding-type url] [-r] [--payer requester] [--version-id versionId] [-c file] 
`,
	detailHelpText: ` 
    object-tagging命令通过设置method选项值为put、get、delete,可以设置、查询或者删除object的tag配置
    每个tag的key和value必须以字符'#'分隔,最多可以连续输入10个tag信息

用法:
    该命令有三种用法:
	
    1) ossutil object-tagging --method put oss://bucket/object  tagkey#tagvalue
        这个命令设置object的tag配置,key和value分别为tagkey、tagvalue
	
    2) ossutil object-tagging --method get oss://bucket/object 
        这个命令查询object的tag配置
	
    3) ossutil object-tagging --method delete oss://bucket/object
        这个命令删除object的tag配置
`,
	sampleText: ` 
    1) 设置object的tag配置
       ossutil object-tagging --method put oss://bucket/object  tagkey#tagvalue
    
    2) 批量设置objects的tag配置
       ossutil object-tagging --method put oss://bucket/prefix -r tagkey#tagvalue 
    
    3) 设置object的多个tag配置
       ossutil object-tagging --method put oss://bucket/object  tagkey1#tagvalue1 tagkey2#tagvalue2
	
    4) 查询object的tag配置
       ossutil object-tagging --method get oss://bucket/object
    
    5) 批量查询object的tag配置
       ossutil object-tagging --method get oss://bucket/prefix -r
	
    6) 删除object的tag配置
       ossutil object-tagging --method delete oss://bucket/object
    
    7) 批量删除object的tag配置
       ossutil object-tagging --method delete oss://bucket/prefix -r
`,
}

var specEnglishObjectTag = SpecText{
	synopsisText: "Set, get or delete object tag configuration",

	paramText: "cloud_url [tag_parameter] [options]",

	syntaxText: ` 
    ossutil object-tagging --method put oss://bucket[/prefix] key#value [--encoding-type url] [-r] [--payer requester] [--version-id versionId] [-c file] 
    ossutil object-tagging --method get oss://bucket[/prefix] [--encoding-type url] [-r]  [--payer requester] [--version-id versionId] [-c file] 
    ossutil object-tagging --method delete oss://bucket[/prefix] [--encoding-type url] [-r] [--payer requester] [--version-id versionId] [-c file] 
`,
	detailHelpText: ` 
    object-tagging command can set, get and delete the tag configuration of the oss object by set method option value to put, get, delete
    the key and value of each tag must be separated by the character '#', you can enter up to 10 tag parameters.
Usage:
    There are three usages for this command:
	
    1) ossutil object-tagging --method put oss://bucket/object tagkey#tagvalue
        The command sets the tag configuration of the object. The key and value are tagkey and tagvalue
	
    2) ossutil object-tagging --method get oss://bucket/object 
        The command gets the tag configuration of object

    3) ossutil object-tagging --method delete oss://bucket/object
        The command deletes the tag configuration of bucket/object
`,
	sampleText: ` 
    1) set object tag configuration with one tag   
       ossutil object-tagging --method put oss://bucket/object tagkey#tagvalue
    
    2) batch set objects tag configuration
       ossutil object-tagging --method put oss://bucket/prefix -r tagkey#tagvalue 
    
    3) set object tag configuration with serveral tags
       ossutil object-tagging --method put oss://bucket/object tagkey1#tagvalue1 tagkey2#tagvalue2 

    4) get object tag configuration
       ossutil object-tagging --method get oss://bucket/object
    
    5) batch get objects tag configuration
       ossutil object-tagging --method get oss://bucket/prefix -r
	
    6) delete object tag configuration
       ossutil object-tagging --method delete oss://bucket/object
    
    7) batch delete objects tag configuration
       ossutil object-tagging --method delete oss://bucket/prefix -r
`,
}

type ObjectTagCommand struct {
	command       Command
	monitor       Monitor
	method        string
	tagging       oss.Tagging
	commonOptions []oss.Option
	lock          sync.Mutex
	printHeader   bool
	objectIndex   int32
	reportOption  batchOptionType
}

var objectTagCommand = ObjectTagCommand{
	command: Command{
		name:        "object-tagging",
		nameAlias:   []string{"object-tagging"},
		minArgc:     1,
		maxArgc:     11,
		specChinese: specChineseObjectTag,
		specEnglish: specEnglishObjectTag,
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
			OptionRoutines,
			OptionMethod,
			OptionLogLevel,
			OptionEncodingType,
			OptionRecursion,
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
func (otc *ObjectTagCommand) formatHelpForWhole() string {
	return otc.command.formatHelpForWhole()
}

func (otc *ObjectTagCommand) formatIndependHelp() string {
	return otc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (otc *ObjectTagCommand) Init(args []string, options OptionMapType) error {
	return otc.command.Init(args, options, otc)
}

// RunCommand simulate inheritance, and polymorphism
func (otc *ObjectTagCommand) RunCommand() error {
	otc.tagging.Tags = []oss.Tag{} // clear tags for test
	otc.monitor.init("ObjectTagging")
	encodingType, _ := GetString(OptionEncodingType, otc.command.options)
	cloudUrL, err := GetCloudUrl(otc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	strMethod, _ := GetString(OptionMethod, otc.command.options)
	if strMethod == "" {
		return fmt.Errorf("--method value is empty")
	}

	strMethod = strings.ToLower(strMethod)
	if strMethod != "put" && strMethod != "get" && strMethod != "delete" {
		return fmt.Errorf("--method value is not in the optional value:put|get|delete")
	}
	otc.method = strMethod

	payer, _ := GetString(OptionRequestPayer, otc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		otc.commonOptions = append(otc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	recursive, _ := GetBool(OptionRecursion, otc.command.options)
	versionId, _ := GetString(OptionVersionId, otc.command.options)
	if len(versionId) > 0 {
		if recursive {
			return fmt.Errorf("--version-id and -r can't be both used")
		} else {
			otc.commonOptions = append(otc.commonOptions, oss.VersionId(versionId))
		}
	}

	if strMethod == "put" {
		if len(otc.command.args) < 2 {
			return fmt.Errorf("When the method value is put, there must be at least 2 parameters")
		}

		tagList := otc.command.args[1:len(otc.command.args)]
		for _, tag := range tagList {
			pSlice := strings.Split(tag, "#")
			if len(pSlice) != 2 {
				return fmt.Errorf("%s error,tag name and tag value must be separated by #", tag)
			}
			otc.tagging.Tags = append(otc.tagging.Tags, oss.Tag{Key: pSlice[0], Value: pSlice[1]})
		}
	}

	bucket, err := otc.command.ossBucket(cloudUrL.bucket)
	if err != nil {
		return err
	}

	if !recursive {
		err = otc.SingleObjectTagging(bucket, cloudUrL.object)
	} else {
		err = otc.BatchObjectTagging(bucket, *cloudUrL)
	}
	return err
}

func (otc *ObjectTagCommand) SingleObjectTagging(bucket *oss.Bucket, objectName string) error {
	if len(objectName) == 0 {
		return fmt.Errorf("object key is empty")
	}

	var tagError error
	if otc.method == "put" {
		tagError = bucket.PutObjectTagging(objectName, otc.tagging, otc.commonOptions...)
	} else if otc.method == "get" {
		resutl, err := bucket.GetObjectTagging(objectName, otc.commonOptions...)
		tagError = err
		if err == nil {
			otc.lock.Lock()
			if len(resutl.Tags) > 0 {
				otc.objectIndex++
				if !otc.printHeader {
					fmt.Printf("%-15s%-15s%s\t%s\t%s\n", "object index", "tag index", "tag key", "tag value", "object")
					fmt.Printf("---------------------------------------------------------------------------\n")
					otc.printHeader = true
				}
			}

			for index, tag := range resutl.Tags {
				fmt.Printf("%-15d%-15d\"%s\"\t\"%s\"\t%s\n", otc.objectIndex, index, tag.Key, tag.Value, CloudURLToString(bucket.BucketName, objectName))
			}
			otc.lock.Unlock()
		}
	} else if otc.method == "delete" {
		tagError = bucket.DeleteObjectTagging(objectName, otc.commonOptions...)
	}

	if tagError == nil {
		LogInfo("%s tagging success,object:%s\n", otc.method, objectName)
	} else {
		LogError("%s tagging error,object:%s,error info:%s\n", otc.method, objectName, tagError.Error())
	}

	return tagError
}

func (otc *ObjectTagCommand) BatchObjectTagging(bucket *oss.Bucket, cloudURL CloudURL) error {
	otc.reportOption.ctnu = true
	outputDir, _ := GetString(OptionOutputDir, otc.command.options)

	var err error
	if otc.reportOption.reporter, err = GetReporter(otc.reportOption.ctnu, outputDir, commandLine); err != nil {
		return err
	}
	defer otc.reportOption.reporter.Clear()

	routines, _ := GetInt(OptionRoutines, otc.command.options)
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	go otc.command.objectStatistic(bucket, cloudURL, &otc.monitor, []filterOptionType{}, otc.commonOptions...)
	go otc.command.objectProducer(bucket, cloudURL, chObjects, chListError, []filterOptionType{}, otc.commonOptions...)
	for i := 0; int64(i) < routines; i++ {
		go otc.objectTaggingConsumer(bucket, chObjects, chError)
	}
	return otc.waitRoutinueComplete(chError, chListError, routines)
}

func (otc *ObjectTagCommand) objectTaggingConsumer(bucket *oss.Bucket, chObjects <-chan string, chError chan<- error) {
	for object := range chObjects {
		err := otc.objectTaggingWithReport(bucket, object)
		if err != nil {
			chError <- err
			if !otc.reportOption.ctnu {
				return
			}
			continue
		}
	}
	chError <- nil
}

func (otc *ObjectTagCommand) objectTaggingWithReport(bucket *oss.Bucket, object string) error {
	err := otc.SingleObjectTagging(bucket, object)
	if otc.method != "get" {
		otc.command.updateMonitor(err, &otc.monitor)
		msg := fmt.Sprintf("%s %s object tagging", otc.method, CloudURLToString(bucket.BucketName, object))
		if err == nil {
			otc.command.report(msg, err, &otc.reportOption)
		} else {
			otc.command.report(msg, ObjectError{err, bucket.BucketName, object}, &otc.reportOption)
		}
	}
	return ObjectError{err, bucket.BucketName, object}
}

func (otc *ObjectTagCommand) waitRoutinueComplete(chError, chListError <-chan error, routines int64) error {
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
				if !otc.reportOption.ctnu {
					fmt.Printf(otc.monitor.progressBar(true, errExit))
					return err
				}
			}
		}
	}
	return otc.formatResultPrompt(ferr)
}

func (otc *ObjectTagCommand) formatResultPrompt(err error) error {
	if otc.method != "get" {
		fmt.Printf(otc.monitor.progressBar(true, normalExit))
	}

	if err != nil && otc.reportOption.ctnu {
		return nil
	}

	return err
}
