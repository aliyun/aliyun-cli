package lib

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	goopt "github.com/droundy/goopt"
)

type optionType int

// option types, only support three kinds now
const (
	OptionTypeString optionType = iota
	OptionTypeInt64
	OptionTypeFlagTrue
	OptionTypeAlternative
	OptionTypeStrings
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
	OptionMarker:          Option{"", "--marker", "", OptionTypeString, "", "", "列举Buckets时的marker，或列举objects或Multipart Uploads时的key marker, 或者其他有需要marker的地方。", "the marker of bucket when list buckets, or the marker of key when list object or Multipart Uploads, Or other places where a marker is needed"},
	OptionUploadIDMarker:  Option{"", "--upload-id-marker", "", OptionTypeString, "", "", "列举Multipart Uploads时的uploadID marker。", "the marker of object when list object or Multipart Uploads."},
	OptionACL:             Option{"", "--acl", "", OptionTypeString, "", "", "acl信息的配置。", "acl information."},
	OptionShortFormat:     Option{"-s", "--short-format", "", OptionTypeFlagTrue, "", "", "显示精简格式，如果未指定该选项，默认显示长格式。", "Show by short format, if the option is not specified, show long format by default."},
	OptionDirectory:       Option{"-d", "--directory", "", OptionTypeFlagTrue, "", "", "返回当前目录下的文件和子目录，而非递归显示所有子目录下的所有object。", "Return matching subdirectory names instead of contents of the subdirectory."},
	OptionMultipart:       Option{"-m", "--multipart", "", OptionTypeFlagTrue, "", "", "指定操作的对象为bucket中未完成的Multipart事件，而非默认情况下的object。", "Indicate that the subject of the command are uncompleted Multipart Uploads, instead of objects(which is the subject in default situation."},
	OptionAllType:         Option{"-a", "--all-type", "", OptionTypeFlagTrue, "", "", "指定操作的对象为bucket中的object和未完成的Multipart事件。", "Indicate that the subject of the command contains both objects and uncompleted Multipart Uploads."},
	OptionRecursion:       Option{"-r", "--recursive", "", OptionTypeFlagTrue, "", "", "递归进行操作。对于支持该选项的命令，当指定该选项时，命令会对bucket下所有符合条件的objects进行操作，否则只对url中指定的单个object进行操作。", "operate recursively, for those commands which support the option, when use them, if the option is specified, the command will operate on all match objects under the bucket, else we will search the specified object and operate on the single object."},
	OptionBucket:          Option{"-b", "--bucket", "", OptionTypeFlagTrue, "", "", "对bucket进行操作，该选项用于确认操作作用于bucket", "the option used to make sure the operation will operate on bucket"},
	OptionStorageClass: Option{"", "--storage-class", DefaultStorageClass, OptionTypeAlternative, fmt.Sprintf("%s/%s/%s/%s", StorageStandard, StorageIA, StorageArchive, StorageColdArchive), "",
		fmt.Sprintf("设置对象的存储方式，默认值：%s，取值范围：%s/%s/%s/%s。", DefaultStorageClass, StorageStandard, StorageIA, StorageArchive, StorageColdArchive),
		fmt.Sprintf("set the storage class of bucket(default: %s), value range is: %s/%s/%s/%s.", DefaultStorageClass, StorageStandard, StorageIA, StorageArchive, StorageColdArchive)},
	OptionForce:  Option{"-f", "--force", "", OptionTypeFlagTrue, "", "", "强制操作，不进行询问提示。", "operate silently without asking user to confirm the operation."},
	OptionUpdate: Option{"-u", "--update", "", OptionTypeFlagTrue, "", "", "更新操作", "update"},
	OptionDelete: Option{"", "--delete", "", OptionTypeFlagTrue, "", "", "删除操作", "delete"},
	OptionOutputDir: Option{"", "--output-dir", DefaultOutputDir, OptionTypeString, "", "",
		fmt.Sprintf("指定输出文件所在的目录，输出文件目前包含：cp命令批量拷贝文件出错时所产生的report文件（关于report文件更多信息，请参考cp命令帮助）。默认值为：当前目录下的%s目录。", DefaultOutputDir),
		fmt.Sprintf("The option specify the directory to place output file in, output file contains: report file generated by cp command when error happens of batch copy operation(for more information about report file, see help of cp command). The default value of the option is: %s directory in current directory.", DefaultOutputDir)},
	OptionBigFileThreshold: Option{"", "--bigfile-threshold", strconv.FormatInt(DefaultBigFileThreshold, 10), OptionTypeInt64, strconv.FormatInt(MinBigFileThreshold, 10), strconv.FormatInt(MaxBigFileThreshold, 10),
		fmt.Sprintf("开启大文件断点续传的文件大小阈值，默认值:%dM，取值范围：%dB-%dB", DefaultBigFileThreshold/1048576, MinBigFileThreshold, MaxBigFileThreshold),
		fmt.Sprintf("the threshold of file size, the file size larger than the threshold will use resume upload or download(default: %d), value range is: %d-%d", DefaultBigFileThreshold, MinBigFileThreshold, MaxBigFileThreshold)},
	OptionPartSize: Option{"", "--part-size", strconv.FormatInt(DefaultPartSize, 10), OptionTypeInt64, strconv.FormatInt(MinPartSize, 10), strconv.FormatInt(MaxPartSize, 10),
		fmt.Sprintf("分片大小，单位为Byte，默认情况下ossutil根据文件大小自行计算合适的分片大小值。如果有特殊需求或者需要性能调优，可以设置该值，取值范围：%d-%d(Byte)", MinPartSize, MaxPartSize),
		fmt.Sprintf("Part size, the unit is: Byte, in default situation, ossutil will calculate the suitable part size according to file size. The option is useful when user has special needs or user need to performance tuning, the value range is: %d-%d(Byte)", MinPartSize, MaxPartSize)},
	OptionDisableCRC64: Option{"", "--disable-crc64", "", OptionTypeFlagTrue, "", "", "该选项关闭crc64，默认情况下，ossutil进行数据传输都打开crc64校验。", "Disable crc64, in default situation, ossutil open crc64 check when transmit data."},
	OptionCheckpointDir: Option{"", "--checkpoint-dir", CheckpointDir, OptionTypeString, "", "",
		fmt.Sprintf("checkpoint目录的路径(默认值为:%s)，断点续传时，操作失败ossutil会自动创建该目录，并在该目录下记录checkpoint信息，操作成功会删除该目录。如果指定了该选项，请确保所指定的目录可以被删除。", CheckpointDir),
		fmt.Sprintf("Path of checkpoint directory(default:%s), the directory is used in resume upload or download, when operate failed, ossutil will create the directory automatically, and record the checkpoint information in the directory, when the operation is succeed, the directory will be removed, so when specify the option, please make sure the directory can be removed.", CheckpointDir)},
	OptionSnapshotPath: Option{"", "--snapshot-path", "", OptionTypeString, "", "",
		"该选项用于在某些场景下加速增量上传批量文件或者增量下载批量object。在cp上传文件或者下载object时使用该选项，ossutil在指定的目录下生成快照文件，记录文件上传或者object下载的快照信息，在下一次指定该选项上传或下载时，ossutil会读取指定目录下的快照信息进行增量上传或者下载。用户指定的snapshot目录必须为本地文件系统上的可写目录，若该目录不存在，ossutil会创建该文件用于记录快照信息，如果该目录已存在，ossutil会读取里面的快照信息，根据快照信息进行增量上传（只上传上次未成功上传的文件和本地进行过修改的文件）或者增量下载（只下载上次未成功下载的object和修改过的object），并更新快照信息。注意：该选项在本地记录了成功上传的文件的本地lastModifiedTime或者记录了下载object的lastModifiedTime，从而在下次上传或者下载时通过比较lastModifiedTime来决定是否跳过相同文件的上传或者跳过相同的object下载。当使用该选项上传时，请确保两次上传期间没有其他用户更改了oss上的对应object。当不满足该场景时，如果想要增量上传批量文件，请使用--update选项。ossutil不会主动删除snapshot-path下的快照信息，当用户确定快照信息无用时，请用户及时自行删除snapshot-path。",
		"This option is used to accelerate the incremental upload of batch files or download objects in certain scenarios. If you use the option when upload files or download objects, ossutil will generate files to record the snapshot information in the specified directory. When the next time you upload files or download objects with the option, ossutil will read the snapshot information under the specified directory for incremental upload or incremental download. The snapshot-path you specified must be a local file system directory can be written in, if the directory does not exist, ossutil creates the files for recording snapshot information, else ossutil will read snapshot information from the path for incremental upload(ossutil will only upload the files which haven't not been successfully uploaded to oss or been locally modified) or incremental download(ossutil will only download the objects which have not been successfully downloaded or have been modified), and update the snapshot information to the directory. Note: The option record the lastModifiedTime of local files which have been successfully uploaded in local file system or lastModifiedTime of objects which have been successfully downloaded, and compare the lastModifiedTime of local files or objects in the next cp to decided whether to skip the file or object. If you use the option to achieve incremental upload, please make sure no other user modified the corresponding object in oss during the two uploads. If you can not guarantee the scenarios, please use --update option to achieve incremental upload. In addition, ossutil does not automatically delete snapshot-path snapshot information, in order to avoid too much snapshot information, when the snapshot information is useless, please clean up your own snapshot-path on your own immediately."},
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
		fmt.Sprintf("输入或者输出的object名或文件名的编码方式，目前只支持url encode，即指定该选项时，取值范围为：%s，如果不指定该选项，则表示object名或文件名未经过编码。bucket名不支持url encode。注意，如果指定了该选项，则形如oss://bucket/object的cloud_url，输入形式为：oss://bucket/url_encode(object)，其中oss://bucket/字符串不需要编码。", URLEncodingType),
		fmt.Sprintf("the encoding type of object name or file name that user inputs or outputs, currently ossutil only supports url encode, which means the value range of the option is: %s, if you do not specify the option, it means the object name or file name that user inputed or outputed was not encoded. bucket name does not support url encode. Note, if the option is specified, the cloud_url like: oss://bucket/object should be inputted as: oss://bucket/url_encode(object), the string: oss://bucket/ should not be url encoded.", URLEncodingType)},
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
	OptionVersion:      Option{"-v", "--version", "", OptionTypeFlagTrue, "", "", fmt.Sprintf("显示ossutil的版本（%s）并退出。", Version), fmt.Sprintf("Show ossutil version (%s) and exit.", Version)},
	OptionRequestPayer: Option{"", "--payer", "", OptionTypeString, "", "", "请求的支付方式，如果为请求者付费模式，可以将该值设置成\"requester\"", "The payer of the request. You can set this value to \"requester\" if you want pay for requester"},
	OptionLogLevel: Option{"", "--loglevel", "", OptionTypeString, "", "",
		"日志级别，默认为空,表示不输出日志文件,可选值为:info|debug,info输出提示信息日志,debug输出详细信息日志(包括http请求和响应信息)",
		"log level,default is empty(no log file output),optional value is:info|debug,info will output information logs,debug will output detail logs(including http request and response logs)"},
	OptionMaxUpSpeed: Option{"", "--maxupspeed", "", OptionTypeInt64, "", "",
		"最大上传速度,单位:KB/s,缺省值为0(不受限制)",
		"max upload speed,the unit is:KB/s,default value is 0(unlimited)"},
	OptionMaxDownSpeed: Option{"", "--maxdownspeed", "", OptionTypeInt64, "", "",
		"最大下载速度,单位:KB/s,缺省值为0(不受限制)",
		"max download speed,the unit is:KB/s,default value is 0(unlimited)"},
	OptionUpload: Option{"", "--upload", "", OptionTypeFlagTrue, "", "",
		"表示上传到oss,主要在命令probe中使用",
		"specifies upload action to oss,primarily used in probe command"},
	OptionDownload: Option{"", "--download", "", OptionTypeFlagTrue, "", "",
		"表示从oss下载,主要在命令probe中使用",
		"specifies download action from oss,primarily used in probe command"},
	OptionUrl: Option{"", "--url", "", OptionTypeString, "", "",
		"表示一个url地址,主要在命令probe中使用",
		"specifies a url address,primarily used in probe command"},
	OptionBucketName: Option{"", "--bucketname", "", OptionTypeString, "", "",
		"表示bucket的名称,主要在命令probe中使用",
		"specifies a name of bucket,primarily used in probe command"},
	OptionObject: Option{"", "--object", "", OptionTypeString, "", "",
		"表示oss中对象的名称,主要在命令probe中使用",
		"specifies a name of object,primarily used in probe command"},
	OptionAddr: Option{"", "--addr", "", OptionTypeString, "", "",
		"表示一个网络地址,通常为域名,主要在命令probe中使用",
		"specifies a network address,usually a domain,primarily used in probe command"},
	OptionUpMode: Option{"", "--upmode", "", OptionTypeString, "", "",
		"表示上传模式,缺省值为normal,取值为:normal|append|multipart,分别表示正常上传、追加上传、分块上传,主要在命令probe中使用",
		"specifies the upload mode,default value is normal,optional value is:normal|append|multipart, which means normal upload、append upload and multipart upload,it is primarily used in probe command."},
	OptionDisableEmptyReferer: Option{"", "--disable-empty-referer", "", OptionTypeFlagTrue, "", "",
		"表示不允许referer字段为空,主要在referer命令中使用",
		"specifies that the referer field is not allowed to be empty,primarily used in referer command"},
	OptionMethod: Option{"", "--method", "", OptionTypeString, "", "",
		"表示命令的操作类型,取值为PUT、GET、DELETE、LIST等",
		"specifies the command's operation type. the values ​​are PUT, GET, DELETE, LIST, etc"},
	OptionOrigin: Option{"", "--origin", "", OptionTypeString, "", "",
		"表示http请求头origin字段的值",
		"specifies the value of origin field in http header"},
	OptionPartitionDownload: Option{"", "--partition-download", "", OptionTypeString, "", "",
		"分区下载使用,一个ossutil命令下载一个分区,其值格式为\"分区编号:总分区数\",比如1:5,表示当前ossutil下载分区1,总共有5个分区;分区号从1开始编号,objects的分区规则由工具内部算法决定;利用该选项,待下载的objects分成多个区,可以由多个ossutil命令一起下载完成,每个ossutil命令下载各自的分区,多个ossutil命令可以并行在不同机器上执行",
		"the option is used in partition download mode, one command to download one partition,the value format is \"partition number:total count of partitions\",such as 1:5, indicating that the command downloads partition 1,total partition count is 5; the partition number is numbered from 1, and the partitioning rules for objects are determined by ossutil; with this option, the objects to be downloaded are divided into multiple partitions, which can be downloaded by multiple ossutil commands,each ossutil command can download its own partition,multiple ossutil commands can be executed on different machines in parallel."},
	OptionSSEAlgorithm: Option{"", "--sse-algorithm", "", OptionTypeString, "", "",
		"表示服务端加密算法，取值为KMS或者AES256",
		"specifies the server side encryption algorithm,value is KMS or AES256."},
	OptionKMSMasterKeyID: Option{"", "--kms-masterkey-id", "", OptionTypeString, "", "",
		"表示kms秘钥托管服务中的主秘钥id",
		"specifies the primary key id in the kms(key management service)"},
	OptionKMSDataEncryption: Option{"", "--kms-data-encryption", "", OptionTypeString, "", "",
		"表示kms秘钥托管服务使用的加密算法,目前取值仅支持为SM4, 或者为空",
		"specifies the kms data service encryption algorithm,Currently only supports the value SM4 or emtpy"},
	OptionAcrHeaders: Option{"", "--acr-headers", "", OptionTypeString, "", "",
		"表示http header Access-Control-Request-Headers的值,主要用于cors-options命令",
		"specifies the value of the http header Access-Control-Request-Headers, primarily used in cors-options command."},
	OptionAcrMethod: Option{"", "--acr-method", "", OptionTypeString, "", "",
		"表示http header Access-Control-Request-Method的值,主要用于cors-options命令",
		"specifies the value of the http header Access-Control-Request-Method,primarily used in cors-options command."},
	OptionVersionId: Option{"", "--version-id", "", OptionTypeString, "", "",
		"表示object的版本id",
		"specifies the object's version id"},
	OptionAllversions: Option{"", "--all-versions", "", OptionTypeFlagTrue, "", "",
		"表示object所有版本",
		"specifies the object's all versions"},
	OptionVersionIdMarker: Option{"", "--version-id-marker", "", OptionTypeString, "", "",
		"表示列举objects所有版本的version id marker",
		"specifies the marker of object version id when list objects's all versions"},
	OptionTrafficLimit: Option{"", "--trafic-limit", "", OptionTypeInt64, "", "",
		"http请求限速,单位:bit/s,缺省值为0(不受限制),用于sign命令",
		"http request speed limit,the unit is:bit/s,default value is 0(unlimited),primarily used in sign command"},
	OptionProxyHost: Option{"", "--proxy-host", "", OptionTypeString, "", "",
		"网络代理服务器的url地址,支持http/https/socks5,比如 https://120.79.128.211:3128, socks5://120.79.128.211:1080",
		"url of network proxy server, which supports http/https/socks5, such as https://120.79.128.211:3128, socks5://120.79.128.211:1080"},
	OptionProxyUser: Option{"", "--proxy-user", "", OptionTypeString, "", "",
		"网络代理服务器的用户名,默认为空",
		"username of network proxy, default is empty"},
	OptionProxyPwd: Option{"", "--proxy-pwd", "", OptionTypeString, "", "",
		"网络代理服务器的密码,默认为空",
		"password of network proxy, default is empty"},
	OptionLocalHost: Option{"", "--local-host", "", OptionTypeString, "", "",
		"工具的ip地址,比如 127.0.0.1",
		"ossutil's ip ,such as 127.0.0.1"},
	OptionEnableSymlinkDir: Option{"", "--enable-symlink-dir", "", OptionTypeFlagTrue, "", "",
		"表示允许上传链接子目录下的文件,默认不上传; probe命令可以探测是否存在死循环链接文件或者目录",
		"specifies uploading link subdirectories,default are not uploaded; The probe command can detect whether there is a dead cycle symlink file or directory."},
	OptionOnlyCurrentDir: Option{"", "--only-current-dir", "", OptionTypeFlagTrue, "", "",
		"表示仅操作当前目录下的文件或者object, 忽略子目录",
		"specifies that only files or objects in the current directory are manipulated, and subdirectories are ignored."},
	OptionProbeItem: Option{"", "--probe-item", "", OptionTypeString, "", "",
		"表示probe命令的探测项目, 取值可为upload-speed, download-speed, cycle-symlink",
		"specifies probe command's probe item, the value can be upload-speed, download-speed, cycle-symlink"},
	OptionDisableEncodeSlash: Option{"", "--disable-encode-slash", "", OptionTypeFlagTrue, "", "",
		"表示不对url path中的'/'进行编码, 主要用于sign命令",
		"specifies no encoding of '/' in url path section, primarily used in sign command"},
	OptionDisableDirObject: Option{"", "--disable-dir-object", "", OptionTypeFlagTrue, "", "",
		"表示上传文件时不为目录生成oss对象,主要用于cp命令",
		"specifies that oss object is not generated for directory itself when uploading, primarily used in cp command"},
	OptionRedundancyType: Option{"", "--redundancy-type", "", OptionTypeString, "", "",
		"表示bucket的数据容灾类型, 取值可为LRS, ZRS. LRS为默认值,表示本地容灾, ZRS表示更高可用的同城多可用区容灾(3AZ)",
		"specifies bucket data redundancy type, the value can be LRS, ZRS. LRS is default value, specifies locally redundant storage; ZRS specifies higher availability of redundant storage"},
	OptionDisableAllSymlink: Option{"", "--disable-all-symlink", "", OptionTypeFlagTrue, "", "",
		"表示不允许上传目录下的链接文件以及链接目录, 缺省值为false",
		"specifies that uploading of symlink files and symlink directories under the directory is not allowed, the default value is false."},
	OptionDisableIgnoreError: Option{"", "--disable-ignore-error", "", OptionTypeFlagTrue, "", "",
		"批量操作时候不忽略错误, 缺省值为false",
		"specifies that do not ignore errors during batch cp, default value is false"},
	OptionTagging: Option{"", "--tagging", "", OptionTypeString, "", "",
		"设置object的tagging,取值格式如[\"TagA=A&TagB=B...\"]",
		"Set object tagging, value format is [\"TagA=A&TagB=B...]\""},
	OptionStartTime: Option{"", "--start-time", "", OptionTypeInt64, "", "",
		"起始时间,为linux/Unix系统里面的时间戳,既从1970年1月1日(UTC/GMT的午夜)开始所经过的秒数",
		"The start time is the timestamp in the Linux/Unix system, that is, the number of seconds that have passed since January 1, 1970 (midnight UTC/GMT)"},
	OptionEndTime: Option{"", "--end-time", "", OptionTypeInt64, "", "",
		"结束时间,为linux/Unix系统里面的时间戳,既从1970年1月1日(UTC/GMT的午夜)开始所经过的秒数",
		"The end time is the timestamp in the Linux/Unix system, that is, the number of seconds that have passed since January 1, 1970 (midnight UTC/GMT)"},
	OptionBackupDir: Option{"", "--backup-dir", "", OptionTypeString, "", "",
		"sync命令使用的备份文件的目录",
		"The directory of the backup file used by the sync command"},
	OptionPassword: Option{"", "--password", "", OptionTypeFlagTrue, "", "",
		"表示从键盘输入accessKeySecret参数",
		"specifies that the accessKeySecret is inputted from the keyboard"},
	OptionBlockSize: Option{"-B", "--block-size", "", OptionTypeString, "", "",
		"表示du命令字节显示的单位,取值可以为KB, MB, GB, TB",
		"specifies the unit of byte display for du command, the value can be KB, MB, GB, TB"},
	OptionMode: Option{"", "--mode", "", OptionTypeString, "", "",
		"表示鉴权模式，取值可以为AK，StsToken，RamRoleArn，EcsRamRole，缺省值为空",
		"specifies the authentication mode, the value can be AK，StsToken，RamRoleArn，EcsRamRole, default value is empty."},
	OptionECSRoleName: Option{"", "--ecs-role-name", "", OptionTypeString, "", "",
		"表示角色名，主要用于EcsRamRole模式",
		"specifies the authentication mode, primarily used in EcsRamRole mode."},
	OptionTokenTimeout: Option{"", "--token-timeout", "", OptionTypeInt64, "", "",
		"表示token的有效时间，单位为秒, 缺省值为3600，主要用于RamRoleArn模式下的AssumeRole参数",
		"specifies the valid time of a token, the unit is: s, default value is 3600, primarily used for AssumeRole parameters in RamRoleArn mode"},
	OptionRamRoleArn: Option{"", "--ram-role-arn", "", OptionTypeString, "", "",
		"表示RAM角色的ARN，主要用于RamRoleArn模式",
		"specifies the ARN of ram role, primarily used in RamRoleArn mode."},
	OptionRoleSessionName: Option{"", "--role-session-name", "", OptionTypeString, "", "",
		"表示会话名字，主要用于RamRoleArn模式",
		"specifies the session name, primarily used in RamRoleArn mode."},
	OptionExternalId: Option{"", "--external-id", "", OptionTypeString, "", "",
		"表示外部ID，主要用于RamRoleArn模式",
		"specifies the external ID, primarily used in RamRoleArn mode."},
	OptionReadTimeout: Option{"", "--read-timeout", "", OptionTypeInt64, "", "",
		"表示客户端读超时的时间，单位为秒, 缺省值为1200",
		"specifies the time that the client read timed out, the unit is: s, default value is 1200."},
	OptionConnectTimeout: Option{"", "--connect-timeout", "", OptionTypeInt64, "", "",
		"表示客户端连接超时的时间，单位为秒, 缺省值为120",
		"specifies the time that the client connection timed out, the unit is: s, default value is 120."},
	OptionSTSRegion: Option{"", "--sts-region", "", OptionTypeString, "", "",
		"指定sts endpoint的地区，比如cn-shenzhen，其中，cn指代的是国家，shenzhen指代的是地区，用于构造sts endpoint，该选项缺省时，sts endpoint为sts.aliyuncs.com，主要用于RamRoleArn模式",
		"specifies the region of sts endpoint, such as cn-shenzhen, in this case, cn refers to the country and shenzhen refers to the region, to construct sts endpoint, when this option defaults, the sts endpoint is sts.aliyuncs.com, primarily used in RamRoleArn mode."},
	OptionSkipVerifyCert: Option{"", "--skip-verify-cert", "", OptionTypeFlagTrue, "", "",
		"表示不校验服务端的数字证书",
		"specifies that the oss server's digital certificate file will not be verified"},
	OptionItem: Option{"", "--item", "", OptionTypeString, "", "",
		"表示命令的功能类型，取值为LOCATION、PROGRESS等",
		"specifies the command's function type. the values are LOCATION, PROGRESS, etc"},
	OptionUserAgent: Option{"", "--ua", "", OptionTypeString, "", "",
		"指定http请求中的user agent, 会在缺省值后面加上指定值",
		"Specify the user agent in the http request, and the specified value will be added after the default value"},
	OptionObjectFile: Option{"", "--object-file", "", OptionTypeString, "", "",
		"表示所有待处理的objects，取值为一个存在的文件路径",
		"Specify all the objects that need to be operated, and the specified value should be a exists file path"},
	OptionSignVersion: Option{"", "--sign-version", "", OptionTypeString, "", "",
		"http请求使用的签名算法版本, 缺省为空, 表示v1版本",
		"The version of the signature algorithm used in the HTTP request. It is empty by default, indicating the V1 version"},
	OptionRegion: Option{"", "--region", "", OptionTypeString, "", "",
		"bucket所在的地区, 比如cn-hangzhou, 缺省值为空, 如果使用v4签名则必须传入",
		"The region where the bucket is located, such as cn-hangzhou. The default value is empty. If V4 signature is used, it must be inputted"},
	OptionCloudBoxID: Option{"", "--cloudbox-id", "", OptionTypeString, "", "",
		"云盒的id，缺省值为空，适用于云盒场景",
		"The ID of the cloud box. The default value is empty. It is applicable to cloud box scenarios"},
	OptionQueryParam: Option{"", "--query-param", "", OptionTypeStrings, "", "",
		"设置请求的query参数",
		"Set the query parameters for the request"},
	OptionForcePathStyle: Option{"", "--force-path-style", "", OptionTypeFlagTrue, "", "",
		"使用 path style 访问方式",
		"Use path-style access "},
	OptionRuntime: Option{"", "--runtime", "", OptionTypeInt64, "", "",
		"设置命令的持续的运行时间",
		"specifies the max running time of the command."},
	OptionInsecure: Option{"", "--insecure", "", OptionTypeFlagTrue, "", "",
		"表示不校验服务端的数字证书",
		"specifies that the oss server's digital certificate file will not be verified."},
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
		case OptionTypeStrings:
			val, _ := stringsOption(option)
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

func stringsOption(option Option) (*[]string, error) {
	names, err := makeNames(option)
	if err == nil {
		return goopt.Strings(names, "", option.getHelp(DefaultLanguage)), nil
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

// GetStrings is used to get slice option from option map parsed by ParseArgOptions
func GetStrings(name string, options OptionMapType) ([]string, error) {
	if option, ok := options[name]; ok {
		if val, ook := option.(*[]string); ook {
			return *val, nil
		}
		return nil, fmt.Errorf("Error: Option value of %s is not []string", name)
	}
	return nil, fmt.Errorf("Error: There is no option for %s", name)
}
