package lib

import (
	"bufio"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/syndtr/goleveldb/leveldb"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var headerOptionMap = map[string]interface{}{
	oss.HTTPHeaderContentType:                  oss.ContentType,
	oss.HTTPHeaderCacheControl:                 oss.CacheControl,
	oss.HTTPHeaderContentDisposition:           oss.ContentDisposition,
	oss.HTTPHeaderContentEncoding:              oss.ContentEncoding,
	oss.HTTPHeaderExpires:                      oss.Expires,
	oss.HTTPHeaderAcceptEncoding:               oss.AcceptEncoding,
	oss.HTTPHeaderOssServerSideEncryption:      oss.ServerSideEncryption,
	oss.HTTPHeaderOssObjectACL:                 oss.ObjectACL,
	oss.HTTPHeaderOrigin:                       oss.Origin,
	oss.HTTPHeaderOssStorageClass:              oss.ObjectStorageClass,
	oss.HTTPHeaderOssServerSideEncryptionKeyID: oss.ServerSideEncryptionKeyID,
	oss.HTTPHeaderOssServerSideDataEncryption:  oss.ServerSideDataEncryption,
	oss.HTTPHeaderSSECAlgorithm:                oss.SSECAlgorithm,
	oss.HTTPHeaderSSECKey:                      oss.SSECKey,
	oss.HTTPHeaderSSECKeyMd5:                   oss.SSECKeyMd5,
}

func formatHeaderString(hopMap map[string]interface{}, sep string) string {
	str := ""
	for header := range hopMap {
		if header == oss.HTTPHeaderExpires {
			str += header + fmt.Sprintf("(time.RFC3339: %s)", time.RFC3339) + sep
		} else {
			str += header + sep
		}
	}
	if len(str) >= len(sep) {
		str = str[:len(str)-len(sep)]
	}
	return str
}

func fetchHeaderOptionMap(hopMap map[string]interface{}, name string) (interface{}, error) {
	for header, f := range hopMap {
		if strings.ToLower(name) == strings.ToLower(header) {
			return f, nil
		}
	}
	return nil, fmt.Errorf("unsupported header: %s, please check", name)
}

func getOSSOption(hopMap map[string]interface{}, name string, param string) (oss.Option, error) {
	if f, err := fetchHeaderOptionMap(hopMap, name); err == nil {
		switch f.(type) {
		case func(string) oss.Option:
			return f.(func(string) oss.Option)(param), nil
		case func(oss.ACLType) oss.Option:
			return f.(func(oss.ACLType) oss.Option)(oss.ACLType(param)), nil
		case func(t time.Time) oss.Option:
			val, err := time.Parse(http.TimeFormat, param)
			if err != nil {
				val, err = time.Parse(time.RFC3339, param)
				if err != nil {
					return nil, err
				}
			}
			return f.(func(time.Time) oss.Option)(val), nil
		case func(oss.StorageClassType) oss.Option:
			return f.(func(oss.StorageClassType) oss.Option)(oss.StorageClassType(param)), nil
		default:
			return nil, fmt.Errorf("error option type, internal error")
		}
	}
	return nil, fmt.Errorf("unsupported header: %s, please check", name)
}

