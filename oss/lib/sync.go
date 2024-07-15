package lib

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

/*
 * Put same type variables together to make them 64bits alignment to avoid
 * atomic.AddInt64() panic
 * Please guarantee the alignment if you add new filed
 */

var MaxSyncNumbers int = 1000000

type syncOptionType struct {
	bDelete      bool
	encodingType string
	backupDir    string
	force        bool

	enableSymlinkDir  bool
	onlyCurrentDir    bool
	disableDirObject  bool
	disableAllSymlink bool
	cpDir             string
	removeCount       int

	filters      []filterOptionType
	payerOptions []oss.Option
}

var specChineseSync = SpecText{

	synopsisText: "将本地文件目录或者oss prefix从源端同步到目的端",

	paramText: "src dest [options]",

	syntaxText: ` 
    ossutil sync local_dir cloud_url [-f] [-u] [--delete] [--backup-dir] [--enable-symlink-dir] [--disable-all-symlink] [--disable-ignore-error] [--only-current-dir] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--snapshot-path=sdir] [--payer requester]
    ossutil sync cloud_url local_dir [-f] [-u] [--delete] [--backup-dir] [--only-current-dir] [--disable-ignore-error] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--range=x-y] [--payer requester]
    ossutil sync cloud_url cloud_url [-f] [-u] [--delete] [--backup-dir] [--only-current-dir] [--disable-ignore-error] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--payer requester]
`,

	detailHelpText: ` 
    该命令和cp命令类似:支持从本地文件系统上传文件到oss，从oss下载object到本地文件系统，在oss
    上进行object拷贝; 用于源端和目的端数据同步. 

    sync命令和cp命令不同之处如下: 
    1、当输入--delete选项时,该命令会自动删除目的端存在而源端不存在的object或者移走本地文件
       如果目的端是oss, 则会删除多余的object; 如果目的端是本地文件, 则会移走本地多余的文件

       警告: 在没有完全搞清楚sync命令的行为之前, 请慎用--delete

    2、sync强制是以recursive方式遍历文件或者object, 所以不用输入-r --recursive
    3、当源端是oss://bucket/prefix, sync命令会自动在prefix后面加上字符 '/', 但是cp命令不会
       当目的端是oss://bucket/prefix, sync命令会在prefix后面加上字符 '/'; cp命令在--recursive被输入时也会在prefix后面加上字符'/'

--delete选项
    表示需要删除或者移走目的端存在而源端不存在的object或者文件

--backup-dir
    该选项表示用于备份目的端文件的目录, 不能是目的端目录的子目录,如果输入了--delete, 该选项必须输入

  
    其他选项说明、用法和cp命令相同
`,
	sampleText: ` 
    1) 从本地文件上传到oss
    前置条件如下:
    
    [root]# ls -al ./test_sync
    total 16
    drwxr-xr-x 2 root root 4096 Oct  5 13:09 .
    drwxr-xr-x 5 root root 4096 Oct  5 13:10 ..
    -rw-r--r-- 1 root root   38 Oct  5 12:48 a.txt
    -rw-r--r-- 1 root root  118 Oct  5 12:48 b.txt
    
    [root]# ./ossutil ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 13:10:23 +0800 CST          156      Standard   D282A160AB960C11ED797F009858D41D      oss://wangtw-test-sync2/a.txt
    2020-10-05 13:09:06 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/c.txt
    Object Number is: 2
    

    # run ossutil sync command
    [root]# ./ossutil64 sync ./test_sync oss://wangtw-test-sync2/ --delete -f

    total file(directory) count:2
    oss://wangtw-test-sync2,total oss object count:2
    object will be deleted count:1
    Succeed: Total num: 2, size: 156. OK num: 2(upload 2 files).
    
    average speed 6000(byte/s)
    delete object count:1
    0.111655(s) elapsed
    
    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 13:13:21 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/a.txt
    2020-10-05 13:13:21 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/b.txt
    Object Number is: 2
    

    执行sync命令之后, 本地文件a.txt, b.txt被上传到oss, oss object key c.txt 被删除

    2) 从oss上下载
    前置条件如下:
    
    [root]# ls -al ./test_sync
    total 36
    drwxr-xr-x 2 root root  4096 Oct  5 12:42 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:42 ..
    -rw-r--r-- 1 root root    38 Oct  5 11:22 a.txt
    -rw-r--r-- 1 root root 21029 Oct  5 11:22 d.txt    

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 11:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/a.txt
    2020-10-05 11:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/b.txt
    2020-10-05 11:55:47 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/c.txt
    Object Number is: 3
    

    # run ossutil sync command
    [root]# ./ossutil64 sync oss://wangtw-test-sync2 ./test_sync  --delete -f --backup-dir backup

    oss://wangtw-test-sync2,total oss object count:3
    total file(directory) count:2
    file(directory) will be removed count:1
    Succeed: Total num: 3, size: 21,185. OK num: 3(download 3 objects).
    
    average speed 243000(byte/s)
    remove file(dir) count:1
    0.172867(s) elapsed

    [root]# ls -al ./test_sync/
    total 40
    drwxr-xr-x 2 root root  4096 Oct  5 12:48 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:48 ..
    -rw-r--r-- 1 root root    38 Oct  5 12:48 a.txt
    -rw-r--r-- 1 root root   118 Oct  5 12:48 b.txt
    -rw-r--r-- 1 root root 21029 Oct  5 12:48 c.txt

    [root]# ls -al ./backup/
    total 32
    drwxr-xr-x 2 root root  4096 Oct  5 12:48 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:48 ..
    -rw-r--r-- 1 root root 21029 Oct  5 11:22 d.txt

    执行sync命令之后, oss object keys a.txt, b.txt, c.txt被下载下来
    本地文件d.txt被移动到备份目录backup

    3) 在oss之间copy
    前置条件如下:

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 12:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix1/a.txt
    2020-10-05 12:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix1/b.txt
    2020-10-05 13:00:58 +0800 CST           65      Standard   77F476710889A7FAA2603093B08EB31B      oss://wangtw-test-sync2/prefix2/a.txt
    2020-10-05 12:56:23 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/prefix2/c.txt
    Object Number is: 4
    
    0.067659(s) elapsed

    [root]# ./ossutil64 sync oss://wangtw-test-sync2/prefix1 oss://wangtw-test-sync2/prefix2 --delete -f
    oss://wangtw-test-sync2/prefix1/,total oss object count:2
    oss://wangtw-test-sync2/prefix2/,total oss object count:2
    object will be deleted count:1
    Succeed: Total num: 2, size: 156. OK num: 2(copy 2 objects).
    
    average speed 5000(byte/s)
    delete object count:1
    0.071010(s) elapsed

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 12:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix1/a.txt
    2020-10-05 12:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix1/b.txt
    2020-10-05 13:02:02 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix2/a.txt
    2020-10-05 13:02:02 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix2/b.txt
    Object Number is: 4

    执行sync命令之后, oss object key prefix1/a.txt, prefix1/b.txt 被拷贝到
    prefix2/a.txt, prefix2/b.txt 并且 prefix2/c.txt 被删除
`,
}

