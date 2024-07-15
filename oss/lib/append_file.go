package lib

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseAppendFile = SpecText{
	synopsisText: "将本地文件内容以append上传方式上传到oss中的appendable object中",

	paramText: "local_file_name oss_object [options]",

	syntaxText: ` 
	ossutil appendfromfile local_file_name oss://bucket/object [options]
`,

	detailHelpText: ` 
	1) 如果object不存在，可以通过--meta设置object的meta信息，比如输入 --meta "x-oss-meta-author:luxun"
       可以设置x-oss-meta-author的值为luxun
    2) 如果object已经存在，不可以输入--meta信息,因为oss不支持在已经存在的append object上设置meta

用法：

    该命令只有一种用法：

    1) ossutil appendfromfile local_file_name oss://bucket/object [--meta=meta-value]
      将local_file_name内容以append方式上传到可追加的object
      如果输入--meta选项，可以设置object的meta信息
`,

	sampleText: ` 
	1) append上传文件内容，不设置meta信息
       ossutil appendfromfile local_file_name oss://bucket/object
	
    2) append上传文件内容，设置meta信息
       ossutil appendfromfile local_file_name oss://bucket/object --meta "x-oss-meta-author:luxun"
    
    3) 以访问者付费模式上传文件内容
       ossutil appendfromfile local_file_name oss://bucket/object --payer requester
`,
}

var specEnglishAppendFile = SpecText{
	synopsisText: "Upload the contents of the local file to the oss appendable object by append upload mode",

	paramText: "local_file_name oss_object [options]",

	syntaxText: ` 
	ossutil appendfromfile local_file_name oss://bucket/object [options]
`,

	detailHelpText: ` 
	1) If the object does not exist, you can set the meta information of the object with --meta
      for example:
      inputting --meta "x-oss-meta-author:luxun" can set the value of x-oss-meta-author to luxun
    2) If the object already exists, you can't input the --meta option,
      oss does not support setting the meta on the exist appendable object.

Usages：

    There is only one usage for this command:：

    1) ossutil appendfromfile local_file_name oss://bucket/object [--meta=meta-value]
      Upload the local_file_name content to the object by append mode
      If you input the --meta option, you can set the meta value of the object
`,

	sampleText: ` 
	1) Uploads file content by append mode without setting meta value
       ossutil appendfromfile local_file_name oss://bucket/object
	
    2) Uploads file content by append mode with setting meta value
       ossutil appendfromfile local_file_name oss://bucket/object --meta "x-oss-meta-author:luxun"
    
    3) Uploads file content with requester payment mode
       ossutil appendfromfile local_file_name oss://bucket/object --payer requester
`,
}

type AppendProgressListener struct {
	lastMilliSecond int64
	lastSize        int64
	currSize        int64
}

// ProgressChanged handle progress event
func (l *AppendProgressListener) ProgressChanged(event *oss.ProgressEvent) {
	if event.EventType == oss.TransferDataEvent || event.EventType == oss.TransferCompletedEvent {
		if l.lastMilliSecond == 0 {
			l.lastSize = l.currSize
			l.currSize = event.ConsumedBytes
			l.lastMilliSecond = time.Now().UnixNano() / 1000 / 1000
		} else {
			now := time.Now()
			cost := now.UnixNano()/1000/1000 - l.lastMilliSecond
			if cost > 1000 || event.EventType == oss.TransferCompletedEvent {
				l.lastSize = l.currSize
				l.currSize = event.ConsumedBytes
				l.lastMilliSecond = now.UnixNano() / 1000 / 1000

				speed := (float64(l.currSize-l.lastSize) / 1024) / (float64(cost) / 1000)
				rate := float64(l.currSize) * 100 / float64(event.TotalBytes)
				fmt.Printf("\rtotal append %d(%.2f%%) byte,speed is %.2f(KB/s)", event.ConsumedBytes, rate, speed)
			}
		}
	}
}

type appendFileOptionType struct {
	bucketName   string
	objectName   string
	encodingType string
	fileName     string
	fileSize     int64
	ossMeta      string
}

type AppendFileCommand struct {
	command       Command
	afOption      appendFileOptionType
	commonOptions []oss.Option
}