var specChineseSetMeta = SpecText{

	synopsisText: "设置已上传的objects的元信息",

	paramText: "cloud_url [meta] [options]",

	syntaxText: ` 
    ossutil set-meta oss://bucket[/prefix] [header:value#header:value...] [--update] [--delete] [-r] [-f] [-c file] [--version-id versionId] [--object-file file] [--snapshot-path dir] [--disable-ignore-error]
`,

	detailHelpText: ` 
    该命令可设置或者更新或者删除指定objects的meta信息。当指定--recursive选项时，ossutil
    获取所有与指定cloud_url匹配的objects，批量设置这些objects的meta，否则，设置指定的单个
    object的元信息，如果该object不存在，ossutil会报错。

    （1）设置全量值：如果用户未指定--update选项和--delete选项，ossutil会设置指定objects的
        meta为用户输入的[header:value#header:value...]。当缺失[header:value#header:value...]
        信息时，相当于删除全部meta信息（对于不可删除的headers，即：不以` + oss.HTTPHeaderOssMetaPrefix + `开头的headers，
        其值不会改变）。此时ossutil会进入交互模式并要求用户确认meta信息。

    （2）更新meta：如果用户设置--update选项，ossutil会更新指定objects的指定header为输入
        的value值，其中value可以为空，指定objects的其他meta信息不会改变。此时不支持--delete
        选项。

    （3）删除meta：如果用户设置--delete选项，ossutil会删除指定objects的指定header（对于不可
        删除的headers，即：不以` + oss.HTTPHeaderOssMetaPrefix + `开头的headers，该选项不起作用），该此时value必须
        为空（header:或者header），指定objects的其他meta信息不会改变。此时不支持--update选项。

    该命令不支持bucket的meta设置，需要设置bucket的meta信息，请使用bucket相关操作。
    查看bucket或者object的meta信息，请使用stat命令。

Headers:

    可选的header列表如下：
        ` + formatHeaderString(headerOptionMap, "\n        ") + `
        以及以` + oss.HTTPHeaderOssMetaPrefix + `开头的header

    注意：header不区分大小写，但value区分大小写。

用法：

    该命令有两种用法：

    1) ossutil set-meta oss://bucket/object [header:value#header:value...] [--update] [--delete] [-f] [--version-id versionId]
        如果未指定--recursive选项，ossutil设置指定的单个object的meta信息，此时请确保输入
    的cloud_url精确指定了想要设置meta的object，当object不存在时会报错。如果指定了--force
    选项，则不会进行询问提示。如果用户未输入[header:value#header:value...]，相当于删除
    object的所有meta。
        --update选项和--delete选项的用法参考上文。

    2) ossutil set-meta oss://bucket[/prefix] [header:value#header:value...] -r [--update] [--delete] [-f]
        如果指定了--recursive选项，ossutil会查找所有前缀匹配cloud_url的objects，批量设置
    这些objects的meta信息。当一个object操作出现错误时会将出错object的错误信息记录到report
    文件，并继续操作其他object，成功操作的object信息将不会被记录到report文件中（更多信息
    见cp命令的帮助）。
        如果指定了--include/--exclude选项，ossutil会查找所有匹配pattern的objects，批量设置。
        --include和--exclude选项说明，请参考cp命令帮助。
        如果--force选项被指定，则不会进行询问提示。
        --update选项和--delete选项的用法参考上文。

    3) ossutil set-meta oss://bucket [header:value#header:value...] --object-file file [--snapshot-path dir] [--disable-ignore-error] [--update] [--delete] [-f]
        如果指定了--object-file选项，ossutil会读取指定文件中的所有objects，批量设置
    这些objects的meta信息。当一个object操作出现错误时会将出错object的错误信息记录到report
    文件，并继续操作其他object，成功操作的object信息将不会被记录到report文件中（更多信息
    见cp命令的帮助）。
        如果--snapshot-path选项被指定，则会对本次操作的object进行快照，如果操作对象已经存在
        快照，则忽略本次操作。（仅支持在-r、--object-file基础上）
        如果--force选项被指定，则不会进行询问提示。
        --update选项和--delete选项的用法参考上文。
`,

	sampleText: ` 
    (1)ossutil set-meta oss://bucket1/obj1 Cache-Control:no-cache#Content-Encoding:gzip#X-Oss-Meta-a:b
        设置obj1的Cache-Control，Content-Encoding和X-Oss-Meta-a头域

    (2)ossutil set-meta oss://bucket1/o X-Oss-Meta-empty:#Content-Type:plain/text --update -r
        批量更新以o开头的objects的X-Oss-Meta-empty和Content-Type头域

    (3)ossutil set-meta oss://bucket1/ X-Oss-Meta-empty:#Content-Type:plain/text --update -r --include "*.jpg"
        批量更新后缀为.jpg的objects的X-Oss-Meta-empty和Content-Type头域

    (4)ossutil set-meta oss://bucket1/o X-Oss-Meta-empty:#Content-Type:plain/text --update -r --exclude "*.jpg"
        批量更新以o开头后缀为.jpg的objects的X-Oss-Meta-empty和Content-Type头域

    (5)ossutil set-meta oss://bucket1/obj1 X-Oss-Meta-delete --delete
        删除obj1的X-Oss-Meta-delete头域

    (6)ossutil set-meta oss://bucket/o -r
        批量设置以o开头的objects的meta为空

    (7)ossutil set-meta oss://bucket1/%e4%b8%ad%e6%96%87 X-Oss-Meta-delete --delete --encoding-type url
        删除oss://bucket1/中文的X-Oss-Meta-delete头域

    (8)ossutil set-meta oss://bucket1/obj1 X-Oss-Meta-delete --delete --version-id versionId
        删除指定版本obj1的X-Oss-Meta-delete头域，并生成最新版本

    (9)ossutil set-meta oss://bucket1 X-Oss-Meta-empty:#Content-Type:plain/text --update --object-file file
        批量更新file文件中所有objects的X-Oss-Meta-empty和Content-Type头域

    (10)ossutil set-meta oss://bucket1 X-Oss-Meta-empty:#Content-Type:plain/text --update --object-file file --snapshot-path dir
        批量更新file文件中所有objects的X-Oss-Meta-empty和Content-Type头域，并开启快照
`,
}