var specEnglishSync = SpecText{

	synopsisText: "Sync the local file directory or oss prefix from the source to the destination",

	paramText: "src dest [options]",

	syntaxText: ` 
    ossutil sync local_dir cloud_url [-f] [-u] [--delete] [--backup-dir] [--enable-symlink-dir] [--disable-all-symlink] [--disable-ignore-error] [--only-current-dir] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--snapshot-path=sdir] [--payer requester]
    ossutil sync cloud_url local_dir [-f] [-u] [--delete] [--backup-dir] [--only-current-dir] [--disable-ignore-error] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--range=x-y] [--payer requester]
    ossutil sync cloud_url cloud_url [-f] [-u] [--delete] [--backup-dir] [--only-current-dir] [--disable-ignore-error] [--output-dir=odir] [--bigfile-threshold=size] [--checkpoint-dir=cdir] [--payer requester]
`,

	detailHelpText: ` 
    This command is similar to the cp command: it supports uploading files from the local file
    system to oss, and downloading objects from oss to the local file system, Copying objects between 
    oss; It's used for data synchronization between source and destination. 

    The differences between the sync command and the cp command are as follows:
    1、When the --delete option is entered, the command will automatically delete objects or remove files 
       that exist on the destination but not exist on the source.
       If the destination is oss, the redundant objects will be deleted; 
       If the destination is local directory, the redundant files will be removed to back up directory.
       
       Warning: Before you fully understand the behavior of the sync command, please use --delete carefully
    
    2、Sync is forced to traverse files or objects recursively, so there is no need to enter -r or --recursive
    3、When the src is oss://bucket/prefix, sync command will add character '/' after the prefix, but cp command doesn't add
       When the destination is oss://bucket/prefix, sync command will add character '/' after the prefix; The cp command will also add '/' after the prefix with --recusive is entered

--delete
    Indicates that the objects or files which exist on destination
    and not exist on src needs to be deleted or removed to backup dir

--backup-dir
    This option indicates the directory used to back up the destination files needed to be removed, 
    It cannot be a subdirectory of the destination directory. 
    If you enter --delete, this option must be entered

    Other options descriptions and usage are the same as the cp command
`,

	sampleText: ` 
    1) Upload to oss
    The preconditions are as follows:
    
    [root]# ls -al ./test_sync
    total 16
    drwxr-xr-x 2 root root 4096 Oct  5 13:09 .
    drwxr-xr-x 5 root root 4096 Oct  5 13:10 ..
    -rw-r--r-- 1 root root   38 Oct  5 12:48 a.txt
    -rw-r--r-- 1 root root  118 Oct  5 12:48 b.txt

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 13:10:23 +0800 CST          156      Standard   D282A160AB960C11ED797F009858D41D      oss://wangtw-test-sync2/a.txt
    2020-10-05 13:09:06 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/c.txt
    Object Number is: 2
    

    # run ossutil sync command
    [root]# ./ossutil64 sync ./test_sync oss://wangtw-test-sync2/ --delete -f

    total file(directory) count:2
    oss://wangtw-test-sync2,total oss object count:2
    object will be deleted count:1
    Succeed: Total num: 2, size: 156. OK num: 2(upload 2 files).
    
    average speed 6000(byte/s)
    delete object count:1
    0.111655(s) elapsed

    [root]# ./ossutil64 ls oss://wangtw-test-sync2/
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 13:13:21 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/a.txt
    2020-10-05 13:13:21 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/b.txt
    Object Number is: 2

    After the sync command is executed, local files a.txt, b.txt are uploaded and oss object 
    key c.txt is deleted
    
    2) download from oss
    The preconditions are as follows:
    
    [root]# ls -al ./test_sync
    total 36
    drwxr-xr-x 2 root root  4096 Oct  5 12:42 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:42 ..
    -rw-r--r-- 1 root root    38 Oct  5 11:22 a.txt
    -rw-r--r-- 1 root root 21029 Oct  5 11:22 d.txt    

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 11:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/a.txt
    2020-10-05 11:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/b.txt
    2020-10-05 11:55:47 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/c.txt
    Object Number is: 3
    

    # run ossutil sync command
    [root]# ./ossutil64 sync oss://wangtw-test-sync2 ./test_sync  --delete -f --backup-dir backup

    oss://wangtw-test-sync2,total oss object count:3
    total file(directory) count:2
    file(directory) will be removed count:1
    Succeed: Total num: 3, size: 21,185. OK num: 3(download 3 objects).
    
    average speed 243000(byte/s)
    remove file(dir) count:1
    0.172867(s) elapsed

    [root]# ls -al ./test_sync/
    total 40
    drwxr-xr-x 2 root root  4096 Oct  5 12:48 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:48 ..
    -rw-r--r-- 1 root root    38 Oct  5 12:48 a.txt
    -rw-r--r-- 1 root root   118 Oct  5 12:48 b.txt
    -rw-r--r-- 1 root root 21029 Oct  5 12:48 c.txt

    [root]# ls -al ./backup/
    total 32
    drwxr-xr-x 2 root root  4096 Oct  5 12:48 .
    drwxr-xr-x 4 root root  4096 Oct  5 12:48 ..
    -rw-r--r-- 1 root root 21029 Oct  5 11:22 d.txt

    After the sync command is executed, oss keys a.txt, b.txt, c.txt are downloaded
    and local file d.txt is removed to backup dir

    3) copy between oss
    The preconditions are as follows:

    [root]# ./ossutil64 ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 12:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix1/a.txt
    2020-10-05 12:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix1/b.txt
    2020-10-05 13:00:58 +0800 CST           65      Standard   77F476710889A7FAA2603093B08EB31B      oss://wangtw-test-sync2/prefix2/a.txt
    2020-10-05 12:56:23 +0800 CST        21029      Standard   975F9F8EC2B34B15936FDAE1C85E71C9      oss://wangtw-test-sync2/prefix2/c.txt
    Object Number is: 4
    
    0.067659(s) elapsed

    [root]# ./ossutil64 sync oss://wangtw-test-sync2/prefix1 oss://wangtw-test-sync2/prefix2 --delete -f
    oss://wangtw-test-sync2/prefix1/,total oss object count:2
    oss://wangtw-test-sync2/prefix2/,total oss object count:2
    object will be deleted count:1
    Succeed: Total num: 2, size: 156. OK num: 2(copy 2 objects).

    average speed 5000(byte/s)
    delete object count:1
    0.071010(s) elapsed

    [root]# ./ossutil ls oss://wangtw-test-sync2
    LastModifiedTime                   Size(B)  StorageClass   ETAG                                  ObjectName
    2020-10-05 12:55:47 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix1/a.txt
    2020-10-05 12:55:47 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix1/b.txt
    2020-10-05 13:02:02 +0800 CST           38      Standard   533F2D421ED096DBB4CF458AADA64F8E      oss://wangtw-test-sync2/prefix2/a.txt
    2020-10-05 13:02:02 +0800 CST          118      Standard   C6639C466B70758C82596413C2BB050A      oss://wangtw-test-sync2/prefix2/b.txt
    Object Number is: 4

    After the sync command is executed, oss keys prefix1/a.txt, prefix1/b.txt are copied to
    prefix2/a.txt, prefix2/b.txt and prefix2/c.txt is deleted
`,
}

