package lib

import (
	"errors"
	"fmt"
	goopt "github.com/droundy/goopt"
	"strconv"
	"strings"
)

type optionType int

// option types, only support three kinds now
const (
	OptionTypeString optionType = iota
	OptionTypeInt64
	OptionTypeFlagTrue
	OptionTypeAlternative
)

// Option describe the component of a option
type Option struct {
	name        string
	nameAlias   string
	def         string
	optionType  optionType
	minVal      string // empty means no check, for OptionTypeAlternative, minVal is the alternative values connected by '/', eg: CN/EN
	maxVal      string // empty means no check, for OptionTypeAlternative, maxVal is empty
	helpChinese string
	helpEnglish string
}

// LEnglishLanguage is the lower case of EnglishLanguage
var LEnglishLanguage = strings.ToLower(EnglishLanguage)

// OptionMap is a collection of ossutil supported options
var OptionMap = map[string]Option{
	OptionConfigFile: Option{"-c", "--config-file", "", OptionTypeString, "", "",
		"ossutil工具的配置文件路径，ossutil启动时从配置文件读取配置，在config命令中，ossutil将配置写入该文件。",
		"Path of ossutil configuration file, where to dump config in config command, or to load config in other commands that need credentials."},
	OptionEndpoint: Option{"-e", "--endpoint", "", OptionTypeString, "", "",
		fmt.Sprintf("ossutil工具的基本endpoint配置（该选项值会覆盖配置文件中的相应设置），注意其必须为一个二级域名。"),
		fmt.Sprintf("Base endpoint for oss endpoint(Notice that the value of the option will cover the value in config file). Take notice that it should be second-level domain(SLD).")},
	OptionAccessKeyID:     Option{"-i", "--access-key-id", "", OptionTypeString, "", "", "访问oss使用的AccessKeyID（该选项值会覆盖配置文件中的相应设置）。", "AccessKeyID while access oss(Notice that the value of the option will cover the value in config file)."},
	OptionAccessKeySecret: Option{"-k", "--access-key-secret", "", OptionTypeString, "", "", "访问oss使用的AccessKeySecret（该选项值会覆盖配置文件中的相应设置）。", "AccessKeySecret while access oss(Notice that the value of the option will cover the value in config file)."},
	OptionSTSToken:        Option{"-t", "--sts-token", "", OptionTypeString, "", "", "访问oss使用的STSToken（该选项值会覆盖配置文件中的相应设置），非必须设置项。", "STSToken while access oss(Notice that the value of the option will cover the value in config file), not necessary."},
	OptionLimitedNum:      Option{"", "--limited-num", strconv.Itoa(DefaultLimitedNum), OptionTypeInt64, strconv.FormatInt(MinLimitedNum, 10), "", "返回结果的最大个数。", "the limited number of return results."},
	OptionMarker:          Option{"", "--marker", "", OptionTypeString, "", "", "列举Buckets时的marker，或列举objects或Multipart Uploads时的key marker。", "the marker of bucket when list buckets, or the marker of key when list object or Multipart Uploads."},
	OptionUploadIDMarker:  Option{"", "--upload-id-marker", "", OptionTypeString, "", "", "列举Multipart Uploads时的uploadID marker。", "the marker of object when list object or Multipart Uploads."},
	OptionACL:             Option{"", "--acl", "", OptionTypeString, "", "", "acl信息的配置。", "acl information."},
	OptionShortFormat:     Option{"-s", "--short-format", "", OptionTypeFlagTrue, "", "", "显示精简格式，如果未指定该选项，默认显示长格式。", "Show by short format, if the option is not specified, show long format by default."},
	OptionDirectory:       Option{"-d", "--directory", "", OptionTypeFlagTrue, "", "", "返回当前目录下的文件和子目录，而非递归显示所有子目录下的所有object。", "Return matching subdirectory names instead of contents of the subdirectory."},
	OptionMultipart:       Option{"-m", "--multipart", "", OptionTypeFlagTrue, "", "", "指定操作的对象为bucket中未完成的Multipart事件，而非默认情况下的object。", "Indicate that the subject of the command are uncompleted Multipart Uploads, instead of objects(which is the subject in default situation."},
	OptionAllType:         Option{"-a", "--all-type", "", OptionTypeFlagTrue, "", "", "指定操作的对象为bucket中的object和未完成的Multipart事件。", "Indicate that the subject of the command contains both objects and uncompleted Multipart Uploads."},
	OptionRecursion:       Option{"-r", "--recursive", "", OptionTypeFlagTrue, "", "", "递归进行操作。对于支持该选项的命令，当指定该选项时，命令会对bucket下所有符合条件的objects进行操作，否则只对url中指定的单个object进行操作。", "operate recursively, for those commands which support the option, when use them, if the option is specified, the command will operate on all match objects under the bucket, else we will search the specified object and operate on the single object."},
	OptionBucket:          Option{"-b", "--bucket", "", OptionTypeFlagTrue, "", "", "对bucket进行操作，该选项用于确认操作作用于bucket", "the option used to make sure the operation will operate on bucket"},
	OptionStorageClass: Option{"", "--storage-class", DefaultStorageClass, OptionTypeAlternative, fmt.Sprintf("%s/%s/%s", StorageStandard, StorageIA, StorageArchive), "",
		fmt.Sprintf("设置对象的存储方式，默认值：%s，取值范围：%s/%s/%s。", DefaultStorageClass, StorageStandard, StorageIA, StorageArchive),
		fmt.Sprintf("set the storage class of bucket(default: %s), value range is: %s/%s/%s.", DefaultStorageClass, StorageStandard, StorageIA, StorageArchive)},
	OptionForce:  Option{"-f", "--force", "", OptionTypeFlagTrue, "", "", "强制操作，不进行询问提示。", "operate silently without asking user to confirm the operation."},
	OptionUpdate: Option{"-u", "--update", "", OptionTypeFlagTrue, "", "", "更新操作", "update"},
	OptionDelete: Option{"", "--delete", "", OptionTypeFlagTrue, "", "", "删除操作", "delete"},
	OptionOutputDir: Option{"", "--output-dir", DefaultOutputDir, OptionTypeString, "", "",
		fmt.Sprintf("指定输出文件所在的目录，输出文件目前包含：cp命令批量拷贝文件出错时所产生的report文件（关于report文件更多信息，请参考cp命令帮助）。默认值为：当前目录下的%s目录。", DefaultOutputDir),
		fmt.Sprintf("The option specify the directory to place output file in, output file contains: report file generated by cp command when error happens of batch copy operation(for more information about report file, see help of cp command). The default value of the option is: %s directory in current directory.", DefaultOutputDir)},
	OptionBigFileThreshold: Option{"", "--bigfile-threshold", strconv.FormatInt(DefaultBigFileThreshold, 10), OptionTypeInt64, strconv.FormatInt(MinBigFileThreshold, 10), strconv.FormatInt(MaxBigFileThreshold, 10),
		fmt.Sprintf("开启大文件断点续传的文件大小阀值，默认值:%dM，取值范围：%dB-%dB", DefaultBigFileThreshold/1048576, MinBigFileThreshold, MaxBigFileThreshold),
		fmt.Sprintf("the threshold of file size, the file size larger than the threshold will use resume upload or download(default: %d), value range is: %d-%d", DefaultBigFileThreshold, MinBigFileThreshold, MaxBigFileThreshold)},
	OptionPartSize: Option{"", "--part-size", strconv.FormatInt(DefaultPartSize, 10), OptionTypeInt64, strconv.FormatInt(MinPartSize, 10), strconv.FormatInt(MaxPartSize, 10),
		fmt.Sprintf("分片大小，单位为Byte，默认情况下ossutil根据文件大小自行计算合适的分片大小值。如果有特殊需求或者需要性能调优，可以设置该值，取值范围：%d-%d(Byte)", MinPartSize, MaxPartSize),
		fmt.Sprintf("Part size, the unit is: Byte, in default situation, ossutil will calculate the suitable part size according to file size. The option is useful when user has special needs or user need to performance tuning, the value range is: %d-%d(Byte)", MinPartSize, MaxPartSize)},
	OptionDisableCRC64: Option{"", "--disable-crc64", "", OptionTypeFlagTrue, "", "", "该选项关闭crc64，默认情况下，ossutil进行数据传输都打开crc64校验。", "Disable crc64, in default situation, ossutil open crc64 check when transmit data."},
	OptionCheckpointDir: Option{"", "--checkpoint-dir", CheckpointDir, OptionTypeString, "", "",
		fmt.Sprintf("checkpoint目录的路径(默认值为:%s)，断点续传时，操作失败ossutil会自动创建该目录，并在该目录下记录checkpoint信息，操作成功会删除该目录。如果指定了该选项，请确保所指定的目录可以被删除。", CheckpointDir),
		fmt.Sprintf("Path of checkpoint directory(default:%s), the directory is used in resume upload or download, when operate failed, ossutil will create the directory automatically, and record the checkpoint information in the directory, when the operation is succeed, the directory will be removed, so when specify the option, please make sure the directory can be removed.", CheckpointDir)},
	OptionSnapshotPath: Option{"", "--snapshot-path", "", OptionTypeString, "", "",
		"该选项用于在某些场景下加速增量上传批量文件（目前，下载和拷贝不支持该选项）。在cp上传文件时使用该选项，ossutil在指定的目录下生成文件记录文件上传的快照信息，在下一次指定该选项上传时，ossutil会读取指定目录下的快照信息进行增量上传。用户指定的snapshot目录必须为本地文件系统上的可写目录，若该目录不存在，ossutil会创建该文件用于记录快照信息，如果该目录已存在，ossutil会读取里面的快照信息，根据快照信息进行增量上传（只上传上次未成功上传的文件和本地进行过修改的文件），并更新快照信息。注意：因为该选项通过在本地记录成功上传的文件的本地lastModifiedTime，从而在下次上传时通过比较lastModifiedTime来决定是否跳过相同文件的上传，所以在使用该选项时，请确保两次上传期间没有其他用户更改了oss上的对应object。当不满足该场景时，如果想要增量上传批量文件，请使用--update选项。另外，ossutil不会主动删除snapshot-path下的快照信息，为了避免快照信息过多，当用户确定快照信息无用时，请用户自行清理snapshot-path。",
		"This option is used to accelerate the incremental upload of batch files in certain scenarios(currently, download and copy do not support this option). If you use the option when batch copy files, ossutil will generate files to record the snapshot information in the specified directory. When the next time you upload files with the option, ossutil will read the snapshot information under the specified directory for incremental upload. The snapshot-path you specified must be a local file system directory can be written in, if the directory does not exist, ossutil creates the files for recording snapshot information, else ossutil will read snapshot information from the path for incremental upload(ossutil will only upload the files which has not been successfully upload to oss and the files has been locally modified), and update the snapshot information to the directory. Note: The option record the lastModifiedTime of local files which has been successfully upload in local file system, and compare the lastModifiedTime of local files in the next cp to decided whether to skip the upload of the files, so if you use the option to achieve incremental upload, please make sure no other user modified the corresponding object in oss during the two uploads. If you can not guarantee the scenarios, please use --update option to achieve incremental upload. In addition, ossutil does not automatically delete snapshot-path snapshot information, in order to avoid too much snapshot information, when the snapshot information is useless, please clean up your own snapshot-path on your own."},
	OptionRetryTimes: Option{"", "--retry-times", strconv.Itoa(RetryTimes), OptionTypeInt64, strconv.FormatInt(MinRetryTimes, 10), strconv.FormatInt(MaxRetryTimes, 10),
		fmt.Sprintf("当错误发生时的重试次数，默认值：%d，取值范围：%d-%d", RetryTimes, MinRetryTimes, MaxRetryTimes),
		fmt.Sprintf("retry times when fail(default: %d), value range is: %d-%d", RetryTimes, MinRetryTimes, MaxRetryTimes)},
	OptionRoutines: Option{"-j", "--jobs", strconv.Itoa(Routines), OptionTypeInt64, strconv.FormatInt(MinRoutines, 10), strconv.FormatInt(MaxRoutines, 10),
		fmt.Sprintf("多文件操作时的并发任务数，默认值：%d，取值范围：%d-%d", Routines, MinRoutines, MaxRoutines),
		fmt.Sprintf("amount of concurrency tasks between multi-files(default: %d), value range is: %d-%d", Routines, MinRoutines, MaxRoutines)},
	OptionParallel: Option{"", "--parallel", "", OptionTypeInt64, strconv.FormatInt(MinParallel, 10), strconv.FormatInt(MaxParallel, 10),
		fmt.Sprintf("单文件内部操作的并发任务数，取值范围：%d-%d, 默认将由ossutil根据操作类型和文件大小自行决定。", MinRoutines, MaxRoutines),
		fmt.Sprintf("amount of concurrency tasks when work with a file, value range is: %d-%d, by default the value will be decided by ossutil intelligently.", MinRoutines, MaxRoutines)},
	OptionRange: Option{"", "--range", "", OptionTypeString, "", "", "下载文件时，指定文件下载的范围，格式为：3-9或3-或-9", "the range when download objects, the form is like: 3-9 or 3- or -9"},
	OptionEncodingType: Option{"", "--encoding-type", "", OptionTypeAlternative, URLEncodingType, "",
		fmt.Sprintf("输入的object名或文件名的编码方式，目前只支持url encode，即指定该选项时，取值范围为：%s，如果不指定该选项，则表示object名或文件名未经过编码。bucket名不支持url encode。注意，如果指定了该选项，则形如oss://bucket/object的cloud_url，输入形式为：oss://bucket/url_encode(object)，其中oss://bucket/字符串不需要编码。", URLEncodingType),
		fmt.Sprintf("the encoding type of object name or file name that user inputs, currently ossutil only supports url encode, which means the value range of the option is: %s, if you do not specify the option, it means the object name or file name that user inputed was not encoded. bucket name does not support url encode. Note, if the option is specified, the cloud_url like: oss://bucket/object should be inputted as: oss://bucket/url_encode(object), the string: oss://bucket/ should not be url encoded.", URLEncodingType)},
	OptionInclude: Option{"", "--include", DefaultNonePattern, OptionTypeString, "", "",
		fmt.Sprintf("包含对象匹配模式，如：*.jpg"),
		fmt.Sprintf("Include Pattern of key, e.g., *.jpg")},
	OptionExclude: Option{"", "--exclude", DefaultNonePattern, OptionTypeString, "", "",
		fmt.Sprintf("不包含对象匹配模式，如：*.txt"),
		fmt.Sprintf("Exclude Pattern of key, e.g., *.txt")},
	OptionMeta: Option{"", "--meta", "", OptionTypeString, "", "",
		fmt.Sprintf("设置object的meta为[header:value#header:value...]，如：Cache-Control:no-cache#Content-Encoding:gzip"),
		fmt.Sprintf("Set object meta as [header:value#header:value...], e.g., Cache-Control:no-cache#Content-Encoding:gzip")},
	OptionTimeout: Option{"", "--timeout", strconv.FormatInt(DefaultTimeout, 10), OptionTypeInt64, strconv.FormatInt(MinTimeout, 10), strconv.FormatInt(MaxTimeout, 10),
		fmt.Sprintf("签名url的超时时间，单位为秒，默认值为：%d，取值范围：%d-%d", DefaultTimeout, MinTimeout, MaxTimeout),
		fmt.Sprintf("time out of signurl, the unit is: s, default value is %d, the value range is: %d-%d", DefaultTimeout, MinTimeout, MaxTimeout)},
	OptionLanguage: Option{"-L", "--language", DefaultLanguage, OptionTypeAlternative, fmt.Sprintf("%s/%s", ChineseLanguage, EnglishLanguage), "",
		fmt.Sprintf("设置ossutil工具的语言，默认值：%s，取值范围：%s/%s，若设置成\"%s\"，请确保您的系统编码为UTF-8。", DefaultLanguage, ChineseLanguage, EnglishLanguage, ChineseLanguage),
		fmt.Sprintf("set the language of ossutil(default: %s), value range is: %s/%s, if you set it to \"%s\", please make sure your system language is UTF-8.", DefaultLanguage, ChineseLanguage, EnglishLanguage, ChineseLanguage)},
	OptionHashType: Option{"", "--type", DefaultHashType, OptionTypeAlternative, fmt.Sprintf("%s/%s", DefaultHashType, MD5HashType), "", fmt.Sprintf("计算的类型, 默认值：%s, 取值范围: %s/%s", DefaultHashType, DefaultHashType, MD5HashType),
		fmt.Sprintf("hash type, Default: %s, value range is: %s/%s", DefaultHashType, DefaultHashType, MD5HashType)},
	OptionVersion: Option{"-v", "--version", "", OptionTypeFlagTrue, "", "", fmt.Sprintf("显示ossutil的版本（%s）并退出。", Version), fmt.Sprintf("Show ossutil version (%s) and exit.", Version)},
}