var specEnglishSetMeta = SpecText{

	synopsisText: "set metadata on already uploaded objects",

	paramText: "cloud_url [meta] [options]",

	syntaxText: ` 
    ossutil set-meta oss://bucket[/prefix] [header:value#header:value...] [--update] [--delete] [-r] [-f] [-c file] [--version-id versionId] [--object-file file] [--snapshot-path dir] [--disable-ignore-error]
`,

	detailHelpText: ` 
    The command can be used to set, update or delete the specified objects' meta data. 
    If --recursive option is specified, ossutil find all matching objects and batch set 
    meta on these objects, else, ossutil set meta on single object, if the object not 
    exist, error happens. 

    (1) Set full meta: If --update option and --delete option is not specified, ossutil 
        will set the meta of the specified objects to [header:value#header:value...], what
        user inputs. If [header:value#header:value...] is missing, it means clear the meta 
        data of the specified objects(to those headers which can not be deleted, that is, 
        the headers do not start with: ` + oss.HTTPHeaderOssMetaPrefix + `, the value will not be changed), at the 
        time ossutil will ask user to confirm the input.

    (2) Update meta: If --update option is specified, ossutil will update the specified 
        headers of objects to the values that user inputs(the values can be empty), other 
        meta data of the specified objects will not be changed. --delete option is not 
        supported in the usage. 

    (3) Delete meta: If --delete option is specified, ossutil will delete the specified 
        headers of objects that user inputs(to those headers which can not be deleted, 
        that is, the headers do not start with: ` + oss.HTTPHeaderOssMetaPrefix + `, the value will not be changed), 
        in this usage the value must be empty(like header: or header), other meta data 
        of the specified objects will not be changed. --update option is not supported 
        in the usage.

    The meta data of bucket can not be setted by the command, please use other commands. 
    User can use stat command to check the meta information of bucket or objects.

Headers:

    ossutil supports following headers:
        ` + formatHeaderString(headerOptionMap, "\n        ") + `
        and headers starts with: ` + oss.HTTPHeaderOssMetaPrefix + `

    Warning: headers are case-insensitive, but value are case-sensitive.

Usage:

    There are two usages:

    1) ossutil set-meta oss://bucket/object [header:value#header:value...] [--update] [--delete] [-f] [--version-id versionId]
        If --recursive option is not specified, ossutil set meta on the specified single 
    object. In the usage, please make sure cloud_url exactly specified the object you want to 
    set meta on, if object not exist, error occurs. If --force option is specified, ossutil 
    will not show prompt question. 
        The usage of --update option and --delete option is showed in detailHelpText. 

    2) ossutil set-meta oss://bucket[/prefix] [header:value#header:value...] -r [--update] [--delete] [-f]
        If --recursive option is specified, ossutil will search for prefix-matching objects 
    and set meta on these objects. If an error occurs, ossutil will record the error message 
    to report file, and ossutil will continue to attempt to set acl on the remaining objects(
    more information see help of cp command). 
        If --include/--exclude option is specified, ossutil will search for pattern-matching objects and 
    set meta on those objects. 
	    --include and --exclude option, please refer cp command help.
        If --force option is specified, ossutil will not show prompt question.
        The usage of --update option and --delete option is showed in detailHelpText.

    3) ossutil set-meta oss://bucket [header:value#header:value...] --object-file file [--snapshot-path dir] [--disable-ignore-error] [--update] [--delete] [-f]
        如果指定了--object-file选项，ossutil会读取指定文件中的所有objects，批量设置
    这些objects的meta信息。当一个object操作出现错误时会将出错object的错误信息记录到report
    文件，并继续操作其他object，成功操作的object信息将不会被记录到report文件中（更多信息
    见cp命令的帮助）。
        如果--snapshot-path选项被指定，则会对本次操作的object进行快照，如果操作对象已经存在
		快照，则忽略本次操作。（仅支持在-r、--object-file基础上）
        如果--force选项被指定，则不会进行询问提示。
        --update选项和--delete选项的用法参考上文。

		If --object-file option is specified, ossutil will read objects in file, then 
    set meta on these objects. If an error occurs, ossutil will record the error message 
    to report file, and ossutil will continue to attempt to set acl on the remaining objects(
    more information see help of cp command). 
        If --snapshot-path option is specified, ossutil will create snapshot for this operation, 
		and if the snapshot exists, then cancel this operate.
        If --force option is specified, ossutil will not show prompt question.
        The usage of --update option and --delete option is showed in detailHelpText.
`,

	sampleText: ` 
    (1)ossutil set-meta oss://bucket1/obj1 Cache-Control:no-cache#Content-Encoding:gzip#X-Oss-Meta-a:b
        Set Cache-Control, Content-Encoding and X-Oss-Meta-a header for obj1

    (2)ossutil set-meta oss://bucket1/o X-Oss-Meta-empty:#Content-Type:plain/text -u -r
        Batch update X-Oss-Meta-empty and Content-Type header on objects that start with o

    (3)ossutil set-meta oss://bucket1/ X-Oss-Meta-empty:#Content-Type:plain/text --update -r --include "*.jpg"
        Batch update X-Oss-Meta-empty and Content-Type header on objects ending with .jpg

    (4)ossutil set-meta oss://bucket1/o X-Oss-Meta-empty:#Content-Type:plain/text --update -r --exclude ".jpg"
        Batch update X-Oss-Meta-empty and Content-Type header on objects starting with o and ending with .jpg

    (5)ossutil set-meta oss://bucket1/obj1 X-Oss-Meta-delete -d
        Delete X-Oss-Meta-delete header of obj1 

    (6)ossutil set-meta oss://bucket/o -r
        Batch set the meta of objects that start with o to empty

    (7)ossutil set-meta oss://bucket1/%e4%b8%ad%e6%96%87 X-Oss-Meta-delete --delete --encoding-type url
        Delete X-Oss-Meta-delete header of oss://bucket1/中文
    
	(8)ossutil set-meta oss://bucket1/obj1 X-Oss-Meta-delete --delete --version-id versionId
        Delete X-Oss-Meta-delete header of a specific version of obj1，and generate the latest version obj1

    (9)ossutil set-meta oss://bucket1 X-Oss-Meta-empty:#Content-Type:plain/text --update --object-file file
        Batch update X-Oss-Meta-empty and Content-Type header on objects that in file

    (10)ossutil set-meta oss://bucket1 X-Oss-Meta-empty:#Content-Type:plain/text --update --object-file file --snapshot-path dir
        Batch update X-Oss-Meta-empty and Content-Type header on objects that in file, and open snapshot
`,
}

