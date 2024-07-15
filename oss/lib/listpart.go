package lib

import (
	"fmt"
	"strconv"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseListPart = SpecText{
	synopsisText: "列出没有完成分块上传的object的分块信息",

	paramText: "oss_object uploadid [options]",

	syntaxText: ` 
	ossutil listpart oss://bucket/object uploadid [options]
`,

	detailHelpText: ` 
	可以通过ls命令查看bucket的object和uploadid信息，在用本命令查看详细信息
  

用法：

    该命令只有一种用法：

    1) ossutil listpart oss://bucket/object uploadid [options]
      根据object和uploadid查询块信息
`,

	sampleText: ` 
	1) 根据object和uploadid查询块信息
       ossutil listpart oss://bucket/object 8A1912289A705A5F0503FCA71DABFD5A
`,
}

var specEnglishListPart = SpecText{
	synopsisText: "List parts information of uncompleted multipart object",

	paramText: "oss_object uploadid [options]",

	syntaxText: ` 
	ossutil listpart oss://bucket/object uploadid [options]
`,

	detailHelpText: ` 
	You can use the ls command to view the object and uploadid information of a bucket.
    Then Use this command to view detailed part information.

Usages：

    There is only one usage for this command:

    1) ossutil listpart oss://bucket/object uploadid [options]

      Query parts information according to object and uploadid
`,

	sampleText: ` 
	1) Query parts information according to object and uploadid

      ossutil listpart oss://bucket/object 8A1912289A705A5F0503FCA71DABFD5A
`,
}

type listPartOptionType struct {
	cloudUrl     CloudURL
	uploadId     string
	encodingType string
}

type ListPartCommand struct {
	command  Command
	lpOption listPartOptionType
}

var listPartCommand = ListPartCommand{
	command: Command{
		name:        "listpart",
		nameAlias:   []string{"listpart"},
		minArgc:     2,
		maxArgc:     2,
		specChinese: specChineseListPart,
		specEnglish: specEnglishListPart,
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

// function for FormatHelper interface
func (lpc *ListPartCommand) formatHelpForWhole() string {
	return lpc.command.formatHelpForWhole()
}

func (lpc *ListPartCommand) formatIndependHelp() string {
	return lpc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (lpc *ListPartCommand) Init(args []string, options OptionMapType) error {
	return lpc.command.Init(args, options, lpc)
}

// RunCommand simulate inheritance, and polymorphism
func (lpc *ListPartCommand) RunCommand() error {
	lpc.lpOption.encodingType, _ = GetString(OptionEncodingType, lpc.command.options)
	srcBucketUrL, err := GetCloudUrl(lpc.command.args[0], lpc.lpOption.encodingType)
	if err != nil {
		return err
	}

	if srcBucketUrL.object == "" {
		return fmt.Errorf("object name is empty")
	}

	lpc.lpOption.cloudUrl = *srcBucketUrL
	lpc.lpOption.uploadId = lpc.command.args[1]

	return lpc.ListPart()
}

func (lpc *ListPartCommand) ListPart() error {
	client, err := lpc.command.ossClient(lpc.lpOption.cloudUrl.bucket)
	if err != nil {
		return err
	}

	bucket, err := client.Bucket(lpc.lpOption.cloudUrl.bucket)
	if err != nil {
		return err
	}

	var imur oss.InitiateMultipartUploadResult
	imur.Bucket = lpc.lpOption.cloudUrl.bucket
	imur.Key = lpc.lpOption.cloudUrl.object
	imur.UploadID = lpc.lpOption.uploadId

	partNumberMarker := 0
	totalPartCount := 0
	var totalPartSize int64 = 0
	for i := 0; ; i++ {
		lpOptions := []oss.Option{}
		lpOptions = append(lpOptions, oss.MaxParts(1000))
		lpOptions = append(lpOptions, oss.PartNumberMarker(partNumberMarker))

		lpRes, err := bucket.ListUploadedParts(imur, lpOptions...)
		if err != nil {
			return err
		} else {
			totalPartCount += len(lpRes.UploadedParts)
			if i == 0 && len(lpRes.UploadedParts) > 0 {
				fmt.Printf("%-10s\t%-32s\t%-10s\t%s\n", "PartNumber", "Etag", "Size(Byte)", "LastModifyTime")
			}
		}

		for _, v := range lpRes.UploadedParts {
			//PartNumber,ETag,Size,LastModified
			fmt.Printf("%-10d\t%-32s\t%-10d\t%s\n", v.PartNumber, v.ETag, v.Size, v.LastModified.Format("2006-01-02 15:04:05"))
			totalPartSize += int64(v.Size)
		}

		if lpRes.IsTruncated {
			partNumberMarker, err = strconv.Atoi(lpRes.NextPartNumberMarker)
			if err != nil {
				return err
			}
		} else {
			if totalPartCount > 0 {
				fmt.Printf("\ntotal part count:%d\ttotal part size(MB):%.2f\n\n", totalPartCount, float64(totalPartSize/1024)/1024)
			}
			break
		}
	}
	return nil
}