func (T *Option) getHelp(language string) string {
	switch strings.ToLower(language) {
	case LEnglishLanguage:
		return T.helpEnglish
	default:
		return T.helpChinese
	}
}

// OptionMapType is the type for ossutil got options
type OptionMapType map[string]interface{}

// ParseArgOptions parse command line and returns args and options
func ParseArgOptions() ([]string, OptionMapType, error) {
	options := initOption()
	goopt.Args = make([]string, 0, 4)
	goopt.Description = func() string {
		return "Simple tool for access OSS."
	}
	goopt.Parse(nil)
	if err := checkOption(options); err != nil {
		return nil, nil, err
	}
	return goopt.Args, options, nil
}

func initOption() OptionMapType {
	m := make(OptionMapType, len(OptionMap))
	for name, option := range OptionMap {
		switch option.optionType {
		case OptionTypeInt64:
			val, _ := stringOption(option)
			m[name] = val
		case OptionTypeFlagTrue:
			val, _ := flagTrueOption(option)
			m[name] = val
		case OptionTypeAlternative:
			val, _ := stringOption(option)
			m[name] = val
		default:
			val, _ := stringOption(option)
			m[name] = val
		}
	}
	return m
}

func stringOption(option Option) (*string, error) {
	names, err := makeNames(option)
	if err == nil {
		// ignore option.def, set it to "", will assemble it after
		return goopt.String(names, "", option.getHelp(DefaultLanguage)), nil
	}
	return nil, err
}