// SetMetaCommand is the command set meta for object
type SetMetaCommand struct {
	monitor     Monitor //Put first for atomic op on some fileds
	command     Command
	smOption    batchOptionType
	filters     []filterOptionType
	skipCount   uint64
	hasObjFile  bool
	objFilePath string
}

var setMetaCommand = SetMetaCommand{
	command: Command{
		name:        "set-meta",
		nameAlias:   []string{"setmeta", "set_meta"},
		minArgc:     1,
		maxArgc:     2,
		specChinese: specChineseSetMeta,
		specEnglish: specEnglishSetMeta,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			OptionRecursion,
			OptionUpdate,
			OptionDelete,
			OptionForce,
			OptionEncodingType,
			OptionInclude,
			OptionExclude,
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
			OptionLanguage,
			OptionOutputDir,
			OptionLogLevel,
			OptionVersionId,
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
func (sc *SetMetaCommand) formatHelpForWhole() string {
	return sc.command.formatHelpForWhole()
}

func (sc *SetMetaCommand) formatIndependHelp() string {
	return sc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (sc *SetMetaCommand) Init(args []string, options OptionMapType) error {
	return sc.command.Init(args, options, sc)
}

// RunCommand simulate inheritance, and polymorphism
func (sc *SetMetaCommand) RunCommand() error {
	sc.monitor.init("Setted meta on")
	isUpdate, _ := GetBool(OptionUpdate, sc.command.options)
	isDelete, _ := GetBool(OptionDelete, sc.command.options)
	recursive, _ := GetBool(OptionRecursion, sc.command.options)
	force, _ := GetBool(OptionForce, sc.command.options)
	routines, _ := GetInt(OptionRoutines, sc.command.options)
	language, _ := GetString(OptionLanguage, sc.command.options)
	language = strings.ToLower(language)
	encodingType, _ := GetString(OptionEncodingType, sc.command.options)
	versionId, _ := GetString(OptionVersionId, sc.command.options)
	objFileXml, _ := GetString(OptionObjectFile, sc.command.options)
	snapshotPath, _ := GetString(OptionSnapshotPath, sc.command.options)

	var err error
	// load snapshot
	sc.smOption.snapshotPath = snapshotPath
	if sc.smOption.snapshotPath != "" {
		if sc.smOption.snapshotldb, err = leveldb.OpenFile(sc.smOption.snapshotPath, nil); err != nil {
			return fmt.Errorf("load snapshot error, reason: %s", err.Error())
		}
		defer sc.smOption.snapshotldb.Close()
	}

	cloudURL, err := CloudURLFromString(sc.command.args[0], encodingType)
	if err != nil {
		return err
	}
	if err := sc.checkOptions(cloudURL, isUpdate, isDelete, force, recursive, language, versionId, objFileXml); err != nil {
		return err
	}
	bucket, err := sc.command.ossBucket(cloudURL.bucket)
	if err != nil {
		return err
	}

	str, err := sc.getMetaData(force, language)
	if err != nil {
		return err
	}
	headers, err := sc.command.parseHeaders(str, isDelete)
	if err != nil {
		return err
	}

	sc.smOption.ctnu = true

	// check --object-file mode
	if objFileXml != "" {
		// check objFileXml and parse it
		if err := sc.checkObjectFile(objFileXml); err != nil {
			return err
		}
		recursive = true
		err = sc.batchSetObjectsMetaFromFile(bucket, cloudURL, headers, isUpdate, isDelete, recursive, routines)
	} else {
		if !recursive {
			err = sc.setObjectMeta(bucket, cloudURL.object, headers, isUpdate, isDelete, false, versionId)
		} else {
			err = sc.batchSetObjectMeta(bucket, cloudURL, headers, isUpdate, isDelete, force, routines)
		}
	}

	if isUpdate {
		LogInfo("update skip count:%d\n", sc.skipCount)
	}
	return err
}

func (sc *SetMetaCommand) checkOptions(cloudURL CloudURL, isUpdate, isDelete, force, recursive bool, language, versionId, objFileXml string) error {
	if cloudURL.bucket == "" {
		return fmt.Errorf("invalid cloud url: %s, miss bucket", cloudURL.urlStr)
	}
	if cloudURL.object == "" {
		if !recursive && objFileXml == "" {
			return fmt.Errorf("set object meta invalid cloud url: %s, object empty. Set bucket meta is not supported, if you mean batch set meta on objects, please use --recursive or --object-file", sc.command.args[0])
		}
	} else {
		if objFileXml != "" {
			return fmt.Errorf("the first arg of `ossutil set-meta` only support oss://bucket when set option --object-file")
		}
	}

	var res bool
	res, sc.filters = getFilter(os.Args)
	if !res {
		return fmt.Errorf("--include or --exclude does not support format containing dir info")
	}

	if !recursive && len(sc.filters) > 0 {
		return fmt.Errorf("--include or --exclude only work with --recursive")
	}

	if (recursive && len(versionId) > 0) || (objFileXml != "" && len(versionId) > 0) {
		return fmt.Errorf("--version-id only work on single object")
	}

	if !force {
		var val string
		if !recursive && objFileXml == "" {
			return nil
		}
		fmt.Printf("Do you really mean to recursivlly set meta on objects of %s(y or N)? ", sc.command.args[0])
		if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
			fmt.Println("operation is canceled.")
			return nil
		}
	}

	if isUpdate && isDelete {
		return fmt.Errorf("--update option and --delete option are not supported for %s at the same time, please check", sc.command.args[0])
	}
	if !isUpdate && !isDelete && !force {
		if language == LEnglishLanguage {
			fmt.Printf("Warning: --update option means update the specified header, --delete option means delete the specified header, miss both options means update the whole meta info, continue to update the whole meta info(y or N)? ")
		} else {
			fmt.Printf("警告：--update选项更新指定的header，--delete选项删除指定的header，两者同时缺失会更改object的全量meta信息，请确认是否要更改全量meta信息(y or N)? ")
		}
		var str string
		if _, err := fmt.Scanln(&str); err != nil || (strings.ToLower(str) != "yes" && strings.ToLower(str) != "y") {
			return fmt.Errorf("operation is canceled")
		}
		fmt.Println("")
	}
	return nil
}

func (sc *SetMetaCommand) getMetaData(force bool, language string) (string, error) {
	if len(sc.command.args) > 1 {
		return strings.TrimSpace(sc.command.args[1]), nil
	}

	if force {
		return "", nil
	}

	if language == LEnglishLanguage {
		fmt.Printf("Do you really mean the empty meta(or forget to input header:value pair)? \nEnter yes(y) to continue with empty meta, enter no(n) to show supported headers, other inputs will cancel operation: ")
	} else {
		fmt.Printf("你是否确定你想设置的meta信息为空（或者忘记了输入header:value对）? \n输入yes(y)使用空meta继续设置，输入no(n)来展示支持的headers，其他输入将取消操作：")
	}
	var str string
	if _, err := fmt.Scanln(&str); err != nil || (strings.ToLower(str) != "yes" && strings.ToLower(str) != "y" && strings.ToLower(str) != "no" && strings.ToLower(str) != "n") {
		return "", fmt.Errorf("unknown input, operation is canceled")
	}
	if strings.ToLower(str) == "yes" || strings.ToLower(str) == "y" {
		return "", nil
	}

	if language == LEnglishLanguage {
		fmt.Printf("\nSupported headers:\n    %s\n    And the headers start with: \"%s\"\n\nPlease enter the header:value#header:value... pair you want to set: ", formatHeaderString(headerOptionMap, "\n    "), oss.HTTPHeaderOssMetaPrefix)
	} else {
		fmt.Printf("\n支持的headers:\n    %s\n    以及以\"%s\"开头的headers\n\n请输入你想设置的header:value#header:value...：", formatHeaderString(headerOptionMap, "\n    "), oss.HTTPHeaderOssMetaPrefix)
	}
	if _, err := fmt.Scanln(&str); err != nil {
		return "", fmt.Errorf("meta empty, please check, operation is canceled")
	}
	return strings.TrimSpace(str), nil
}

func (cmd *Command) parseHeaders(str string, isDelete bool) (map[string]string, error) {
	if str == "" {
		return nil, nil
	}

	headers := map[string]string{}
	sli := strings.Split(str, "#")
	for _, s := range sli {
		pair := strings.SplitN(s, ":", 2)
		name := pair[0]
		value := ""
		if len(pair) > 1 {
			value = pair[1]
		}
		if isDelete && value != "" {
			return nil, fmt.Errorf("delete meta for object do no support value for header:%s, please set value:%s to empty", name, value)
		}
		if _, err := fetchHeaderOptionMap(headerOptionMap, name); err != nil && !strings.HasPrefix(strings.ToLower(name), "x-oss-") {
			return nil, fmt.Errorf("unsupported header:%s, please try \"help %s\" to see supported headers", name, cmd.name)
		}
		headers[name] = value
	}
	return headers, nil
}

func (sc *SetMetaCommand) setObjectMeta(bucket *oss.Bucket, object string, headers map[string]string, isUpdate, isDelete, batchOperate bool, versionId string) error {
	allheaders := headers
	isSkip := false
	spath := ""
	msg := "set_meta"
	nowt := time.Now().Unix()

	if batchOperate && sc.smOption.snapshotPath != "" {
		spath = sc.formatSnapshotKey(bucket.BucketName, object, msg)
		if skip := sc.skipSetMeta(spath); skip {
			sc.updateSkip(1)
			LogInfo("restore obj skip: %s\n", object)
			return nil
		}
	}

	if isUpdate || isDelete {
		var options []oss.Option
		if len(versionId) > 0 {
			options = append(options, oss.VersionId(versionId))
		}

		// get object meta
		props, err := sc.command.ossGetObjectStatRetry(bucket, object, options...)
		if err != nil {
			return err
		}

		// get object acl
		objectACL, err := bucket.GetObjectACL(object, options...)
		if err != nil {
			return err
		}
		props.Set(StatACL, objectACL.ACL)

		// merge
		allheaders, isSkip = sc.mergeHeader(props, headers, isUpdate, isDelete)
		if isSkip {
			atomic.AddUint64(&sc.skipCount, uint64(1))
			return nil
		}
	}

	options, err := sc.command.getOSSOptions(headerOptionMap, allheaders)
	if err != nil {
		return err
	}
	if len(versionId) > 0 {
		options = append(options, oss.VersionId(versionId))
	}

	err = sc.ossSetObjectMetaRetry(bucket, object, options...)
	if batchOperate && sc.smOption.snapshotPath != "" {
		if err != nil {
			_ = sc.updateSnapshot(err, spath, nowt)
			return err
		} else {
			err = sc.updateSnapshot(err, spath, nowt)
			if err != nil {
				return err
			}
		}
	} else {
		return err
	}

	return nil
}

func (sc *SetMetaCommand) mergeHeader(props http.Header, headers map[string]string, isUpdate, isDelete bool) (map[string]string, bool) {
	allheaders := map[string]string{}
	for name := range props {
		if _, err := fetchHeaderOptionMap(headerOptionMap, name); err == nil || strings.HasPrefix(strings.ToLower(name), strings.ToLower(oss.HTTPHeaderOssMetaPrefix)) {
			allheaders[strings.ToLower(name)] = props.Get(name)
		}
		if strings.ToLower(name) == strings.ToLower(StatACL) {
			allheaders[strings.ToLower(oss.HTTPHeaderOssObjectACL)] = props.Get(name)
		}
	}

	if isUpdate {
		equalCount := 0
		for name, val := range headers {
			objectVal, ok := allheaders[strings.ToLower(name)]
			if ok && val == objectVal {
				equalCount += 1
			}
		}

		if equalCount == len(headers) {
			// skip update
			return allheaders, true
		}

		for name, val := range headers {
			allheaders[strings.ToLower(name)] = val
		}
	}
	if isDelete {
		for name := range headers {
			delete(allheaders, strings.ToLower(name))
		}
	}
	return allheaders, false
}

func (sc *SetMetaCommand) ossSetObjectMetaRetry(bucket *oss.Bucket, object string, options ...oss.Option) error {
	retryTimes, _ := GetInt(OptionRetryTimes, sc.command.options)
	cpOptions := append(options, oss.MetadataDirective(oss.MetaReplace))

	for i := 1; ; i++ {
		_, err := bucket.CopyObject(object, object, cpOptions...)
		if err == nil {
			return nil
		}
		if int64(i) >= retryTimes {
			return ObjectError{err, bucket.BucketName, object}
		}
	}
}

func (sc *SetMetaCommand) batchSetObjectMeta(bucket *oss.Bucket, cloudURL CloudURL, headers map[string]string, isUpdate, isDelete, force bool, routines int64) error {
	outputDir, _ := GetString(OptionOutputDir, sc.command.options)

	// init reporter
	var err error
	if sc.smOption.reporter, err = GetReporter(sc.smOption.ctnu, outputDir, commandLine); err != nil {
		return err
	}
	defer sc.smOption.reporter.Clear()

	return sc.setObjectMetas(bucket, cloudURL, headers, isUpdate, isDelete, force, routines)
}

func (sc *SetMetaCommand) setObjectMetas(bucket *oss.Bucket, cloudURL CloudURL, headers map[string]string, isUpdate, isDelete, force bool, routines int64) error {
	// producer list objects
	// consumer set meta
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go sc.command.objectStatistic(bucket, cloudURL, &sc.monitor, sc.filters)
	go sc.command.objectProducer(bucket, cloudURL, chObjects, chListError, sc.filters)

	for i := 0; int64(i) < routines; i++ {
		go sc.setObjectMetaConsumer(bucket, headers, isUpdate, isDelete, chObjects, chError)
	}

	return sc.waitRoutinueComplete(chError, chListError, routines)
}

func (sc *SetMetaCommand) setObjectMetaConsumer(bucket *oss.Bucket, headers map[string]string, isUpdate, isDelete bool, chObjects <-chan string, chError chan<- error) {
	for object := range chObjects {
		err := sc.setObjectMetaWithReport(bucket, object, headers, isUpdate, isDelete)
		if err != nil {
			chError <- err
			if !sc.smOption.ctnu {
				return
			}
			continue
		}
	}

	chError <- nil
}

func (sc *SetMetaCommand) setObjectMetaWithReport(bucket *oss.Bucket, object string, headers map[string]string, isUpdate, isDelete bool) error {
	err := sc.setObjectMeta(bucket, object, headers, isUpdate, isDelete, true, "")
	sc.command.updateMonitor(err, &sc.monitor)
	msg := fmt.Sprintf("set meta on %s", CloudURLToString(bucket.BucketName, object))
	sc.command.report(msg, err, &sc.smOption)
	return err
}

func (sc *SetMetaCommand) waitRoutinueComplete(chError, chListError <-chan error, routines int64) error {
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
				if !sc.smOption.ctnu {
					fmt.Printf(sc.monitor.progressBar(true, errExit))
					return err
				}
			}
		}
	}
	return sc.formatResultPrompt(ferr)
}