// SyncCommand is the command upload, download and copy objects
type SyncCommand struct {
	command    Command
	syncOption syncOptionType
}

var syncCommand = SyncCommand{
	command: Command{
		name:        "sync",
		nameAlias:   []string{"sync"},
		minArgc:     2,
		maxArgc:     2,
		specChinese: specChineseSync,
		specEnglish: specEnglishSync,
		group:       GroupTypeNormalCommand,
		validOptionNames: []string{
			// The following options are supported by sc command and cp command
			//OptionRecursion,
			OptionForce,
			OptionUpdate,
			OptionContinue,
			OptionOutputDir,
			OptionBigFileThreshold,
			OptionPartSize,
			OptionCheckpointDir,
			OptionRange,
			OptionEncodingType,
			OptionInclude,
			OptionExclude,
			OptionMeta,
			OptionACL,
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
			OptionParallel,
			OptionSnapshotPath,
			OptionDisableCRC64,
			OptionRequestPayer,
			OptionLogLevel,
			OptionMaxUpSpeed,
			//OptionPartitionDownload,
			//OptionVersionId,
			OptionLocalHost,
			OptionEnableSymlinkDir,
			OptionOnlyCurrentDir,
			OptionDisableDirObject,
			OptionDisableAllSymlink,
			OptionDisableIgnoreError,
			OptionTagging,
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
			OptionMaxDownSpeed,
			OptionUserAgent,
			OptionSignVersion,
			OptionRegion,
			OptionCloudBoxID,
			OptionForcePathStyle,

			// The following options are only supported by sc command, not supported by cp command
			OptionDelete,
			OptionBackupDir,
		},
	},
}