func flagTrueOption(option Option) (*bool, error) {
	names, err := makeNames(option)
	if err == nil {
		return goopt.Flag(names, []string{}, option.getHelp(DefaultLanguage), ""), nil
	}
	return nil, err
}

func makeNames(option Option) ([]string, error) {
	if option.name == "" && option.nameAlias == "" {
		return nil, errors.New("Internal Error, invalid option whose name and nameAlias empty!")
	}

	var names []string
	if option.name == "" || option.nameAlias == "" {
		names = make([]string, 1)
		if option.name == "" {
			names[0] = option.nameAlias
		} else {
			names[0] = option.name
		}
	} else {
		names = make([]string, 2)
		names[0] = option.name
		names[1] = option.nameAlias
	}
	return names, nil
}

func checkOption(options OptionMapType) error {
	for name, optionInfo := range OptionMap {
		if option, ok := options[name]; ok {
			if optionInfo.optionType == OptionTypeInt64 {
				if val, ook := option.(*string); ook && *val != "" {
					num, err := strconv.ParseInt(*val, 10, 64)
					if err != nil {
						return fmt.Errorf("invalid option value of %s, the value: %s is not int64, please check", name, *val)
					}

					if optionInfo.minVal != "" {
						minv, _ := strconv.ParseInt(optionInfo.minVal, 10, 64)
						if num < minv {
							return fmt.Errorf("invalid option value of %s, the value: %d is smaller than the min value range: %d", name, num, minv)
						}
					}
					if optionInfo.maxVal != "" {
						maxv, _ := strconv.ParseInt(optionInfo.maxVal, 10, 64)
						if num > maxv {
							return fmt.Errorf("invalid option value of %s, the value: %d is bigger than the max value range: %d", name, num, maxv)
						}
					}
				}
			}
			if optionInfo.optionType == OptionTypeAlternative {
				if val, ook := option.(*string); ook && *val != "" {
					vals := strings.Split(optionInfo.minVal, "/")
					if FindPosCaseInsen(*val, vals) == -1 {
						return fmt.Errorf("invalid option value of %s, the value: %s is not anyone of %s", name, *val, optionInfo.minVal)
					}
				}
			}
		}
	}
	return nil
}