func (sc *SetMetaCommand) formatResultPrompt(err error) error {
	fmt.Printf(sc.monitor.progressBar(true, normalExit))
	if err != nil && sc.smOption.ctnu {
		return nil
	}
	return err
}

func (sc *SetMetaCommand) checkObjectFile(objFileXml string) error {
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

	sc.hasObjFile = true
	sc.objFilePath = objFileXml
	return nil
}

func (sc *SetMetaCommand) batchSetObjectsMetaFromFile(bucket *oss.Bucket, cloudURL CloudURL, headers map[string]string, isUpdate, isDelete, recursive bool, routines int64) error {
	if sc.hasObjFile || recursive {
		disableIgnoreError, _ := GetBool(OptionDisableIgnoreError, sc.command.options)
		sc.smOption.ctnu = !disableIgnoreError
	}
	outputDir, _ := GetString(OptionOutputDir, sc.command.options)

	// init reporter
	var err error
	if sc.smOption.reporter, err = GetReporter(sc.smOption.ctnu, outputDir, commandLine); err != nil {
		return err
	}
	defer sc.smOption.reporter.Clear()

	return sc.setObjectsMetaFromFile(bucket, cloudURL, sc.objFilePath, headers, isUpdate, isDelete, routines)
}

func (sc *SetMetaCommand) setObjectsMetaFromFile(bucket *oss.Bucket, cloudURL CloudURL, objectFile string, headers map[string]string, isUpdate, isDelete bool, routines int64) error {
	// producer list objects
	// consumer set meta
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)
	go sc.setObjectMetaStatistic(objectFile, &sc.monitor, sc.filters)
	go sc.setObjectMetaProducer(objectFile, chObjects, chListError, sc.filters)
	for i := 0; int64(i) < routines; i++ {
		go sc.setObjectMetaConsumer(bucket, headers, isUpdate, isDelete, chObjects, chError)
	}

	return sc.waitRoutinueComplete(chError, chListError, routines)
}