var appendFileCommand = AppendFileCommand{
	command: Command{
		name:        "appendfromfile",
		nameAlias:   []string{"appendfromfile"},
		minArgc:     2,
		maxArgc:     2,
		specChinese: specChineseAppendFile,
		specEnglish: specEnglishAppendFile,
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
			OptionEncodingType,
			OptionMeta,
			OptionMaxUpSpeed,
			OptionLogLevel,
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
func (afc *AppendFileCommand) formatHelpForWhole() string {
	return afc.command.formatHelpForWhole()
}

func (afc *AppendFileCommand) formatIndependHelp() string {
	return afc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (afc *AppendFileCommand) Init(args []string, options OptionMapType) error {
	return afc.command.Init(args, options, afc)
}

// RunCommand simulate inheritance, and polymorphism
func (afc *AppendFileCommand) RunCommand() error {
	afc.afOption.encodingType, _ = GetString(OptionEncodingType, afc.command.options)
	afc.afOption.ossMeta, _ = GetString(OptionMeta, afc.command.options)

	srcBucketUrL, err := GetCloudUrl(afc.command.args[1], afc.afOption.encodingType)
	if err != nil {
		return err
	}

	if srcBucketUrL.object == "" {
		return fmt.Errorf("object key is empty")
	}

	payer, _ := GetString(OptionRequestPayer, afc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		afc.commonOptions = append(afc.commonOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	afc.afOption.bucketName = srcBucketUrL.bucket
	afc.afOption.objectName = srcBucketUrL.object

	// check input file
	fileName := afc.command.args[0]
	stat, err := os.Stat(fileName)
	if err != nil {
		return err
	}

	if stat.IsDir() {
		return fmt.Errorf("%s is dir", fileName)
	}

	if stat.Size() > MaxAppendObjectSize {
		return fmt.Errorf("locafile:%s is bigger than %d, it is not supported by append", fileName, MaxAppendObjectSize)
	}

	afc.afOption.fileName = fileName
	afc.afOption.fileSize = stat.Size()

	// check object exist or not
	client, err := afc.command.ossClient(afc.afOption.bucketName)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(afc.afOption.bucketName)
	if err != nil {
		return err
	}

	isExist, err := bucket.IsObjectExist(afc.afOption.objectName, afc.commonOptions...)
	if err != nil {
		return err
	}

	if isExist && afc.afOption.ossMeta != "" {
		return fmt.Errorf("setting meta on existing append object is not supported")
	}

	position := int64(0)
	if isExist {
		//get object size
		props, err := bucket.GetObjectMeta(afc.afOption.objectName, afc.commonOptions...)
		if err != nil {
			return err
		}

		position, err = strconv.ParseInt(props.Get("Content-Length"), 10, 64)
		if err != nil {
			return err
		}
	}

	err = afc.AppendFromFile(bucket, position)

	return err
}

func (afc *AppendFileCommand) AppendFromFile(bucket *oss.Bucket, position int64) error {
	file, err := os.OpenFile(afc.afOption.fileName, os.O_RDONLY, 0660)
	if err != nil {
		return err
	}
	defer file.Close()

	var options []oss.Option
	if afc.afOption.ossMeta != "" {
		metas, err := afc.command.parseHeaders(afc.afOption.ossMeta, false)
		if err != nil {
			return err
		}

		options, err = afc.command.getOSSOptions(headerOptionMap, metas)
		if err != nil {
			return err
		}
	}

	var listener *AppendProgressListener = &AppendProgressListener{}
	options = append(options, oss.Progress(listener))
	options = append(options, afc.commonOptions...)

	startT := time.Now()
	newPosition, err := bucket.AppendObject(afc.afOption.objectName, file, position, options...)
	endT := time.Now()
	if err != nil {
		return err
	} else {
		cost := endT.UnixNano()/1000/1000 - startT.UnixNano()/1000/1000
		speed := float64(afc.afOption.fileSize) / float64(cost)
		fmt.Printf("\nlocal file size is %d,the object new size is %d,average speed is %.2f(KB/s)\n\n", afc.afOption.fileSize, newPosition, speed)
		return nil
	}
}