// GetBool is used to get bool option from option map parsed by ParseArgOptions
func GetBool(name string, options OptionMapType) (bool, error) {
	if option, ok := options[name]; ok {
		if val, ook := option.(*bool); ook {
			return *val, nil
		}
		return false, fmt.Errorf("Error: option value of %s is not bool", name)
	}
	return false, fmt.Errorf("Error: there is no option for %s", name)
}

// GetInt is used to get int option from option map parsed by ParseArgOptions
func GetInt(name string, options OptionMapType) (int64, error) {
	if option, ok := options[name]; ok {
		switch option.(type) {
		case *string:
			val, err := strconv.ParseInt(*(option.(*string)), 10, 64)
			if err == nil {
				return val, nil
			}
			if *(option.(*string)) == "" {
				return 0, fmt.Errorf("Option value of %s is empty", name)
			}
			return 0, err
		case *int64:
			return *(option.(*int64)), nil
		default:
			return 0, fmt.Errorf("Option value of %s is not int64", name)
		}
	} else {
		return 0, fmt.Errorf("There is no option for %s", name)
	}
	return 0, nil
}

// GetString is used to get string option from option map parsed by ParseArgOptions
func GetString(name string, options OptionMapType) (string, error) {
	if option, ok := options[name]; ok {
		if val, ook := option.(*string); ook {
			return *val, nil
		}
		return "", fmt.Errorf("Error: Option value of %s is not string", name)
	}
	return "", fmt.Errorf("Error: There is no option for %s", name)
}
