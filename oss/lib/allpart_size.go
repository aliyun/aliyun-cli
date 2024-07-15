package lib

import (
	"fmt"
	"net/url"
	"strconv"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseAllPartSize = SpecText{
	synopsisText: "获取bucket所有未完成上传的multipart object的分块大小以及总和",

	paramText: "bucket_url [options]",

	syntaxText: ` 
	ossutil getallpartsize oss://bucket [options]
`,

	detailHelpText: ` 
	该命令会获取bucket所有未完成上传的multipart object的分块大小以及总和
  

用法：

    该命令只有一种用法：

    1) ossutil getallpartsize oss://bucket [options]
      查询bucket的所有未完成上传的multipart object的块大小信息以及总和
`,

	sampleText: ` 
	1) 根据bucket查询所有未完成上传的multipart object的块大小信息以及总和
       ossutil getallpartsize oss://bucket
`,
}

var specEnglishAllPartSize = SpecText{
	synopsisText: "Get bucket all uncompleted multipart objects's parts size and sum size",

	paramText: "bucket_url [options]",

	syntaxText: ` 
	ossutil getallpartsize oss://bucket [options]
`,

	detailHelpText: ` 
	This command will list every uncompleted multipart objects's part size and sum size
  

Usages：

    There is only one usage for this command:

    1) ossutil getallpartsize oss://bucket [options]
       Get bucket all uncompleted mulitpart objects's parts size and sum size
`,

	sampleText: ` 
	1)  Get bucket all uncompleted multipart objects's parts size and sum size
       ossutil getallpartsize oss://bucket
`,
}

type allPartSizeOptionType struct {
	bucketName     string
	encodingType   string
	headLineShowed bool
	statList       []StatPartInfo
}

type AllPartSizeCommand struct {
	command  Command
	apOption allPartSizeOptionType
}

var allPartSizeCommand = AllPartSizeCommand{
	command: Command{
		name:        "getallpartsize",
		nameAlias:   []string{"getallpartsize"},
		minArgc:     1,
		maxArgc:     1,
		specChinese: specChineseAllPartSize,
		specEnglish: specEnglishAllPartSize,
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
			OptionLogLevel,
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

type StatPartInfo struct {
	objectName string
	uploadId   string
}

// function for FormatHelper interface
func (apc *AllPartSizeCommand) formatHelpForWhole() string {
	return apc.command.formatHelpForWhole()
}

func (apc *AllPartSizeCommand) formatIndependHelp() string {
	return apc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (apc *AllPartSizeCommand) Init(args []string, options OptionMapType) error {
	return apc.command.Init(args, options, apc)
}

// RunCommand simulate inheritance, and polymorphism
func (apc *AllPartSizeCommand) RunCommand() error {
	srcBucketUrL, err := GetCloudUrl(apc.command.args[0], "")
	if err != nil {
		return err
	}
	apc.apOption.bucketName = srcBucketUrL.bucket
	apc.apOption.encodingType, _ = GetString(OptionEncodingType, apc.command.options)

	// first:get all object uploadid
	err = apc.GetAllStatInfo()
	if err != nil {
		return err
	}

	//second:stat every object parts
	client, err := apc.command.ossClient(apc.apOption.bucketName)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(apc.apOption.bucketName)
	if err != nil {
		return err
	}

	var totalPartCount int64 = 0
	var totalPartSize int64 = 0
	for _, v := range apc.apOption.statList {
		partCount, partSize, err := apc.GetObjectPartsSize(bucket, v)
		if err != nil {
			return err
		}
		totalPartCount += partCount
		totalPartSize += partSize
	}

	if totalPartSize > 0 {
		fmt.Printf("\ntotal part count:%d\ttotal part size(MB):%.2f\n\n", totalPartCount, float64(totalPartSize/1024)/1024)
	}

	return nil
}

func (apc *AllPartSizeCommand) GetAllStatInfo() error {
	client, err := apc.command.ossClient(apc.apOption.bucketName)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(apc.apOption.bucketName)
	if err != nil {
		return err
	}

	keyMarker := ""
	uploadIdMarker := ""
	for i := 0; ; i++ {
		lpOptions := []oss.Option{}
		lpOptions = append(lpOptions, oss.MaxParts(1000))
		lpOptions = append(lpOptions, oss.KeyMarker(keyMarker))
		lpOptions = append(lpOptions, oss.UploadIDMarker(uploadIdMarker))

		lpRes, err := bucket.ListMultipartUploads(lpOptions...)
		if err != nil {
			return err
		}

		for _, v := range lpRes.Uploads {
			var statPartInfo StatPartInfo
			statPartInfo.objectName = v.Key
			statPartInfo.uploadId = v.UploadID
			apc.apOption.statList = append(apc.apOption.statList, statPartInfo)
		}

		if lpRes.IsTruncated {
			keyMarker = lpRes.NextKeyMarker
			uploadIdMarker = lpRes.NextUploadIDMarker
		} else {
			break
		}
	}
	return nil
}

func (apc *AllPartSizeCommand) GetObjectPartsSize(bucket *oss.Bucket, statInfo StatPartInfo) (int64, int64, error) {
	var imur oss.InitiateMultipartUploadResult
	imur.Bucket = apc.apOption.bucketName
	imur.Key = statInfo.objectName
	imur.UploadID = statInfo.uploadId

	partNumberMarker := 0
	var totalPartCount int64 = 0
	var totalPartSize int64 = 0
	var cloudUrl CloudURL
	for i := 0; ; i++ {
		lpOptions := []oss.Option{}
		lpOptions = append(lpOptions, oss.MaxParts(1000))
		lpOptions = append(lpOptions, oss.PartNumberMarker(partNumberMarker))

		lpRes, err := bucket.ListUploadedParts(imur, lpOptions...)
		if err != nil {
			return 0, 0, err
		} else {
			totalPartCount += int64(len(lpRes.UploadedParts))
			if !apc.apOption.headLineShowed && len(lpRes.UploadedParts) > 0 {
				fmt.Printf("%-10s\t%-32s\t%-10s\t%s\n", "PartNumber", "UploadId", "Size(Byte)", "Path")
				apc.apOption.headLineShowed = true
			}
		}

		for _, v := range lpRes.UploadedParts {
			cloudUrl.bucket = apc.apOption.bucketName
			if apc.apOption.encodingType == URLEncodingType {
				cloudUrl.object = url.QueryEscape(imur.Key)
			} else {
				cloudUrl.object = imur.Key
			}

			//PartNumber,uploadId,Size,Path
			fmt.Printf("%-10d\t%-32s\t%-10d\t%s\n", v.PartNumber, imur.UploadID, v.Size, cloudUrl.ToString())
			totalPartSize += int64(v.Size)
		}

		if lpRes.IsTruncated {
			partNumberMarker, err = strconv.Atoi(lpRes.NextPartNumberMarker)
			if err != nil {
				return 0, 0, err
			}
		} else {
			break
		}
	}
	return totalPartCount, totalPartSize, nil

}