func (sc *SetMetaCommand) setObjectMetaStatistic(objectFile string, monitor Monitorer, filters []filterOptionType, options ...oss.Option) {
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

func (sc *SetMetaCommand) setObjectMetaProducer(objectFile string, chObjects chan<- string, chError chan<- error, filters []filterOptionType, options ...oss.Option) {
	defer close(chObjects)
	file, err := os.Open(objectFile)
	if err != nil {
		chError <- err
		return
	}
	defer file.Close()
	encodingType, _ := GetString(OptionEncodingType, sc.command.options)
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

func (sc *SetMetaCommand) formatSnapshotKey(bucket, object, msg string) string {
	return CloudURLToString(bucket, object) + SnapshotConnector + msg
}

func (sc *SetMetaCommand) skipSetMeta(spath string) bool {
	if sc.smOption.snapshotPath != "" {
		_, err := sc.smOption.snapshotldb.Get([]byte(spath), nil)
		if err == nil {
			return true
		}
	}
	return false
}

func (sc *SetMetaCommand) updateSnapshot(err error, spath string, srct int64) error {
	if sc.smOption.snapshotPath != "" && err == nil {
		srctstr := fmt.Sprintf("%d", srct)
		err := sc.smOption.snapshotldb.Put([]byte(spath), []byte(srctstr), nil)
		if err != nil {
			return fmt.Errorf("dump snapshot error: %s", err.Error())
		}
	}
	return nil
}

func (sc *SetMetaCommand) updateSkip(num int64) {
	atomic.AddInt64(&sc.monitor.skipNum, num)
}