// function for FormatHelper interface
func (sc *SyncCommand) formatHelpForWhole() string {
	return sc.command.formatHelpForWhole()
}

func (sc *SyncCommand) formatIndependHelp() string {
	return sc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (sc *SyncCommand) Init(args []string, options OptionMapType) error {
	recursive := true
	bakupOptions := make(OptionMapType)
	for k, v := range options {
		bakupOptions[k] = v
	}

	// force to recursive and delete unsurpported options
	bakupOptions[OptionRecursion] = &recursive
	delete(bakupOptions, OptionDelete)
	delete(bakupOptions, OptionBackupDir)

	copyCommand.cpOption.bSyncCommand = true
	err := (&copyCommand).Init(args, bakupOptions)
	if err != nil {
		return err
	}
	return sc.command.Init(args, options, sc)
}

// RunCommand simulate inheritance, and polymorphism
func (sc *SyncCommand) RunCommand() error {
	sc.syncOption.bDelete, _ = GetBool(OptionDelete, sc.command.options)
	sc.syncOption.encodingType, _ = GetString(OptionEncodingType, sc.command.options)
	sc.syncOption.backupDir, _ = GetString(OptionBackupDir, sc.command.options)

	// for list file
	sc.syncOption.enableSymlinkDir, _ = GetBool(OptionEnableSymlinkDir, sc.command.options)
	sc.syncOption.onlyCurrentDir, _ = GetBool(OptionOnlyCurrentDir, sc.command.options)
	sc.syncOption.disableDirObject, _ = GetBool(OptionDisableDirObject, sc.command.options)
	sc.syncOption.disableAllSymlink, _ = GetBool(OptionDisableAllSymlink, sc.command.options)
	sc.syncOption.force, _ = GetBool(OptionForce, sc.command.options)

	// check point dir
	sc.syncOption.cpDir, _ = GetString(OptionCheckpointDir, sc.command.options)

	// payer
	payer, _ := GetString(OptionRequestPayer, sc.command.options)
	if payer != "" {
		if payer != strings.ToLower(string(oss.Requester)) {
			return fmt.Errorf("invalid request payer: %s, please check", payer)
		}
		sc.syncOption.payerOptions = append(sc.syncOption.payerOptions, oss.RequestPayer(oss.PayerType(payer)))
	}

	// filters
	var res bool
	res, sc.syncOption.filters = getFilter(os.Args)
	if !res {
		return fmt.Errorf("--include or --exclude does not support format containing dir info")
	}

	for k, v := range sc.syncOption.filters {
		LogInfo("filter %d,name:%s,pattern:%s\n", k, v.name, v.pattern)
	}

	srcURL, err := StorageURLFromString(sc.command.args[0], sc.syncOption.encodingType)
	if err != nil {
		return err
	}

	destURL, err := StorageURLFromString(sc.command.args[1], sc.syncOption.encodingType)
	if err != nil {
		return err
	}

	if srcURL.IsFileURL() && destURL.IsFileURL() {
		return fmt.Errorf("not support sync between local directory")
	}

	if srcURL.IsFileURL() {
		f, err := os.Stat(srcURL.ToString())
		if err != nil {
			return err
		}
		if !f.IsDir() {
			return fmt.Errorf("src %s is not directory", srcURL.ToString())
		}
	}

	if !sc.syncOption.bDelete {
		return copyCommand.RunCommand()
	}

	// sync command add '/' afert cloud prefix
	// cp command must have the same action when run as sync command
	srcURL = sc.adjustCloudUrl(srcURL)
	destURL = sc.adjustCloudUrl(destURL)

	// check backup dir
	if destURL.IsFileURL() {
		err = sc.CheckDestBackupDir(destURL)
		if err != nil {
			return err
		}
	}
	opType := sc.getCommandType(srcURL, destURL)

	// get file list or object key list
	srcKeys := make(map[string]string)
	destKeys := make(map[string]string)
	if srcURL.IsFileURL() {
		err = sc.GetLocalFileKeys(srcURL, srcKeys)
	} else {
		err = sc.GetOssKeys(srcURL, srcKeys)
	}

	if err != nil {
		return err
	}

	if destURL.IsFileURL() {
		err = sc.GetLocalFileKeys(destURL, destKeys)
	} else {
		err = sc.GetOssKeys(destURL, destKeys)
	}

	if err != nil {
		return err
	}

	// Get keys to be deleted
	bSame := (string(os.PathSeparator) == "/")
	for k, _ := range srcKeys {
		if bSame || opType == operationTypeCopy {
			delete(destKeys, k)
		} else if opType == operationTypePut {
			delete(destKeys, strings.Replace(k, "\\", "/", -1))
		} else {
			delete(destKeys, strings.Replace(k, "/", "\\", -1))
		}
	}

	if destURL.IsFileURL() {
		fmt.Printf("\nfile(directory) will be removed count:%d\n", len(destKeys))
	} else {
		fmt.Printf("\nobject will be deleted count:%d\n", len(destKeys))
	}

	err = copyCommand.RunCommand()
	if err != nil {
		return err
	}

	// move dest files or rm dest objects which not exist in src
	if opType == operationTypeCopy || opType == operationTypePut {
		err = sc.DeleteExtraObjects(destKeys, destURL)
	} else {
		err = sc.RemoveExtraFiles(destKeys, destURL)
	}
	return err
}

func (sc *SyncCommand) adjustCloudUrl(sUrl StorageURLer) StorageURLer {
	if sUrl.IsFileURL() {
		return sUrl
	}

	cloudUrl := sUrl.(CloudURL)
	if len(cloudUrl.object) > 0 && !strings.HasSuffix(cloudUrl.object, "/") {
		cloudUrl.object += "/"
	}
	return cloudUrl
}

func (sc *SyncCommand) DeleteExtraObjects(keys map[string]string, sUrl StorageURLer) error {
	bucketName := sUrl.(CloudURL).bucket
	bucket, err := sc.command.ossBucket(bucketName)
	if err != nil {
		return err
	}

	deleteCount := 0
	rmOptions := append(sc.syncOption.payerOptions, oss.DeleteObjectsQuiet(true))
	objects := []string{}
	for k, v := range keys {
		if len(objects) >= MaxBatchCount {
			if sc.confirm(objects) {
				err := sc.BatchRmObjects(bucket, objects, rmOptions)
				if err != nil {
					return err
				}
			}
			objects = []string{}
			deleteCount += MaxBatchCount
			fmt.Printf("\rdelete object count:%d", deleteCount)
		}
		// prefix + relativeKey
		objects = append(objects, v+k)
	}

	if len(objects) > 0 && sc.confirm(objects) {
		err := sc.BatchRmObjects(bucket, objects, rmOptions)
		if err != nil {
			return err
		}
		deleteCount += len(objects)
		fmt.Printf("\rdelete object count:%d", deleteCount)
	}
	return nil
}

func (sc *SyncCommand) RemoveExtraFiles(keys map[string]string, sUrl StorageURLer) error {
	var sortList []string
	for k, _ := range keys {
		sortList = append(sortList, k)
	}
	// remove files first,then remove dir
	sort.Sort(sort.Reverse(sort.StringSlice(sortList)))

	absDirName, err := sc.GetAbsPath(sUrl.ToString())
	if err != nil {
		return err
	}

	nowFatherDirName := ""
	for _, k := range sortList {
		if strings.HasSuffix(k, string(os.PathSeparator)) {
			// is dir
			dirName := k[0 : len(k)-1]
			readerInfos, _ := sc.readDirLimit(absDirName+dirName, 10)

			if len(readerInfos) > 0 {
				continue
			} else {
				//empty dir,need to remove or delete
				f, err := os.Stat(sc.syncOption.backupDir + dirName)
				if err != nil {
					sc.movePath(absDirName+dirName, sc.syncOption.backupDir+dirName)
				} else {
					if !f.IsDir() {
						return fmt.Errorf("backup %s is already exist,but is file", sc.syncOption.backupDir+dirName)
					} else {
						// delete the dir
						os.RemoveAll(absDirName + dirName)
					}
				}
			}
		} else {
			// is file
			fatherDir := absDirName
			index := strings.LastIndex(k, string(os.PathSeparator))
			if index >= 0 {
				fatherDir = k[:index]
			}

			if fatherDir != nowFatherDirName && fatherDir != absDirName {
				os.MkdirAll(sc.syncOption.backupDir+fatherDir, 0755)
				nowFatherDirName = fatherDir
			}

			err := sc.movePath(absDirName+k, sc.syncOption.backupDir+k)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (sc *SyncCommand) BatchRmObjects(bucket *oss.Bucket, objects []string, options []oss.Option) error {
	delRes, err := bucket.DeleteObjects(objects, options...)
	if err != nil {
		return err
	}

	if len(delRes.DeletedObjects) > 0 {
		errMsg := ""
		for _, objectKey := range delRes.DeletedObjects {
			errMsg += (" " + objectKey)
		}
		LogError("delete erro %s\n", errMsg)
		return fmt.Errorf("delete %s error", errMsg)
	}

	for _, v := range objects {
		LogInfo("delete object success %s\n", v)
	}

	return nil
}

func (sc *SyncCommand) getCommandType(srcURL StorageURLer, destURL StorageURLer) operationType {
	if srcURL.IsCloudURL() {
		if destURL.IsFileURL() {
			return operationTypeGet
		}
		return operationTypeCopy
	}
	return operationTypePut
}

func (sc *SyncCommand) GetLocalFileKeys(sUrl StorageURLer, keys map[string]string) error {
	strPath := sUrl.ToString()
	if !strings.HasSuffix(strPath, string(os.PathSeparator)) {
		// for symlink dir
		strPath += string(os.PathSeparator)
	}

	chFiles := make(chan fileInfoType, ChannelBuf)
	chFinish := make(chan error, 2)
	go sc.ReadLocalFileKeys(chFiles, chFinish, keys)
	go sc.GetFileList(strPath, chFiles, chFinish)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *SyncCommand) GetFileList(strPath string, chFiles chan<- fileInfoType, chFinish chan<- error) {
	err := getFileListCommon(strPath, chFiles, sc.syncOption.onlyCurrentDir,
		sc.syncOption.disableAllSymlink, sc.syncOption.enableSymlinkDir, sc.syncOption.filters)
	if err != nil {
		chFinish <- err
	}
}

func (sc *SyncCommand) ReadLocalFileKeys(chFiles <-chan fileInfoType, chFinish chan<- error, keys map[string]string) {
	totalCount := 0
	fmt.Printf("\n")
	for fileInfo := range chFiles {
		if copyCommand.filterFile(fileInfo, sc.syncOption.cpDir) { // exclude checkpoint files
			totalCount++
			fmt.Printf("\rtotal file(directory) count:%d", totalCount)
			keys[fileInfo.filePath] = ""
			if len(keys) > MaxSyncNumbers {
				fmt.Printf("\n")
				chFinish <- fmt.Errorf("over max sync numbers %d", MaxSyncNumbers)
				break
			}
		}
	}
	fmt.Printf("\rtotal file(directory) count:%d", totalCount)
	chFinish <- nil
}

func (sc *SyncCommand) GetAbsPath(strPath string) (string, error) {
	if filepath.IsAbs(strPath) {
		return strPath, nil
	}

	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(strPath, string(os.PathSeparator)) {
		strPath += string(os.PathSeparator)
	}

	strPath = currentDir + string(os.PathSeparator) + strPath
	absPath, err := filepath.Abs(strPath)
	if err != nil {
		return "", err
	}

	if !strings.HasSuffix(absPath, string(os.PathSeparator)) {
		absPath += string(os.PathSeparator)
	}
	return absPath, err
}

func (sc *SyncCommand) CheckDestBackupDir(sUrl StorageURLer) error {
	// create bacup dir
	createDirectory := false
	f, err := os.Stat(sUrl.ToString())
	if err != nil {
		if err := os.MkdirAll(sUrl.ToString(), 0755); err != nil {
			return err
		}
		createDirectory = true
	} else if !f.IsDir() {
		return fmt.Errorf("dest dir %s is file,is not directory", sUrl.ToString())
	}

	if createDirectory && sc.syncOption.backupDir == "" {
		return nil
	}

	if sc.syncOption.backupDir == "" {
		return fmt.Errorf("dest backup dir is empty string,please use --backup-dir")
	}

	if !strings.HasSuffix(sc.syncOption.backupDir, string(os.PathSeparator)) {
		sc.syncOption.backupDir += string(os.PathSeparator)
	}

	// check backup dir is Subdirectories or not
	absfilePath, errF := sc.GetAbsPath(sUrl.ToString())
	if errF != nil {
		return errF
	}

	absBackPath, errB := sc.GetAbsPath(sc.syncOption.backupDir)
	if errB != nil {
		return errB
	}

	if strings.Index(absBackPath, absfilePath) >= 0 {
		return fmt.Errorf("backup dir %s is subdirectory of %s", sc.syncOption.backupDir, sUrl.ToString())
	}

	// create bacup dir
	f, err = os.Stat(sc.syncOption.backupDir)
	if err != nil {
		if err := os.MkdirAll(sc.syncOption.backupDir, 0755); err != nil {
			return err
		}
	} else if !f.IsDir() {
		return fmt.Errorf("dest backup dir %s is file,is not directory", sc.syncOption.backupDir)
	}
	return nil
}

func (sc *SyncCommand) GetOssKeys(sUrl StorageURLer, keys map[string]string) error {
	bucketName := sUrl.(CloudURL).bucket
	bucket, err := sc.command.ossBucket(bucketName)
	if err != nil {
		return err
	}

	chFiles := make(chan objectInfoType, ChannelBuf)
	chFinish := make(chan error, 2)
	go sc.ReadOssKeys(keys, sUrl, chFiles, chFinish)
	go sc.GetOssKeyList(bucket, sUrl, chFiles, chFinish)
	select {
	case err := <-chFinish:
		if err != nil {
			return err
		}
	}
	return nil
}

func (sc *SyncCommand) GetOssKeyList(bucket *oss.Bucket, sURL StorageURLer, chObjects chan<- objectInfoType, chFinish chan<- error) {
	cloudURL := sURL.(CloudURL)
	err := getObjectListCommon(bucket, cloudURL, chObjects, sc.syncOption.onlyCurrentDir,
		sc.syncOption.filters, sc.syncOption.payerOptions)
	if err != nil {
		chFinish <- err
	}
}

func (sc *SyncCommand) ReadOssKeys(keys map[string]string, sURL StorageURLer, chObjects <-chan objectInfoType, chFinish chan<- error) {
	totalCount := 0
	fmt.Printf("\n")
	for objectInfo := range chObjects {
		totalCount++
		fmt.Printf("\r%s,total oss object count:%d", sURL.ToString(), totalCount)
		keys[objectInfo.relativeKey] = objectInfo.prefix
		if len(keys) > MaxSyncNumbers {
			fmt.Printf("\n")
			chFinish <- fmt.Errorf("over max sync numbers %d", MaxSyncNumbers)
			break
		}
	}
	fmt.Printf("\r%s,total oss object count:%d", sURL.ToString(), totalCount)
	chFinish <- nil
}

func (sc *SyncCommand) confirm(keys []string) bool {
	if sc.syncOption.force {
		return true
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString("\n")
	for _, v := range keys {
		logBuffer.WriteString(fmt.Sprintf("%s\n", v))
	}
	logBuffer.WriteString(fmt.Sprintf("sync:delete above objects(y or N)? "))
	fmt.Printf(logBuffer.String())

	var val string
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (sc *SyncCommand) readDirLimit(dirName string, limitCount int) ([]os.FileInfo, error) {
	f, err := os.Open(dirName)
	if err != nil {
		return nil, err
	}
	list, err := f.Readdir(limitCount)
	f.Close()
	if err != nil {
		return nil, err
	}
	return list, nil
}
func (sc *SyncCommand) movePath(srcName, destName string) error {
	err := sc.moveFileToPath(srcName, destName)
	if err != nil {
		LogError("rename %s %s error,%s\n", srcName, destName, err.Error())
	} else {
		sc.syncOption.removeCount += 1
		fmt.Printf("\rremove file(directory) count:%d", sc.syncOption.removeCount)
		LogInfo("rename success %s %s\n", srcName, destName)
	}
	return err
}
func (sc *SyncCommand) moveFileToPath(srcName, destName string) error {
	err := os.Rename(srcName, destName)
	if err == nil {
		return nil
	} else {
		inputFile, err := os.Open(srcName)
		defer inputFile.Close()
		if err != nil {
			return err
		}
		outputFile, err := os.Create(destName)
		defer outputFile.Close()
		if err != nil {
			return err
		}
		_, err = io.Copy(outputFile, inputFile)
		if err != nil {
			return err
		}
		err = os.Remove(srcName)
		if err != nil {
			return err
		}
		return nil
	}
}
