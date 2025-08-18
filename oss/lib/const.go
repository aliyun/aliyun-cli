package lib

import (
	"os"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

// all supported options of ossutil
const (
	OptionConfigFile          string = "configFile"
	OptionEndpoint                   = "endpoint"
	OptionAccessKeyID                = "accessKeyID"
	OptionAccessKeySecret            = "accessKeySecret"
	OptionSTSToken                   = "stsToken"
	OptionACL                        = "acl"
	OptionShortFormat                = "shortFormat"
	OptionLimitedNum                 = "limitedNum"
	OptionMarker                     = "marker"
	OptionUploadIDMarker             = "uploadIDMarker"
	OptionDirectory                  = "directory"
	OptionMultipart                  = "multipart"
	OptionAllType                    = "allType"
	OptionRecursion                  = "recursive"
	OptionBucket                     = "bucket"
	OptionStorageClass               = "storageClass"
	OptionForce                      = "force"
	OptionUpdate                     = "update"
	OptionDelete                     = "delete"
	OptionContinue                   = "continue"
	OptionOutputDir                  = "outputDir"
	OptionBigFileThreshold           = "bigfileThreshold"
	OptionCheckpointDir              = "checkpointDir"
	OptionSnapshotPath               = "snapshotPath"
	OptionRetryTimes                 = "retryTimes"
	OptionRoutines                   = "routines"
	OptionParallel                   = "parallel"
	OptionRange                      = "range"
	OptionEncodingType               = "encodingType"
	OptionLanguage                   = "language"
	OptionHashType                   = "hashType"
	OptionVersion                    = "version"
	OptionPartSize                   = "partSize"
	OptionDisableCRC64               = "disableCRC64"
	OptionTimeout                    = "timeout"
	OptionInclude                    = "include"
	OptionExclude                    = "exclude"
	OptionMeta                       = "meta"
	OptionRequestPayer               = "payer"
	OptionLogLevel                   = "loglevel"
	OptionMaxUpSpeed                 = "maxupspeed"
	OptionMaxDownSpeed               = "maxdownspeed"
	OptionUpload                     = "upload"
	OptionDownload                   = "download"
	OptionUrl                        = "url"
	OptionBucketName                 = "bucketname"
	OptionObject                     = "object"
	OptionAddr                       = "addr"
	OptionUpMode                     = "upmode"
	OptionDisableEmptyReferer        = "disableEmptyReferer"
	OptionMethod                     = "method"
	OptionOrigin                     = "origin"
	OptionPartitionDownload          = "partitionDownload"
	OptionSSEAlgorithm               = "SSEAlgorithm"
	OptionKMSMasterKeyID             = "KMSMasterKeyID"
	OptionKMSDataEncryption          = "KMSDataEncryption"
	OptionAcrHeaders                 = "acrHeaders"
	OptionAcrMethod                  = "acrMethod"
	OptionVersionId                  = "versionId"
	OptionAllversions                = "allVersions"
	OptionVersionIdMarker            = "versionIdMarker"
	OptionTrafficLimit               = "trafficLimit"
	OptionProxyHost                  = "proxyHost"
	OptionProxyUser                  = "proxyUser"
	OptionProxyPwd                   = "proxyPwd"
	OptionLocalHost                  = "localHost"
	OptionEnableSymlinkDir           = "enableSymlinkDir"
	OptionOnlyCurrentDir             = "onlyCurrentDir"
	OptionProbeItem                  = "probeItem"
	OptionDisableEncodeSlash         = "disableEncodeSlash"
	OptionDisableDirObject           = "disableDirObject"
	OptionRedundancyType             = "redundancyType"
	OptionDisableAllSymlink          = "disableAllSymlink"
	OptionDisableIgnoreError         = "disableIgnoreError"
	OptionTagging                    = "tagging"
	OptionStartTime                  = "startTime"
	OptionEndTime                    = "endTime"
	OptionBackupDir                  = "backupDir"
	OptionPassword                   = "password"
	OptionBlockSize                  = "blockSize"
	OptionMode                       = "mode"
	OptionECSRoleName                = "ecsRoleName"
	OptionTokenTimeout               = "tokenTimeout"
	OptionRamRoleArn                 = "ramRoleArn"
	OptionRoleSessionName            = "roleSessionName"
	OptionExternalId                 = "externalId"
	OptionReadTimeout                = "readTimeOut"
	OptionConnectTimeout             = "connectTimeOut"
	OptionSTSRegion                  = "stsRegion"
	OptionSkipVerifyCert             = "skipVerifyCert"
	OptionItem                       = "item"
	OptionUserAgent                  = "userAgent"
	OptionObjectFile                 = "objectFile"
	OptionSignVersion                = "signVersion"
	OptionRegion                     = "region"
	OptionCloudBoxID                 = "cloudBoxID"
	OptionQueryParam                 = "queryParam"
	OptionForcePathStyle             = "forcePathStyle"
	OptionRuntime                    = "runtime"
	OptionInsecure                   = "insecure"
)

// the elements show in stat object
const (
	StatName                   string = "Name"
	StatLocation                      = "Location"
	StatCreationDate                  = "CreationDate"
	StatExtranetEndpoint              = "ExtranetEndpoint"
	StatIntranetEndpoint              = "IntranetEndpoint"
	StatACL                           = "ACL"
	StatOwner                         = "Owner"
	StatLastModified                  = "Last-Modified"
	StatContentMD5                    = "Content-Md5"
	StatCRC64                         = "X-Oss-Hash-Crc64ecma"
	StatStorageClass                  = "StorageClass"
	StatSSEAlgorithm                  = "SSEAlgorithm"
	StatKMSMasterKeyID                = "KMSMasterKeyID"
	StatRedundancyType                = "RedundancyType"
	StatKMSDataEncryption             = "KMSDataEncryption"
	StatTransferAcceleration          = "TransferAcceleration"
	StatCrossRegionReplication        = "CrossRegionReplication"
	StatAccessMonitor                 = "AccessMonitor"
)

// the elements show in hash file
const (
	HashCRC64      = "CRC64-ECMA"
	HashMD5        = "MD5"
	HashContentMD5 = "Content-MD5"
)

const (
	updateEndpoint         string = "oss-cn-hangzhou.aliyuncs.com"
	updateBucket                  = "ossutil-version-update"
	updateVersionObject           = "ossutilversion"
	updateBinaryLinux32           = "ossutil32"
	updateBinaryLinux64           = "ossutil64"
	updateBinaryLinuxArm32        = "ossutilarm32"
	updateBinaryLinuxArm64        = "ossutilarm64"
	updateBinaryWindow32          = "ossutil32.exe"
	updateBinaryWindow64          = "ossutil64.exe"
	updateBinaryMac32             = "ossutilmac32"
	updateBinaryMac64             = "ossutilmac64"
	updateBinaryMacArm64          = "ossutilmacarm64"
	updateTmpVersionFile          = ".ossutil_tmp_vsersion"
)

// global public variable
const (
	Package                 string = "ossutil"
	ChannelBuf              int    = 1000
	Version                 string = "v1.7.19"
	DefaultEndpoint         string = "oss.aliyuncs.com"
	ChineseLanguage                = "CH"
	EnglishLanguage                = "EN"
	Scheme                  string = "oss"
	DefaultConfigFile              = "~" + string(os.PathSeparator) + ".ossutilconfig"
	MaxUint                 uint   = ^uint(0)
	MaxInt                  int    = int(MaxUint >> 1)
	MaxUint64               uint64 = ^uint64(0)
	MaxInt64                int64  = int64(MaxUint64 >> 1)
	ReportPrefix                   = "ossutil_report_"
	ReportSuffix                   = ".report"
	DefaultOutputDir               = "ossutil_output"
	CheckpointDir                  = ".ossutil_checkpoint"
	CheckpointSep                  = "---"
	SnapshotConnector              = "==>"
	SnapshotSep                    = "#"
	MaxPartNum                     = 10000
	MaxIdealPartNum                = MaxPartNum / 10
	MinIdealPartNum                = MaxPartNum / 500
	MaxIdealPartSize               = 524288000
	MinIdealPartSize               = 1048576
	DefaultBigFileThreshold int64  = 104857600
	MaxBigFileThreshold     int64  = MaxInt64
	MinBigFileThreshold     int64  = 0
	DefaultPartSize         int64  = -1
	MaxPartSize             int64  = MaxInt64
	MinPartSize             int64  = 1
	DefaultLimitedNum              = -1
	MinLimitedNum                  = 0
	RetryTimes              int    = 10
	MaxRetryTimes           int64  = 500
	MinRetryTimes           int64  = 1
	Routines                int    = 3
	MaxRoutines             int64  = 10000
	MinRoutines             int64  = 1
	MaxParallel             int64  = 10000
	MinParallel             int64  = 1
	DefaultHashType         string = "crc64"
	MD5HashType             string = "md5"
	LogFilePrefix                  = "ossutil_log_"
	URLEncodingType                = "url"
	StorageStandard                = string(oss.StorageStandard)
	StorageIA                      = string(oss.StorageIA)
	StorageArchive                 = string(oss.StorageArchive)
	StorageColdArchive             = string(oss.StorageColdArchive)
	DefaultStorageClass            = StorageStandard
	DefaultMethod                  = string(oss.HTTPGet)
	DefaultTimeout                 = 60
	MinTimeout                     = 0
	MaxTimeout                     = MaxInt64
	DefaultNonePattern             = ""
	IncludePrompt                  = "--include"
	ExcludePrompt                  = "--exclude"
	MaxAppendObjectSize     int64  = 5368709120
	MaxBatchCount           int    = 100
)

const (
	objectType    = 0x00000001
	multipartType = 0x00000010
	allType       = objectType | multipartType // marker for objects
	bucketType    = 0x10000000
)

var DefaultLanguage = getOsLang()
