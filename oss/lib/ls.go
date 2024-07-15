package lib

import (
	"fmt"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var specChineseList = SpecText{

	synopsisText: "列举Buckets或者Objects",

	paramText: "[cloud_url] [options]",

	syntaxText: ` 
    ossutil ls [oss://bucket[/prefix]] [-s] [-d] [-m] [--limited-num num] [--marker marker] [--upload-id-marker umarker] [--payer requester] [--include include-pattern] [--exclude exclude-pattern]  [--version-id-marker id_marker] [--all-versions]  [-c file] 
`,

	detailHelpText: ` 
    该命令列举指定身份凭证下的buckets，或该身份凭证下对应endpoint的objects。默认显示长格式，
    ossutil在列举buckets或者objects的同时展示它们的一些附加信息。如果指定了--short-format选
    项，则显示精简格式。

--encoding-type选项

    如果指定了encoding-type为url，则表示输入的object（或prefix）为经过url编码的，此时如果指定了
    --marker选项或--upload-id-marker选项，ossutil默认指定的marker或upload-id-marker也同样是经过
    url编码的。注意：形如oss://bucket/object的cloud_url，输入形式为：oss://bucket/url_encode(object)，
    其中oss://bucket/字符串不需要编码。

--include和--exclude选项

    可以指定该选项以指定规则筛选要操作的文件/object

    规则支持以下格式：
    *：匹配索引
    ?：匹配单个字符
    [sequence]：匹配sequence的任意字符
    [!sequence]：匹配不在sequence的任意字符
    注意：规则不支持带目录的格式，e.g.，--include "/usr/*/test/*.jpg"。

    --include和--exclude可以出现多次。当多个规则出现时，这些规则按从左往右的顺序应用

用法：

    该命令有两种用法：

    1) ossutil ls [oss://] [-s] [--limited-num num] [--marker marker]
        如果用户列举时缺失cloud_url参数，则ossutil获取用户的身份凭证信息（从配置文件中读取），
    并列举该身份凭证下的所有buckets，并显示每个bucket的最新更新时间，位置，存储方式等信息。
    如果指定了--short-format选项则只输出bucket名称。该用法不支持--directory选项。

    2) ossutil ls oss://bucket[/prefix] [-s] [-d] [-m] [-a] [--limited-num num] [--marker marker] [--upload-id-marker umarker]  [--version-id-marker id_marker] [--all-versions]
        如果未指定--multipart和--all-type选项，则ossutil列举指定bucket下的objects（如果指定
    了前缀，则列举拥有该前缀的objects）。并同时展示object大小，最新更新时间和etag，但是如果
    指定了--short-format选项则只输出object名称。如果指定了--directory选项，则返回指定bucket
    下以指定前缀开头的第一层目录下的文件和子目录，但是不递归显示所有子目录，此时默认为精简
    格式。所有的目录均以/结尾。
        如果指定了--multipart选项，则显示指定URL(oss://bucket[/prefix])下未完成的上传任务，
    即，列举未complete的Multipart Upload事件的uploadId，这些Multipart Upload事件的object名
    称以指定的prefix为前缀。ossutil同时显示uploadId的init时间。该选项同样支持--short-format
    和--directory选项。（Multipart同样用于cp命令中大文件的断点续传，关于Multipart的更多信息
    见：https://help.aliyun.com/document_detail/31991.html?spm=5176.doc31992.6.880.VOSDk5）。
        如果指定了--all-type选项，则显示指定URL(oss://bucket[/prefix])下的object和未完成的
	上传任务（即，同时列举以prefix为前缀的object，和object名称以prefix为前缀的所有未complete
    的uploadId）。该选项同样支持--short-format和--directory选项。
        如果指定了--limited-num选项，ossutil总共会输出的对象个数不超过limited-num个，当同时
    输出object和Multipart Upload时，两者的总数不超过limited-num个。
        在列举objects时，--upload-id-marker选项不起作用。在列举Multipart Uploads事件时，--marker
    和--upload-id-marker选项同时限定了列举的起始位置，更多信息请见oss的官网：
    https://help.aliyun.com/document_detail/31997.html?spm=5176.doc31965.6.887.MK6GVw.
`,

	sampleText: ` 
    1) ossutil ls -s
        oss://bucket1
        oss://bucket2
        oss://bucket3
        Bucket Number is: 3

    2) ossutil ls oss:// -s
        oss://bucket1
        oss://bucket2
        oss://bucket3
        Bucket Number is: 3

    3) ossutil ls oss://bucket1 -s
        oss://bucket1/dir1/obj11
        oss://bucket1/obj1
        oss://bucket1/sample.txt
        Object Number is: 3

    4) ossutil ls oss://bucket1
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:06:29 +0000 CST  201933      Standard   7E2F4A7F1AC9D2F0996E8332D5EA5B41  oss://bucket1/dir1/obj11
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        2016-04-08 14:50:47 +0000 CST 6476984      Standard   4F16FDAE7AC404CEC8B727FCC67779D6  oss://bucket1/sample.txt
        Object Number is: 3

    5) ossutil ls oss://bucket1 -d
        oss://bucket1/obj1
        oss://bucket1/dir1
        oss://bucket1/sample.txt
        Object and Directory Number is: 3

    6) ossutil ls oss://bucket1 -m 
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        2017-01-20 11:16:21 +0800 CST  A20157A7B2FEC4670626DAE0F4C0073C  oss://bucket1/tobj
        UploadID Number is: 3
    
    7) ossutil ls oss://bucket1/obj -m 
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 2
 
    8) ossutil ls oss://bucket1 -a 
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:06:29 +0000 CST  201933      Standard   7E2F4A7F1AC9D2F0996E8332D5EA5B41  oss://bucket1/dir1/obj11
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        2016-04-08 14:50:47 +0000 CST 6476984      Standard   4F16FDAE7AC404CEC8B727FCC67779D6  oss://bucket1/sample.txt
        Object Number is: 3
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:43:13 +0000 CST  2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        2017-01-20 11:16:21 +0800 CST  A20157A7B2FEC4670626DAE0F4C0073C  oss://bucket1/tobj
        UploadID Number is: 4
         
    9) ossutil ls oss://bucket1/obj -a 
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        Object Number is: 1
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:43:13 +0000 CST  2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 3

    10) ossutil ls oss://bucket1/obj -a -s 
        oss://bucket1/obj1
        Object Number is: 1
        UploadID                          ObjectName
        15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 3

    11) ossutil ls oss://bucket1/obj -a -s --marker=obj1 
        Object Number is: 0
        UploadID                          ObjectName
        3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 1

    12) ossutil ls oss://bucket1/obj -a -s --limited-num=2 
        oss://bucket1/obj1
        Object Number is: 1
        UploadID                          ObjectName
        15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        UploadID Number is: 1

    13) ossutil ls oss://bucket1/%e4%b8%ad%e6%96%87 --encoding-type url
        LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
        2017-03-17 17:34:40 +0800 CST      8345742      Standard   BBCC8C0954B869B4A6B34D9404C5BCFD      oss://bucket1/中文
        Object Number is: 1
        0.066567(s) elapsed
    
    14) ossutil ls oss://bucket --include "*.avi" --include "*.mp4" --exclude "*.png" --exclude "*.jpg"
        LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
        2019-05-30 14:23:51 +0800 CST         1030      Standard   4A902D176BE0EE4224BC196BBB8CCC69      oss://bucket/test.avi
        2019-05-30 14:24:05 +0800 CST         1030      Standard   4A902D176BE0EE4224BC196BBB8CCC69      oss://bucket/test.mp4
        Object Number is: 2

    15) ossutil ls oss://bucket --all-versions
`,
}

var specEnglishList = SpecText{

	synopsisText: "List Buckets or Objects",

	paramText: "[cloud_url] [options]",

	syntaxText: ` 
    ossutil ls [oss://bucket[/prefix]] [-s] [-d] [-m] [--limited-num num] [--marker marker] [--upload-id-marker umarker] [--payer requester] [--include include-pattern] [--exclude exclude-pattern]  [--version-id-marker id_marker] [--all-versions]  [-c file] 
`,

	detailHelpText: ` 
    The command list buckets of the specified credentials. or objects of the specified 
    endpoint and credentials, with simple additional information, about each matching 
    provider, bucket, subdirectory, or object. If --short-format option is specified, 
    ossutil will show by short format. 

--encoding-type option

    If the --encoding-type option is setted to url, the object/prefix inputted is url 
    encoded, if the --marker option or --upload-id-marker option is specified, ossutil 
    will consider the marker or upload-id-marker inputted is also url encoded.

    Note: If the option is specified, the cloud_url like: oss://bucket/object should be 
    inputted as: oss://bucket/url_encode(object), the string: oss://bucket/ should not 
    be url encoded. 

--include and --exclude option:

    These parameters perform pattern matching to either exclude or include a particular file or object

    The following pattern symbols are supported.
    *: Matches everything
    ?: Matches any single character
    [sequence]: Matches any character in sequence
    [!sequence]: Matches any character not in sequence
    Note: does not support patterns containing directory info. e.g., --include "/usr/*/test/*.jpg" 

    Any number of these parameters can be passed to a command. You can do this by providing an --exclude
    or --include argument multiple times, e.g.,
      --include "*.txt" --include "*.png". 
    When there are multi filters, the rule is the filters that appear later in the command take precedence
    over filters that appear earlier in the command

Usage:

    There are two usages:

    1) ossutil ls [oss://] [-s] [--limited-num num] [--marker marker]
        If you list without a cloud_url, ossutil lists all the buckets using the credentials
    in config file with last modified time and location in addition. --show_format option 
    will ignore last modified time and location. The usage do not support --directory 
    option.

    2) ossutil ls oss://bucket[/prefix] [-s] [-d] [-m] [-a] [--limited-num num] [--marker marker] [--upload-id-marker umarker] [--version-id-marker id_marker] [--all-versions]
        If you list without --multipart and --all-type option, ossutil will list objects 
    in the specified bucket(with the prefix if you specified), with object size, last 
    modified time and etag in addition, --short-format option ignores all the additional 
    information. --directory option returns top-level subdirectory names instead of contents 
    of the subdirectory, which in default show by short format. the directory is end with /. 
        --multipart option will show multipart upload tasks under the cloud_url(oss://bucket[/prefix]), 
    which means, ossutil will show the uploadId of those uncompleted multipart, whose object 
    name starts with the specified prefix. ossutil will show the init time of uploadId meanwhile. 
    The usage also supports --short-format and --directory option. (Multipart upload is also used 
    in resume cp. More information about multipart see: https://help.aliyun.com/document_detail/31991.html?spm=5176.doc31992.6.880.VOSDk5). 
        --all-type option will show objects and multipart upload tasks under the cloud_url(oss://bucket[/prefix]),  
    which means, ossutil will both show the objects with the specified prefix and the uploadId of 
    those uncompleted multipart, whose object name starts with the specified prefix. The usage also 
    support --short-format and --directory option.
        If user specified --limited-num option, the total num will not exceed the num. If user list 
    objects and Multipart Uploads meanwhile, the total num of objects and Multipart Uploads will not 
    exceed the num. 
        --upload-id-marker option is not effective when list objects. When list Multipart Uploads, 
    --marker option and --upload-id-marker option decide the initial position of listing meanwhile,
    for more initial, see: https://help.aliyun.com/document_detail/31997.html?spm=5176.doc31965.6.887.MK6GVw.
`,

	sampleText: ` 
    1) ossutil ls -s
        oss://bucket1
        oss://bucket2
        oss://bucket3
        Bucket Number is: 3

    2) ossutil ls oss:// -s
        oss://bucket1
        oss://bucket2
        oss://bucket3
        Bucket Number is: 3

    3) ossutil ls oss://bucket1 -s
        oss://bucket1/dir1/obj11
        oss://bucket1/obj1
        oss://bucket1/sample.txt
        Object Number is: 3

    4) ossutil ls oss://bucket1
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:06:29 +0000 CST  201933      Standard   7E2F4A7F1AC9D2F0996E8332D5EA5B41  oss://bucket1/dir1/obj11
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        2016-04-08 14:50:47 +0000 CST 6476984      Standard   4F16FDAE7AC404CEC8B727FCC67779D6  oss://bucket1/sample.txt
        Object Number is: 3

    5) ossutil ls oss://bucket1 -d
        oss://bucket1/obj1
        oss://bucket1/dir1
        oss://bucket1/sample.txt
        Object and Directory Number is: 3

    6) ossutil ls oss://bucket1 -m 
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        2017-01-20 11:16:21 +0800 CST  A20157A7B2FEC4670626DAE0F4C0073C  oss://bucket1/tobj
        UploadID Number is: 3
    
    7) ossutil ls oss://bucket1/obj -m 
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 2
 
    8) ossutil ls oss://bucket1 -a 
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:06:29 +0000 CST  201933      Standard   7E2F4A7F1AC9D2F0996E8332D5EA5B41  oss://bucket1/dir1/obj11
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        2016-04-08 14:50:47 +0000 CST 6476984      Standard   4F16FDAE7AC404CEC8B727FCC67779D6  oss://bucket1/sample.txt
        Object Number is: 3
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:43:13 +0000 CST  2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        2017-01-20 11:16:21 +0800 CST  A20157A7B2FEC4670626DAE0F4C0073C  oss://bucket1/tobj
        UploadID Number is: 4
         
    9) ossutil ls oss://bucket1/obj -a 
        LastModifiedTime              Size(B)  StorageClass   ETAG                              ObjectName
        2015-06-05 14:36:21 +0000 CST  201933      Standard   6185CA2E8EB8510A61B3A845EAFE4174  oss://bucket1/obj1
        Object Number is: 1
        InitiatedTime                  UploadID                          ObjectName
        2017-01-13 03:45:26 +0000 CST  15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2017-01-13 03:43:13 +0000 CST  2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        2017-01-13 03:45:25 +0000 CST  3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 3

    10) ossutil ls oss://bucket1/obj -a -s 
        oss://bucket1/obj1
        Object Number is: 1
        UploadID                          ObjectName
        15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        2A1F9B4A95E341BD9285CC42BB950EE0  oss://bucket1/obj1
        3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 3

    11) ossutil ls oss://bucket1/obj -a -s --marker=obj1 
        Object Number is: 0
        UploadID                          ObjectName
        3998971ACAF94AD9AC48EAC1988BE863  oss://bucket1/obj2
        UploadID Number is: 1

    12) ossutil ls oss://bucket1/obj -a -s --limited-num=2 
        oss://bucket1/obj1
        Object Number is: 1
        UploadID                          ObjectName
        15754AF7980C4DFB8193F190837520BB  oss://bucket1/obj1
        UploadID Number is: 1

    13) ossutil ls oss://bucket1/%e4%b8%ad%e6%96%87 --encoding-type url
        LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
        2017-03-17 17:34:40 +0800 CST      8345742      Standard   BBCC8C0954B869B4A6B34D9404C5BCFD      oss://bucket1/中文
        Object Number is: 1
        0.066567(s) elapsed
    
    14) ossutil ls oss://bucket --include "*.avi" --include "*.mp4" --exclude "*.png" --exclude "*.jpg"
        LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
        2019-05-30 14:23:51 +0800 CST         1030      Standard   4A902D176BE0EE4224BC196BBB8CCC69      oss://bucket/test.avi
        2019-05-30 14:24:05 +0800 CST         1030      Standard   4A902D176BE0EE4224BC196BBB8CCC69      oss://bucket/test.mp4
        Object Number is: 2
    15) ossutil ls oss://bucket[/prefix] --all-versions
`,
}

// ListCommand is the command list buckets or objects
type ListCommand struct {
	command     Command
	payerOption oss.Option
	filters     []filterOptionType
}

var listCommand = ListCommand{
	command: Command{
		name:        "ls",
		nameAlias:   []string{"list"},
		minArgc:     0,
		maxArgc:     1,
		specChinese: specChineseList,
		specEnglish: specEnglishList,
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
			OptionRetryTimes,
			OptionLogLevel,
			OptionRequestPayer,
			OptionShortFormat,
			OptionDirectory,
			OptionMultipart,
			OptionAllType,
			OptionLimitedNum,
			OptionMarker,
			OptionUploadIDMarker,
			OptionEncodingType,
			OptionInclude,
			OptionExclude,
			OptionAllversions,
			OptionVersionIdMarker,
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
func (lc *ListCommand) formatHelpForWhole() string {
	return lc.command.formatHelpForWhole()
}

func (lc *ListCommand) formatIndependHelp() string {
	return lc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (lc *ListCommand) Init(args []string, options OptionMapType) error {
	return lc.command.Init(args, options, lc)
}

// RunCommand simulate inheritance, and polymorphism
func (lc *ListCommand) RunCommand() error {
	if len(lc.command.args) == 0 {
		return lc.listBuckets("")
	}

	encodingType, _ := GetString(OptionEncodingType, lc.command.options)
	cloudURL, err := CloudURLFromString(lc.command.args[0], encodingType)
	if err != nil {
		return err
	}

	payer, _ := GetString(OptionRequestPayer, lc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		} else {
			lc.payerOption = oss.RequestPayer(oss.PayerType(payer))
		}
	}

	if cloudURL.bucket == "" {
		return lc.listBuckets("")
	}

	var res bool
	res, lc.filters = getFilter(os.Args)
	if !res {
		return fmt.Errorf("--include or --exclude does not support format containing dir info")
	}

	return lc.listFiles(cloudURL)
}

func (lc *ListCommand) listBuckets(prefix string) error {
	var err error
	if err = lc.lbCheckArgOptions(); err != nil {
		return err
	}

	shortFormat, _ := GetBool(OptionShortFormat, lc.command.options)
	limitedNum, _ := GetInt(OptionLimitedNum, lc.command.options)
	vmarker, _ := GetString(OptionMarker, lc.command.options)
	if vmarker, err = lc.command.getRawMarker(vmarker); err != nil {
		return fmt.Errorf("invalid marker: %s, marker is not url encoded, %s", vmarker, err.Error())
	}

	var num int64
	num = 0

	client, err := lc.command.ossClient("")
	if err != nil {
		return err
	}

	// list all buckets
	pre := oss.Prefix(prefix)
	marker := oss.Marker(vmarker)
	payer := lc.payerOption
	for limitedNum < 0 || num < limitedNum {
		lbr, err := lc.ossListBucketsRetry(client, pre, marker, payer)
		if err != nil {
			return err
		}
		pre = oss.Prefix(lbr.Prefix)
		marker = oss.Marker(lbr.NextMarker)
		if num == 0 && !shortFormat && len(lbr.Buckets) > 0 {
			fmt.Printf("%-30s %20s%s%12s%s%s\n", "CreationTime", "Region", FormatTAB, "StorageClass", FormatTAB, "BucketName")
		}
		for _, bucket := range lbr.Buckets {
			if limitedNum >= 0 && num >= limitedNum {
				break
			}
			if !shortFormat {
				fmt.Printf("%-30s %20s%s%12s%s%s\n", utcToLocalTime(bucket.CreationDate), bucket.Location, FormatTAB, bucket.StorageClass, FormatTAB, CloudURLToString(bucket.Name, ""))
			} else {
				fmt.Println(CloudURLToString(bucket.Name, ""))
			}
			num++
		}
		if !lbr.IsTruncated {
			break
		}
	}
	fmt.Printf("Bucket Number is: %d\n", num)
	return nil
}

func (lc *ListCommand) lbCheckArgOptions() error {
	if ok, _ := GetBool(OptionDirectory, lc.command.options); ok {
		return fmt.Errorf("ListBucket does not support option: \"%s\"", OptionDirectory)
	}
	return nil
}

func (lc *ListCommand) ossListBucketsRetry(client *oss.Client, options ...oss.Option) (oss.ListBucketsResult, error) {
	retryTimes, _ := GetInt(OptionRetryTimes, lc.command.options)
	for i := 1; ; i++ {
		lbr, err := client.ListBuckets(options...)
		if err == nil || int64(i) >= retryTimes {
			return lbr, err
		}
	}
}

func (lc *ListCommand) listFiles(cloudURL CloudURL) error {
	bucket, err := lc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	shortFormat, _ := GetBool(OptionShortFormat, lc.command.options)
	directory, _ := GetBool(OptionDirectory, lc.command.options)
	limitedNum, _ := GetInt(OptionLimitedNum, lc.command.options)
	allVersions, _ := GetBool(OptionAllversions, lc.command.options)
	typeSet := lc.getSubjectType()
	if typeSet&objectType != 0 {
		if !allVersions {
			_, err = lc.listObjects(bucket, cloudURL, shortFormat, directory, &limitedNum)
		} else {
			_, err = lc.listObjectVersions(bucket, cloudURL, shortFormat, directory, &limitedNum)
		}

		if err != nil {
			return err
		}
	}
	if typeSet&multipartType != 0 {
		if _, err := lc.listMultipartUploads(bucket, cloudURL, shortFormat, directory, &limitedNum); err != nil {
			return err
		}
	}
	return nil
}

func (lc *ListCommand) getSubjectType() int64 {
	var typeSet int64
	typeSet = 0
	if isMultipart, _ := GetBool(OptionMultipart, lc.command.options); isMultipart {
		typeSet |= multipartType
	}
	if isAllType, _ := GetBool(OptionAllType, lc.command.options); isAllType {
		typeSet |= allType
	}
	if typeSet&allType == 0 {
		typeSet = objectType
	}
	return typeSet
}

func (lc *ListCommand) listObjects(bucket *oss.Bucket, cloudURL CloudURL, shortFormat bool, directory bool, limitedNum *int64) (int64, error) {
	//list all objects or directories
	var err error
	var num int64
	num = 0
	pre := oss.Prefix(cloudURL.object)
	vmarker, _ := GetString(OptionMarker, lc.command.options)
	if vmarker, err = lc.command.getRawMarker(vmarker); err != nil {
		return num, fmt.Errorf("invalid marker: %s, marker is not url encoded, %s", vmarker, err.Error())
	}
	marker := oss.Marker(vmarker)
	del := oss.Delimiter("")
	if directory {
		del = oss.Delimiter("/")
	}
	payer := lc.payerOption

	var i int64
	for i = 0; ; i++ {
		if *limitedNum == 0 {
			break
		}
		lor, err := lc.command.ossListObjectsRetry(bucket, marker, pre, del, payer, oss.MaxKeys(1000))
		if err != nil {
			return num, err
		}
		pre = oss.Prefix(lor.Prefix)
		marker = oss.Marker(lor.NextMarker)
		num += lc.displayObjectsResult(lor, cloudURL.bucket, shortFormat, directory, i, limitedNum)
		if !lor.IsTruncated {
			break
		}
	}

	if !directory {
		fmt.Printf("Object Number is: %d\n", num)
	} else {
		fmt.Printf("Object and Directory Number is: %d\n", num)
	}

	return num, nil
}

func (lc *ListCommand) listObjectVersions(bucket *oss.Bucket, cloudURL CloudURL, shortFormat bool, directory bool, limitedNum *int64) (int64, error) {
	//list all object versions or directories
	var err error
	var num int64
	num = 0
	pre := oss.Prefix(cloudURL.object)
	vmarker, _ := GetString(OptionMarker, lc.command.options)
	if vmarker, err = lc.command.getRawMarker(vmarker); err != nil {
		return num, fmt.Errorf("invalid marker: %s, marker is not url encoded, %s", vmarker, err.Error())
	}
	marker := oss.KeyMarker(vmarker)

	strVersionIdMarker, _ := GetString(OptionVersionIdMarker, lc.command.options)
	if strVersionIdMarker, err = lc.command.getRawMarker(strVersionIdMarker); err != nil {
		return num, fmt.Errorf("invalid versionIdMarker: %s, versionIdMarker is not url encoded, %s", strVersionIdMarker, err.Error())
	}
	versionIdMarker := oss.VersionIdMarker(strVersionIdMarker)

	del := oss.Delimiter("")
	if directory {
		del = oss.Delimiter("/")
	}
	payer := lc.payerOption

	var i int64
	for i = 0; ; i++ {
		if *limitedNum == 0 {
			break
		}
		lor, err := bucket.ListObjectVersions(marker, pre, del, payer, versionIdMarker)
		if err != nil {
			return num, err
		}
		pre = oss.Prefix(lor.Prefix)
		marker = oss.KeyMarker(lor.NextKeyMarker)
		versionIdMarker = oss.VersionIdMarker(lor.NextVersionIdMarker)
		num += lc.displayObjectVersionsResult(lor, cloudURL.bucket, shortFormat, directory, i, limitedNum)
		if !lor.IsTruncated {
			break
		}
	}

	if !directory {
		fmt.Printf("Object Number is: %d\n", num)
	} else {
		fmt.Printf("Object and Directory Number is: %d\n", num)
	}
	return num, nil
}

func (lc *ListCommand) displayObjectsResult(lor oss.ListObjectsResult, bucket string, shortFormat bool, directory bool, i int64, limitedNum *int64) int64 {
	if i == 0 && !shortFormat && !directory && len(lor.Objects) > 0 {
		fmt.Printf("%-30s%12s%s%12s%s%-36s%s%s\n", "LastModifiedTime", "Size(B)", "  ", "StorageClass", "   ", "ETAG", "  ", "ObjectName")
	}

	var num int64
	if !directory {
		num = lc.showObjects(lor, bucket, shortFormat, limitedNum)
	} else {
		num = lc.showObjects(lor, bucket, true, limitedNum)
		num1 := lc.showDirectories(lor, bucket, limitedNum)
		num += num1
	}
	return num
}

func (lc *ListCommand) displayObjectVersionsResult(lor oss.ListObjectVersionsResult, bucket string, shortFormat bool, directory bool, i int64, limitedNum *int64) int64 {
	if i == 0 && (len(lor.ObjectDeleteMarkers) > 0 || len(lor.ObjectVersions) > 0) {
		if directory {
			fmt.Printf("%-6s%s%-30s%12s%s%12s%s%-36s%s%-66s%s%-10s%s%-13s%s%s\n", "COMMON-PREFIX", "  ", "LastModifiedTime", "Size(B)", "  ", "StorageClass", "  ", "ETAG", "  ", "VERSIONID", "  ", "IS-LATEST", "  ", "DELETE-MARKER", "  ", "ObjectName")
		} else {
			fmt.Printf("%-30s%12s%s%12s%s%-36s%s%-66s%s%-10s%s%-13s%s%s\n", "LastModifiedTime", "Size(B)", "  ", "StorageClass", "  ", "ETAG", "  ", "VERSIONID", "  ", "IS-LATEST", "  ", "DELETE-MARKER", "  ", "ObjectName")
		}
	}

	var num int64
	num = lc.showObjectVersions(lor, bucket, limitedNum, directory)
	if directory {
		num1 := lc.showDirectoriesVersion(lor, bucket, limitedNum)
		num += num1
	}
	return num
}

func (lc *ListCommand) showObjects(lor oss.ListObjectsResult, bucket string, shortFormat bool, limitedNum *int64) int64 {
	var num int64
	num = 0
	for _, object := range lor.Objects {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(object.Key, lc.filters) {
			continue
		}

		if !shortFormat {
			fmt.Printf("%-30s%12d%s%12s%s%-36s%s%s\n", utcToLocalTime(object.LastModified), object.Size, "  ", object.StorageClass, "   ", strings.Trim(object.ETag, "\""), "  ", CloudURLToString(bucket, object.Key))
		} else {
			fmt.Printf("%s\n", CloudURLToString(bucket, object.Key))
		}
		*limitedNum--
		num++
	}
	return num
}

func (lc *ListCommand) showObjectVersions(lor oss.ListObjectVersionsResult, bucket string, limitedNum *int64, directory bool) int64 {
	var num int64
	num = 0
	for _, object := range lor.ObjectDeleteMarkers {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(object.Key, lc.filters) {
			continue
		}

		//COMMON-PREFIX LastModifiedTime  Size(B)  StorageClass  ETAG VERSIONID  IS-LATEST  DELETE-MARKER  ObjectName
		if directory {
			fmt.Printf("%-13t%s%-30s%12d%s%12s%s%-36s%s%-66s%s%-10t%s%-13t%s%s\n",
				false, "  ",
				utcToLocalTime(object.LastModified),
				0, "  ",
				"", "  ",
				"", "  ",
				object.VersionId, "  ",
				object.IsLatest, "  ",
				true, "  ",
				CloudURLToString(bucket, object.Key))
		} else {
			fmt.Printf("%-30s%12d%s%12s%s%-36s%s%-66s%s%-10t%s%-13t%s%s\n",
				utcToLocalTime(object.LastModified),
				0, "  ",
				"", "  ",
				"", "  ",
				object.VersionId, "  ",
				object.IsLatest, "  ",
				true, "  ",
				CloudURLToString(bucket, object.Key))
		}

		*limitedNum--
		num++
	}

	for _, object := range lor.ObjectVersions {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(object.Key, lc.filters) {
			continue
		}

		//COMMON-PREFIX LastModifiedTime  Size(B)  StorageClass  ETAG VERSIONID  IS-LATEST  DELETE-MARKER  ObjectName
		if directory {
			fmt.Printf("%-13t%s%-30s%12d%s%12s%s%-36s%s%-66s%s%-10t%s%-13t%s%s\n",
				false, "  ",
				utcToLocalTime(object.LastModified),
				object.Size, "  ",
				object.StorageClass, "  ",
				strings.Trim(object.ETag, "\""), "  ",
				object.VersionId, "  ",
				object.IsLatest, "  ",
				false, "  ",
				CloudURLToString(bucket, object.Key))
		} else {
			fmt.Printf("%-30s%12d%s%12s%s%-36s%s%-66s%s%-10t%s%-13t%s%s\n",
				utcToLocalTime(object.LastModified),
				object.Size, "  ",
				object.StorageClass, "  ",
				strings.Trim(object.ETag, "\""), "  ",
				object.VersionId, "  ",
				object.IsLatest, "  ",
				false, "  ",
				CloudURLToString(bucket, object.Key))
		}

		*limitedNum--
		num++
	}

	return num
}

func (lc *ListCommand) showDirectories(lor oss.ListObjectsResult, bucket string, limitedNum *int64) int64 {
	var num int64
	num = 0
	for _, prefix := range lor.CommonPrefixes {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(strings.TrimSuffix(prefix, "/"), lc.filters) {
			continue
		}

		fmt.Printf("%s\n", CloudURLToString(bucket, prefix))
		*limitedNum--
		num++
	}
	return num
}

func (lc *ListCommand) showDirectoriesVersion(lor oss.ListObjectVersionsResult, bucket string, limitedNum *int64) int64 {
	var num int64
	num = 0
	for _, prefix := range lor.CommonPrefixes {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(strings.TrimSuffix(prefix, "/"), lc.filters) {
			continue
		}

		fmt.Printf("%-13t%s%-30s%12s%s%12s%s%-36s%s%-66s%s%-10s%s%-13s%s%s\n",
			true, "  ",
			"", "", "  ",
			"", "  ",
			"", "  ",
			"", "  ",
			"", "  ",
			"", "  ",
			CloudURLToString(bucket, prefix))

		*limitedNum--
		num++
	}
	return num
}

func (lc *ListCommand) listMultipartUploads(bucket *oss.Bucket, cloudURL CloudURL, shortFormat bool, directory bool, limitedNum *int64) (int64, error) {
	var err error
	var multipartNum int64
	multipartNum = 0
	pre := oss.Prefix(cloudURL.object)

	vmarker, _ := GetString(OptionMarker, lc.command.options)
	if vmarker, err = lc.command.getRawMarker(vmarker); err != nil {
		return multipartNum, fmt.Errorf("invalid marker: %s, marker is not url encoded, %s", vmarker, err.Error())
	}
	keyMarker := oss.KeyMarker(vmarker)

	vuploadIdMarker, _ := GetString(OptionUploadIDMarker, lc.command.options)
	if vuploadIdMarker, err = lc.command.getRawMarker(vuploadIdMarker); err != nil {
		return multipartNum, fmt.Errorf("invalid uploadIDMarker: %s, uploadIDMarker is not url encoded, %s", vuploadIdMarker, err.Error())
	}
	uploadIdMarker := oss.UploadIDMarker(vuploadIdMarker)

	del := oss.Delimiter("")
	if directory {
		del = oss.Delimiter("/")
	}
	payer := lc.payerOption

	var i int64
	for i = 0; ; i++ {
		if *limitedNum == 0 {
			break
		}
		lmr, err := lc.command.ossListMultipartUploadsRetry(bucket, keyMarker, uploadIdMarker, pre, del, payer)
		if err != nil {
			return multipartNum, err
		}
		pre = oss.Prefix(lmr.Prefix)
		keyMarker = oss.Marker(lmr.NextKeyMarker)
		uploadIdMarker = oss.UploadIDMarker(lmr.NextUploadIDMarker)
		multipartNum += lc.displayMultipartUploadsResult(lmr, cloudURL.bucket, shortFormat, directory, i, limitedNum)
		if !lmr.IsTruncated {
			break
		}
	}
	fmt.Printf("UploadID Number is: %d\n", multipartNum)
	return multipartNum, nil
}

func (lc *ListCommand) displayMultipartUploadsResult(lmr oss.ListMultipartUploadResult, bucket string, shortFormat bool, directory bool, i int64, limitedNum *int64) int64 {
	if directory {
		shortFormat = true
	}

	if i == 0 && len(lmr.Uploads) > 0 {
		if shortFormat {
			fmt.Printf("%-32s%s%s\n", "UploadID", FormatTAB, "ObjectName")
		} else {
			fmt.Printf("%-30s%s%-32s%s%s\n", "InitiatedTime", FormatTAB, "UploadID", FormatTAB, "ObjectName")
		}
	}

	num := lc.showMultipartUploads(lmr, bucket, shortFormat, limitedNum)
	return num
}

func (lc *ListCommand) showMultipartUploads(lmr oss.ListMultipartUploadResult, bucket string, shortFormat bool, limitedNum *int64) int64 {
	var num int64
	num = 0
	for _, upload := range lmr.Uploads {
		if *limitedNum == 0 {
			break
		}

		if !doesSingleObjectMatchPatterns(upload.Key, lc.filters) {
			continue
		}

		if shortFormat {
			fmt.Printf("%-32s%s%s\n", upload.UploadID, FormatTAB, CloudURLToString(bucket, upload.Key))
		} else {
			fmt.Printf("%-30s%s%-32s%s%s\n", utcToLocalTime(upload.Initiated), FormatTAB, upload.UploadID, FormatTAB, CloudURLToString(bucket, upload.Key))
		}
		*limitedNum--
		num++
	}
	return num
}
