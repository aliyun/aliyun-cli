package lib

import (
	"bytes"
	"container/list"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

const (
	objectPrefex string = "oss-test-probe-"
)

const (
	normalMode    string = "normal"
	appendMode           = "append"
	multipartMode        = "multipart"
)

var specChineseProbe = SpecText{
	synopsisText: "探测命令,支持多种功能探测",

	paramText: "file_name [options]",

	syntaxText: ` 
    ossutil probe --download --url http_url [--addr=domain_name] [file_name]
    ossutil probe --download --bucketname bucket-name  [--object=object_name] [--addr=domain_name] [file_name]
    ossutil probe --upload [file_name] --bucketname bucket-name [--object=object_name] [--addr=domain_name]
    ossutil probe --probe-item item_value --bucketname bucket-name [--object=object_name]
`,

	detailHelpText: ` 
    下载探测(--download表示)分两种情况
      1、利用--url直接输入一个url网络地址，工具会下载该链接地址到本地
      2、利用--bucketname指定某个bucket下载
        如果输入--object，则下载指定bucket的指定object
        如果不输入--object，工具生成一个临时文件上传到oss，再将其下载下来，探测结束后将临时文件和临时object删除
	
    上传探测(--upload表示)
      1、如果输入文件参数file_name，则将该文件上传到指定的bucket
        如果不输入文件参数file_name,则工具生成一个临时文件上传到bucket，探测结束后再将临时文件删除
      2、如果输入--object，则上传到oss中对象名称为object_name,如果该object已经存在，会提示是否覆盖
        如果不输入--object，则上传到oss中object名称是随机生成的，探测结束后会将该临时object删除
		   
    上述命令中，file_name是参数，下载探测表示下载文件保存的目录名或者文件名，上传探测表示文件名

    其他探测功能(--probe-item表示)
    取值cycle-symlink: 探测本地目录是否存在死循环链接文件或者目录
    取值upload-speed: 探测上传带宽
    取值download-speed: 探测下载带宽
     

--url选项

    表示一个网络地址,ossutil会下载该地址

--bucketname

    oss中bucket的名称

--object

    oss中object的名称

--addr选项

    需要网络探测的域名，工具会对该域名进行ping等操作，默认值为www.aliyun.com

--upmode选项

    表示上传模式,缺省值为normal,取值为:normal|append|multipart,分别表示正常上传、追加上传、分块上传

--probe-item选项

    表示其他探测功能的项目

用法：

    该命令有四种用法：

    1) ossutil probe --download --url http_url [--addr=domain_name] [file_name]
        该用法下载http_url地址到本地文件系统中，并输出探测报告;如果不输入file_name，则下载文
    件保存在当前目录下，文件名由工具自动判断;如果输入file_name,则file_name为文件名或者目录名，
    下载的文件名为file_name或者保存在file_name目录下。
        如果输入--addr，工具会探测domain_name, 默认探测 www.aliyun.com

    2) ossutil probe --download --bucketname bucket-name  [--object=object_name] [--addr=domain_name] [file_name]
        该用法下载bucket中的object，并输出探测报告;指定--object则会下载bucket-name中
    的object_name;不指定--object，则工具会生成一个临时文件上传到oss后再将其下载，下载结束后
    会将临时文件和临时object删除
        如果输入--addr，工具会探测domain_name, 默认探测 www.aliyun.com

    3) ossutil probe --upload [file_name] --bucketname bucket-name [--object=object_name] [--addr=domain_name] 
        该用法是上传探测,会输出探测报告;如果指定file_name,则将该file_name文件上传到oss;不指定
	file_name,则工具会生成一个临时文件上传，探测结束后将临时文件删除;如果输入--object，则oss
    中object名称为object_name;如果不输入--object，则oss中object名称为工具自动生成，探测结束
    后会将该临时object删除
        如果输入--addr，工具会探测domain_name, 默认探测 www.aliyun.com
    
    4) ossutil probe --probe-item item_value --bucketname bucket-name [--object=object_name]
       该功能通过选项--probe-item不同的取值,可以实现不同的探测功能,目前取值有cycle-symlink, upload-speed, download-speed, download-time
    分别表示本地死链检测, 探测上传带宽, 探测下载带宽, 探测下载时间
`,

	sampleText: ` 
	1) 下载指定url
        ossutil probe --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"
	
    2) 下载指定Url到指定文件
        ossutil probe --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"  file_name
	
    3) 下载指定url到指定文件、并检测指定地址网络状况
        ossutil probe  --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"  file_name --addr www.aliyun.com

    4) 下载bucket临时文件
        ossutil probe --download --bucketname bucket-name
	
    5) 下载bucket指定文件
        ossutil probe --download --bucketname bucket-name --object object_name

    6) 下载bucket指定的文件并保存到本地指定文件
        ossutil probe --download --bucketname bucket-name --object object_name  file_name

    7) 下载bucket指定文件并保存到本地指定文件，并检测指定地址网络状况
        ossutil probe --download --bucketname bucket-name --object object_name  file_name --addr www.aliyun.com
   
    8) 上传临时文件，以normal方式上传
        ossutil probe --upload --bucketname bucket-name --upmode normal

    9) 上传临时文件，以append方式上传
        ossutil probe --upload --bucketname bucket-name --upmode append

    10) 上传临时文件，以multipart方式上传
        ossutil probe --upload --bucketname bucket-name --upmode multipart

    11) 上传指定文件到指定object
        ossutil probe --upload file_name --bucketname bucket-name --object object_name

    12) 上传指定文件到指定object,并检测addr地址
        ossutil probe --upload file_name --bucketname bucket-name --object object_name --addr www.aliyun.com
    
    13) 检测本地目录dir是否存在死循环链接文件或者目录
        ossutil probe --probe-item cycle-symlink dir

    14) 探测上传带宽
        ossutil probe --probe-item upload-speed --bucketname bucket-name
    
    15) 探测下载带宽, object要已经存在,且大小最好超过5M
        ossutil probe --probe-item download-speed --bucketname bucket-name --object object_name
`,
}

var specEnglishProbe = SpecText{
	synopsisText: "Probe command, support for multiple function detection",

	paramText: "file_name [options]",

	syntaxText: ` 
	ossutil probe --download --url http_url [--addr=domain_name] [file_name]
    ossutil probe --download --bucketname bucket-name  [--object=object_name] [--addr=domain_name] [file_name]
    ossutil probe --upload [file_name] --bucketname bucket-name [--object=object_name] [--addr=domain_name] 
    ossutil probe --probe-item item_value --bucketname bucket-name [--object=object_name]
`,

	detailHelpText: ` 
	Download probe (--download) has two usages
      1、Use --url to input a url, ossutil will download the link
      2、Use --bucketname to download the specified bucket's object
        If input --object, downloads the specified object of the specified bucket.
        If do not input --object, ossutil creates a temporary file to upload to oss, then downloads it, and deletes the temporary file and temporary object after the probe ends.
	
    Upload probe (--upload)
      1、If input parameter file_name, the specified file is uploaded to the specified bucket.
         If do not input parameter file_name, ossutil creates a temporary file to upload, and then deletes the temporary file after the probe ends.
      2、If input --object, the object name is specified,if the object already exists, you will be prompted to overwrite or not.
         If do not input --object, the object name is randomly generated, and the temporary object will be deleted after the probe ends.
		   
    In the above commands, file_name is a parameter which may be a directory name or a file name in the case of download probe, and must be a exist file name in the case of upload probe

    Other detection functions(--probe-item)
    value cycle-symlink: Detects whether there is an infinite loop link file or directory in the local directory
    value upload-speed: probe upload bandwidth
    value download-speed: probe download bandwidth

--url option

    Specifies a network address which will be downloaded by ossutil

--bucketname option

    Specifies a bucket name in oss
   
--object option

    Specifies a object name in oss bucket

--addr option
    
    Specifies a domain name which will be probed by ossutil,the default value is www.aliyun.com

--upmode option

    specifies the upload mode,default value is normal,value is:normal|append|multipart

--probe-item选项

    specifies other detection functions

Usage:

    There are four usages for this command:

    1) ossutil probe --download --url http_url [--addr=domain_name] [file_name]
		
        The command downloads the http_url address to the local file system and outputs
	probe report; if you do not input file_name, the downloaded file is saved in the 
	current directory and the file name is determined by ossutil; if file_name is inputed, 
	The downloaded file is named file_name.
        If you input --addr, ossutil will probe the domain_name,default probe www.aliyun.com

    2) ossutil probe --download --bucketname bucket-name  [--object=object_name] [--addr=domain_name] [file_namefile_name]
		
        The command downloads object in the specified bucket and outputs probe report; 
	if you input --object,ossutil downloads the specified object; if you don't input --object
	,ossutil creates a temporary file to upload and then downloads it; after probe end,temporary 
    file and temporary object will all be deleted
        If you input --addr, ossutil will probe the domain_name,default probe www.aliyun.com

    3) ossutil probe --upload [file_name] --bucketname bucket-name [--object=object_name] [--addr=domain_name] 
		
        The command uploads a file to oss and outputs probe report; if you input file_name,
	the file named file_name is uploaded; if you don't input file_name,ossutil creates a temporary
	file to upload and delete it after the probe ends; if you input --object, the uploaded object
	is named object_name; if you don't input --object, the uploaded object's name is determined by 
	ossutil, and after probe end,the temporary object will be deleted
        If you input --addr, ossutil will probe the domain_name,default probe www.aliyun.com
    
    4) ossutil probe --probe-item item_value --bucketname bucket-name [--object=object_name]
        You can implement different detection functions by using the value of the option --probe-item.
    The current values are cycle-symlink, upload-speed, download-speedh and download-time which represents local dead link detection,
    probe upload bandwidth, probe download bandwidth and probe download time
`,

	sampleText: ` 
	1) downloads specified url
        ossutil probe --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"
	
    2) downloads specified url to specified file
        ossutil probe --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"  file_name
	
    3) downloads specified url to specified file,and ping domain
        ossutil probe  --download --url "http://bucket-name.oss-cn-shenzhen.aliyuncs.com/object_name"  file_name --addr www.aliyun.com

    4) downloads temporary file from specified bucket
        ossutil probe --download --bucketname bucket-name
	
    5) downloads specified object from specified bucket
        ossutil probe --download --bucketname bucket-name --object object_name

    6) downloads specified object from specified bucket to specified file
        ossutil probe --download --bucketname bucket-name --object object_name  file_name

    7) downloads specified object from specified bucket to specified file,and probe domain
        ossutil probe --download --bucketname bucket-name --object object_name  file_name --addr www.aliyun.com
   
    8) uploads a temporary file with normal mode
        ossutil probe --upload --bucketname bucket-name --upmode normal

    9) uploads a temporary file with append mode
        ossutil probe --upload --bucketname bucket-name --upmode append

    10) uploads a temporary file with multipart mode
        ossutil probe --upload --bucketname bucket-name --upmode multipart

    11) uploads specified file to specified object
        ossutil probe --upload file_name --bucketname bucket-name --object object_name

    12) uploads specified file to specified object, and probe domain
        ossutil probe --upload file_name --bucketname bucket-name --object object_name --addr www.aliyun.com
    
    13) Check if the local directory dir has an infinite loop link file or directory
        ossutil probe --probe-item cycle-symlink dir

    14) Probe upload bandwidth
        ossutil probe --probe-item upload-speed --bucketname bucket-name
    
    15) Detect download bandwidth, object must already exist, and the size is better than 5M
        ossutil probe --probe-item download-speed --bucketname bucket-name --object object_name
`,
}

type probeOptionType struct {
	disableNetDetect bool
	opUpload         bool
	opDownload       bool
	fromUrl          string
	bucketName       string
	objectName       string
	netAddr          string
	upMode           string
	logFile          *os.File
	logName          string
	dlFileSize       int64
	dlFilePath       string
	ulObject         string
	probeItem        string
}

type ProbeCommand struct {
	command  Command
	pbOption probeOptionType
}

var probeCommand = ProbeCommand{
	command: Command{
		name:        "probe",
		nameAlias:   []string{"probe"},
		minArgc:     0,
		maxArgc:     MaxInt,
		specChinese: specChineseProbe,
		specEnglish: specEnglishProbe,
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
			OptionUpload,
			OptionDownload,
			OptionUrl,
			OptionBucketName,
			OptionObject,
			OptionAddr,
			OptionUpMode,
			OptionLogLevel,
			OptionProbeItem,
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
			OptionParallel,
			OptionPartSize,
			OptionRuntime,
		},
	},
}

type TestAppendReader struct {
	RandText []byte
	bClosed  bool
}

// Read
func (r *TestAppendReader) Close() {
	r.bClosed = true
}

// Read
func (r *TestAppendReader) Read(p []byte) (n int, err error) {
	if r.bClosed {
		return 0, fmt.Errorf("ossutil probe closed")
	}

	n = copy(p, r.RandText)
	for n < len(p) {
		nn := copy(p[n:], r.RandText)
		n += nn
	}
	return n, nil
}

type AverageInfo struct {
	Parallel int
	AveSpeed float64
}

type StatBandWidth struct {
	Mu         sync.Mutex
	Parallel   int
	StartTick  int64
	TotalBytes int64
	MaxSpeed   float64
}

func (s *StatBandWidth) Reset(pc int) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.Parallel = pc
	s.StartTick = time.Now().UnixNano() / 1000 / 1000
	atomic.StoreInt64(&s.TotalBytes, 0)
	s.MaxSpeed = 0.0
}

func (s *StatBandWidth) AddBytes(bc int64) {
	atomic.AddInt64(&s.TotalBytes, bc)
}

func (s *StatBandWidth) SetMaxSpeed(ms float64) {
	s.Mu.Lock()
	defer s.Mu.Unlock()
	s.MaxSpeed = ms
}

func (s *StatBandWidth) GetStat() *StatBandWidth {
	var rs StatBandWidth
	s.Mu.Lock()
	defer s.Mu.Unlock()
	rs.Parallel = s.Parallel
	rs.StartTick = s.StartTick
	rs.TotalBytes = atomic.LoadInt64(&s.TotalBytes)
	rs.MaxSpeed = s.MaxSpeed
	return &rs
}

func (s *StatBandWidth) ProgressChanged(event *oss.ProgressEvent) {
	if event.EventType == oss.TransferDataEvent || event.EventType == oss.TransferCompletedEvent {
		s.AddBytes(event.RwBytes)
	}
}

// function for FormatHelper interface
func (pc *ProbeCommand) formatHelpForWhole() string {
	return pc.command.formatHelpForWhole()
}

func (pc *ProbeCommand) formatIndependHelp() string {
	return pc.command.formatIndependHelp()
}

// Init simulate inheritance, and polymorphism
func (pc *ProbeCommand) Init(args []string, options OptionMapType) error {
	err := pc.command.Init(args, options, pc)
	if err == nil {
		return nil
	}

	errStr := err.Error()
	if strings.Contains(errStr, "Read config file error") && isNotNeedConigFile(options) {
		return nil
	}
	return err
}

func isNotNeedConigFile(options OptionMapType) bool {
	isDownload, _ := GetBool(OptionDownload, options)
	fromUrl, _ := GetString(OptionUrl, options)
	probeItem, _ := GetString(OptionProbeItem, options)
	if (isDownload && fromUrl != "") || probeItem == "cycle-symlink" {
		return true
	}
	return false
}

// RunCommand simulate inheritance, and polymorphism
func (pc *ProbeCommand) RunCommand() error {
	var err error

	pc.pbOption.opUpload, _ = GetBool(OptionUpload, pc.command.options)
	pc.pbOption.opDownload, _ = GetBool(OptionDownload, pc.command.options)
	pc.pbOption.fromUrl, _ = GetString(OptionUrl, pc.command.options)
	pc.pbOption.bucketName, _ = GetString(OptionBucketName, pc.command.options)
	pc.pbOption.objectName, _ = GetString(OptionObject, pc.command.options)
	pc.pbOption.netAddr, _ = GetString(OptionAddr, pc.command.options)
	pc.pbOption.upMode, _ = GetString(OptionUpMode, pc.command.options)
	pc.pbOption.probeItem, _ = GetString(OptionProbeItem, pc.command.options)

	if pc.pbOption.probeItem != "" {
		var err error
		if pc.pbOption.probeItem == "cycle-symlink" {
			err = pc.CheckCycleSymlinkWithDeepTravel()
			if err == nil {
				fmt.Println("\n", "success")
			}
		} else if pc.pbOption.probeItem == "upload-speed" || pc.pbOption.probeItem == "download-speed" {
			err = pc.DetectBandWidth()
		} else if pc.pbOption.probeItem == "download-time" {
			err = pc.DetectDownloadTime()
		} else {
			err = fmt.Errorf("not support %s", pc.pbOption.probeItem)
		}
		return err
	}

	pc.pbOption.logFile, pc.pbOption.logName, err = logFileMake()
	if err != nil {
		return fmt.Errorf("probe logFileMake error,%s", err.Error())
	}
	defer pc.pbOption.logFile.Close()

	pc.pbOption.logFile.WriteString("*************************	system information	*************************\n")
	pc.pbOption.logFile.WriteString(fmt.Sprintf("operating system:%s_%s\n", runtime.GOOS, runtime.GOARCH))
	pc.pbOption.logFile.WriteString(fmt.Sprintf("operating time:%s\n", time.Now().Format("2006-01-02 15:04:05")))

	if pc.pbOption.opUpload && pc.pbOption.opDownload {
		err = fmt.Errorf("error,upload and download are both true")
	} else if !pc.pbOption.opUpload && !pc.pbOption.opDownload {
		err = fmt.Errorf("error,upload and download are both false")
	} else if !pc.pbOption.opUpload && pc.pbOption.opDownload {
		err = pc.probeDownload()
	} else {
		err = pc.probeUpload()
	}
	return err
}

func (pc *ProbeCommand) PutObjectWithContext(bucket *oss.Bucket, st *StatBandWidth, reader io.Reader, ctx context.Context) {
	for {
		var options []oss.Option
		options = append(options, oss.Progress(st), oss.WithContext(ctx))
		uniqKey := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randStr(10)
		objectName := objectPrefex + uniqKey
		err := bucket.PutObject(objectName, reader, options...)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
	}
}

func (pc *ProbeCommand) GetObjectWithContext(bucket *oss.Bucket, objectName string, st *StatBandWidth, ctx context.Context) {
	var options []oss.Option
	options = append(options, oss.Progress(st), oss.WithContext(ctx))
	options = append(options, oss.AcceptEncoding("identity"))
	for {
		result, err := bucket.DoGetObject(&oss.GetObjectRequest{ObjectKey: objectName}, options)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
			}
		}
		io.Copy(ioutil.Discard, result.Response.Body)
		result.Response.Close()
	}
}

func (pc *ProbeCommand) DetectBandWidth() error {
	if pc.pbOption.bucketName == "" {
		return fmt.Errorf("--bucketname is empty")
	}

	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return err
	}

	if pc.pbOption.probeItem == "download-speed" {
		if pc.pbOption.objectName == "" {
			return fmt.Errorf("--object is empty when probe-item is download-speed")
		}

		bExist, err := bucket.IsObjectExist(pc.pbOption.objectName)
		if err != nil {
			return err
		}

		if !bExist {
			return fmt.Errorf("oss object is not exist,%s", pc.pbOption.objectName)
		}
	}

	numCpu := runtime.NumCPU()
	var statBandwidth StatBandWidth
	statBandwidth.Reset(numCpu)

	var appendReader TestAppendReader
	if pc.pbOption.probeItem == "upload-speed" {
		appendReader.RandText = []byte(strings.Repeat("1", 32*1024))
	}
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	for i := 0; i < numCpu; i++ {
		time.Sleep(time.Duration(50) * time.Millisecond)
		if pc.pbOption.probeItem == "upload-speed" {
			go pc.PutObjectWithContext(bucket, &statBandwidth, &appendReader, ctx)
		} else if pc.pbOption.probeItem == "download-speed" {
			go pc.GetObjectWithContext(bucket, pc.pbOption.objectName, &statBandwidth, ctx)
		}
	}

	time.Sleep(time.Duration(2) * time.Second)

	fmt.Printf("cpu core count:%d\n", numCpu)
	startTick := time.Now().UnixNano() / 1000 / 1000
	nowTick := startTick
	changeTick := startTick
	nowParallel := numCpu
	addParallel := numCpu / 5
	if addParallel == 0 {
		addParallel = 1
	}
	var averageList []AverageInfo

	// ignore the first max speed
	bDiscarded := false
	var oldStat *StatBandWidth
	var nowStat *StatBandWidth

	oldStat = statBandwidth.GetStat()
	for nowParallel <= 2*numCpu {
		time.Sleep(time.Duration(1) * time.Second)
		nowStat = statBandwidth.GetStat()
		nowTick = time.Now().UnixNano() / 1000 / 1000

		nowSpeed := float64(nowStat.TotalBytes-oldStat.TotalBytes) / 1024
		averSpeed := float64(nowStat.TotalBytes/1024) / float64((nowTick-nowStat.StartTick)/1000)
		maxSpeed := nowStat.MaxSpeed
		if nowSpeed > maxSpeed {
			if !bDiscarded && maxSpeed < 0.0001 {
				bDiscarded = true
				oldStat.Reset(nowParallel)
				statBandwidth.Reset(nowParallel) //discard the first max speed,becase is not accurate
				continue
			}
			maxSpeed = nowSpeed
			statBandwidth.SetMaxSpeed(maxSpeed)
		}
		fmt.Printf("\rparallel:%d,average speed:%.2f(KB/s),current speed:%.2f(KB/s),max speed:%.2f(KB/s)", nowStat.Parallel, averSpeed, nowSpeed, maxSpeed)
		oldStat = nowStat

		// 30 second
		if nowTick-changeTick >= 30000 {
			nowParallel += addParallel
			for i := 0; i < addParallel; i++ {
				time.Sleep(time.Duration(50) * time.Millisecond)
				if pc.pbOption.probeItem == "upload-speed" {
					go pc.PutObjectWithContext(bucket, &statBandwidth, &appendReader, ctx)
				} else if pc.pbOption.probeItem == "download-speed" {
					go pc.GetObjectWithContext(bucket, pc.pbOption.objectName, &statBandwidth, ctx)
				}
			}
			fmt.Printf("\n")
			bDiscarded = false
			averageList = append(averageList, AverageInfo{Parallel: nowStat.Parallel, AveSpeed: averSpeed})
			changeTick = nowTick
			oldStat.Reset(nowParallel)
			statBandwidth.Reset(nowParallel)
		}
	}

	cancel()
	appendReader.Close()

	maxIndex := 0
	maxSpeed := 0.0
	for k, v := range averageList {
		if v.AveSpeed > maxSpeed {
			maxIndex = k
			maxSpeed = v.AveSpeed
		}
	}

	fmt.Printf("\nsuggest parallel is %d, max average speed is %.2f(KB/s)\n", averageList[maxIndex].Parallel, averageList[maxIndex].AveSpeed)

	maxRuntime, _ := GetInt(OptionRuntime, pc.command.options)

	if maxRuntime > 0 {
		time.Sleep(time.Duration(5) * time.Second)
		ctx = context.Background()
		ctx, cancel = context.WithCancel(ctx)
		addParallel = averageList[maxIndex].Parallel
		statBandwidth.Reset(addParallel)
		fmt.Printf("\nrun %s  %d seconds with parallel %d\n", pc.pbOption.probeItem, maxRuntime, addParallel)
		for i := 0; i < addParallel; i++ {
			if pc.pbOption.probeItem == "upload-speed" {
				go pc.PutObjectWithContext(bucket, &statBandwidth, &appendReader, ctx)
			} else if pc.pbOption.probeItem == "download-speed" {
				go pc.GetObjectWithContext(bucket, pc.pbOption.objectName, &statBandwidth, ctx)
			}
		}

		startT := time.Now().UnixNano() / 1000 / 1000 / 1000
		for {
			time.Sleep(time.Duration(1) * time.Second)
			nowStat = statBandwidth.GetStat()
			nowTick = time.Now().UnixNano() / 1000 / 1000

			nowSpeed := float64(nowStat.TotalBytes-oldStat.TotalBytes) / 1024
			averSpeed := float64(nowStat.TotalBytes/1024) / float64((nowTick-nowStat.StartTick)/1000)
			maxSpeed := nowStat.MaxSpeed
			if nowSpeed > maxSpeed {
				maxSpeed = nowSpeed
				statBandwidth.SetMaxSpeed(maxSpeed)
			}
			fmt.Printf("\rparallel:%d,average speed:%.2f(KB/s),current speed:%.2f(KB/s),max speed:%.2f(KB/s)", addParallel, averSpeed, nowSpeed, maxSpeed)
			oldStat = nowStat
			currT := time.Now().UnixNano() / 1000 / 1000 / 1000
			if startT+maxRuntime < currT {
				cancel()
				break
			}
		}
	}

	return nil
}

func (pc *ProbeCommand) CheckCycleSymlinkWithDeepTravel() error {
	if len(pc.command.args) == 0 {
		return fmt.Errorf("dir parameter is emtpy")
	}

	dpath := pc.command.args[0]
	fileInfo, err := os.Stat(dpath)
	if err != nil {
		return err
	}

	if !fileInfo.IsDir() {
		return nil
	}

	if !strings.HasSuffix(dpath, string(os.PathSeparator)) {
		dpath += string(os.PathSeparator)
	}

	DirStack := list.New()
	DirStack.PushBack(dpath)
	for DirStack.Len() > 0 {
		dirItem := DirStack.Back()
		DirStack.Remove(dirItem)
		dirName := dirItem.Value.(string)
		fileList, err := ioutil.ReadDir(dirName)
		if err != nil {
			return err
		}

		for _, fileInfo := range fileList {
			realInfo, err := os.Stat(dirName + fileInfo.Name())
			if err != nil {
				return err
			}
			if realInfo.IsDir() {
				DirStack.PushBack(dirName + fileInfo.Name() + string(os.PathSeparator))
			}
		}
	}
	return nil
}

func (pc *ProbeCommand) probeDownload() error {
	var err error
	pingPath := ""
	if pc.pbOption.fromUrl != "" {
		pingPath, _, err = urlCheck(pc.pbOption.fromUrl)
		if err != nil {
			return fmt.Errorf("probeDownloadWithHttpUrl error,%s", err.Error())
		}
	} else {
		if pc.pbOption.bucketName == "" {
			return fmt.Errorf("probeDownloadWithParameter error,bucketName is not exist")
		}

		endPoint, _ := pc.command.getEndpoint(pc.pbOption.bucketName)
		if endPoint == "" {
			return fmt.Errorf("probeDownloadWithParameter error,endpoint is not exist")
		}

		pSlice := strings.Split(endPoint, "//")
		if len(pSlice) == 1 {
			endPoint = pSlice[0]
		} else {
			endPoint = pSlice[1]
		}
		pingPath = endPoint
	}

	fmt.Printf("begin parse parameters and prepare object...[√]\n")

	fmt.Printf("begin network detection...")
	pc.ossNetDetection(pingPath)
	fmt.Printf("\rbegin network detection...[√]\n")

	startT := time.Now()
	if pc.pbOption.fromUrl != "" {
		err = pc.downloadWithHttpUrl()
	} else {
		err = pc.probeDownloadWithParameter()
	}
	endT := time.Now()

	var logBuff bytes.Buffer
	if err == nil {
		fmt.Printf("\rbegin download file...[√]\n")

		logBuff.WriteString("\n*************************  download result  *************************\n")
		logBuff.WriteString("download file:success\n")
		logBuff.WriteString(fmt.Sprintf("download file size:%d(byte)\n", pc.pbOption.dlFileSize))
		logBuff.WriteString(fmt.Sprintf("download time consuming:%d(ms)\n", endT.UnixNano()/1000/1000-startT.UnixNano()/1000/1000))
		logBuff.WriteString("(only the time consumed by probe command)\n\n")

		if pc.pbOption.dlFilePath != "" {
			logBuff.WriteString(fmt.Sprintf("download file is %s\n", pc.pbOption.dlFilePath))
		}

	} else {
		fmt.Printf("\rbegin download file...[x]\n\n")

		logBuff.WriteString("\n*************************  download result  *************************\n")
		logBuff.WriteString("download file:failure\n")

		logBuff.WriteString("\n*************************  error message *************************\n")
		logBuff.WriteString(fmt.Sprintf("%s\n", err.Error()))
	}

	fmt.Printf("%s", logBuff.String())
	pc.pbOption.logFile.WriteString(logBuff.String())

	fmt.Printf("\n************************* report log info*************************\n")
	fmt.Printf("report log file:%s\n\n", pc.pbOption.logName)

	return err
}

// the only arg in this command is input or output file name
func (pc *ProbeCommand) getFileNameArg() (fileName string, err error) {
	if len(pc.command.args) == 0 {
		return "", nil
	}

	fileName = pc.command.args[0]
	fileURL, err := StorageURLFromString(fileName, "")
	if err != nil {
		return "", fmt.Errorf("StorageURLFromString error:%s", err.Error())
	}

	if !fileURL.IsFileURL() {
		return "", fmt.Errorf("not a local file name:%s", fileURL.ToString())
	}
	return
}

func (pc *ProbeCommand) downloadWithHttpUrl() error {
	_, srcName, err := urlCheck(pc.pbOption.fromUrl)
	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl urlCheck error,%s", err.Error())
	}

	fileName, err := pc.getFileNameArg()
	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl getFileNameArg error,%s", err.Error())
	}

	downloadFileName, err := prepareLocalFileName(srcName, fileName)
	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl prepareLocalFileName error,%s", err.Error())
	}

	sizeStat, err := os.Stat(downloadFileName)
	if err == nil {
		bConitnue := confirm(downloadFileName)
		if !bConitnue {
			return nil
		}
	}

	res, err := http.Get(pc.pbOption.fromUrl)
	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl http.Get error,%s", err.Error())
	}
	defer res.Body.Close()

	pc.pbOption.logFile.WriteString("\n************************* response info*************************\n")
	pc.pbOption.logFile.WriteString(fmt.Sprintf("status code:%s\n", res.Status))
	res.Header.Write(pc.pbOption.logFile)

	if res.StatusCode != 200 {
		return fmt.Errorf("http status code:%s", res.Status)
	}

	fileRecord, err := os.Create(downloadFileName)
	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl http.Get error,%s", err.Error())
	}
	io.Copy(fileRecord, res.Body)
	sizeStat, err = fileRecord.Stat()
	fileRecord.Close()

	if err != nil {
		return fmt.Errorf("downloadWithHttpUrl fileRecord.Stat error,%s", err.Error())
	}

	pc.pbOption.dlFileSize = sizeStat.Size()
	pc.pbOption.dlFilePath = downloadFileName

	return nil
}

func (pc *ProbeCommand) probeDownloadWithParameter() error {
	var err error
	var srcURL CloudURL
	var bDeleteObject = false
	srcURL.bucket = pc.pbOption.bucketName
	if pc.pbOption.objectName == "" {
		uniqKey := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randStr(10)
		objectName := objectPrefex + uniqKey
		err := pc.prepareRandomObject(objectName)
		if err != nil {
			return fmt.Errorf("prepareRandomObject error,%s", err.Error())
		}

		srcURL.object = objectName
		bDeleteObject = true
	} else {
		srcURL.object = pc.pbOption.objectName
	}

	srcURL.urlStr = srcURL.ToString()
	err = pc.probeDownloadObject(srcURL, bDeleteObject)
	return err
}

func (pc *ProbeCommand) probeDownloadObject(srcURL CloudURL, bDeleteObject bool) error {
	fileName, err := pc.getFileNameArg()
	if err != nil {
		return fmt.Errorf("probeDownloadObject error,%s", err.Error())
	}

	downloadFileName, err := prepareLocalFileName(srcURL.object, fileName)
	if err != nil {
		return fmt.Errorf("probeDownloadObject error,%s", err.Error())
	} else {
		_, err := os.Stat(downloadFileName)
		if err == nil {
			bConitnue := confirm(downloadFileName)
			if !bConitnue {
				return nil
			}
		}
	}

	bucket, err := pc.command.ossBucket(srcURL.bucket)
	if err != nil {
		return fmt.Errorf("bucket:%s,probeDownloadObject error,%s", srcURL.bucket, err.Error())
	}

	err = bucket.GetObjectToFile(srcURL.object, downloadFileName)
	if err != nil {
		return fmt.Errorf("bucket:%s,GetObjectToFile error,%s", srcURL.bucket, err.Error())
	}

	sizeStat, err := os.Stat(downloadFileName)
	if err != nil {
		return fmt.Errorf("GetObjectToFile error,%s", err.Error())
	}

	pc.pbOption.dlFileSize = sizeStat.Size()
	pc.pbOption.dlFilePath = downloadFileName

	if bDeleteObject {
		pc.deleteObject(srcURL.object)
	}

	return nil
}

func (pc *ProbeCommand) prepareRandomObject(objectName string) (err error) {
	//judge objectName exist or not
	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return err
	}

	isExist, err := bucket.IsObjectExist(objectName)
	if err != nil {
		return err
	}

	if isExist {
		return fmt.Errorf("random object %s exist,please try again", objectName)
	}

	// put up object
	var textBuffer bytes.Buffer
	for i := 0; i < 10240; i++ {
		textBuffer.WriteString("testossprobe")
	}

	err = bucket.PutObject(objectName, strings.NewReader(textBuffer.String()))
	if err != nil {
		return err
	}
	return nil
}

func (pc *ProbeCommand) deleteObject(objectName string) error {
	retryTimes, _ := GetInt(OptionRetryTimes, pc.command.options)
	for i := 1; ; i++ {
		bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
		if err == nil {
			err = bucket.DeleteObject(objectName)
			if err == nil {
				return nil
			}
		}

		_, noNeedRetry := err.(oss.ServiceError)
		if int64(i) >= retryTimes || noNeedRetry {
			return err
		}

		// wait 1 second
		time.Sleep(time.Duration(1) * time.Second)
	}
}

func (pc *ProbeCommand) ossNetDetection(pingPath string) {
	if pc.pbOption.disableNetDetect {
		return // for test:reduce test time
	}

	var netAddr = "www.aliyun.com"

	if pc.pbOption.netAddr != "" {
		netAddr = pc.pbOption.netAddr
	}

	if runtime.GOOS == "windows" {
		pingProcess(pc.pbOption.logFile, "ping", netAddr)
		pingProcess(pc.pbOption.logFile, "ping", pingPath)
		pingProcess(pc.pbOption.logFile, "tracert", pingPath)
		pingProcess(pc.pbOption.logFile, "nslookup", pingPath)
	} else {
		// linux or mac
		pingProcess(pc.pbOption.logFile, "ping", netAddr, "-c", "4")
		pingProcess(pc.pbOption.logFile, "ping", pingPath, "-c", "4")
		pingProcess(pc.pbOption.logFile, "traceroute", "-m", "20", pingPath)
		pingProcess(pc.pbOption.logFile, "dig", pingPath)
	}
}

func prepareLocalFileName(srcName string, destName string) (absDestFileName string, err error) {
	absDestFileName = ""
	err = nil

	keyName := srcName
	urlSplits := strings.Split(srcName, "/")
	if len(urlSplits) > 1 {
		keyName = urlSplits[len(urlSplits)-1]
	}

	// it is absolute path
	currentDir, err := os.Getwd()
	if err != nil {
		return
	}

	if destName == "" {
		absDestFileName = currentDir + string(os.PathSeparator) + keyName
		return
	}

	// get absolute path
	absDestFileName, err = filepath.Abs(destName)
	if err != nil {
		return
	}

	if strings.HasSuffix(destName, string(os.PathSeparator)) {
		err = os.MkdirAll(absDestFileName, 0755)
		if err != nil {
			return
		}
		absDestFileName = absDestFileName + string(os.PathSeparator) + keyName
	} else {
		f, serr := os.Stat(absDestFileName)
		if serr == nil && f.IsDir() {
			absDestFileName = absDestFileName + string(os.PathSeparator) + keyName
		} else {
			err = os.MkdirAll(filepath.Dir(absDestFileName), 0755)
		}
	}
	return
}

func urlCheck(strUrl string) (string, string, error) {
	var err error
	urlSplits := strings.Split(strUrl, "/")
	if len(urlSplits) < 4 {
		err = fmt.Errorf("invalid url:%s", strUrl)
		return "", "", err
	}
	pingPath := urlSplits[2]
	urlGetFileName := urlSplits[len(urlSplits)-1]
	endPos := strings.Index(urlGetFileName, "?")
	if endPos > 0 {
		urlGetFileName = urlGetFileName[0:endPos]
	}

	if pingPath == "" || urlGetFileName == "" {
		return "", "", fmt.Errorf("invalid url:%s", strUrl)
	}

	return pingPath, urlGetFileName, nil
}

func logFileMake() (logFile *os.File, logName string, err error) {
	dirName, err := os.Getwd()
	if err != nil {
		return nil, "", err
	}

	logName = dirName + string(os.PathSeparator) + "logOssProbe" + time.Now().Format("20060102150405") + ".log"
	logFile, err = os.OpenFile(logName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		return nil, "", err
	}
	return
}

func pingProcess(logFile *os.File, instruction string, args ...string) {
	logFile.WriteString("\n\n")
	logFile.WriteString(fmt.Sprintf("*************************	%s	*************************\n", instruction))
	logFile.WriteString(fmt.Sprintf("Command => %s", instruction))
	for _, v := range args {
		logFile.WriteString(fmt.Sprintf(" %s", v))
	}
	logFile.WriteString("\n")

	c := exec.Command(instruction, args...)
	d, _ := c.Output()
	logFile.WriteString(string(d))
	c.Run()
}

func confirm(str string) bool {
	var val string
	fmt.Printf(getClearStr(fmt.Sprintf("probe: overwrite \"%s\"(y or N)? ", str)))
	if _, err := fmt.Scanln(&val); err != nil || (strings.ToLower(val) != "yes" && strings.ToLower(val) != "y") {
		return false
	}
	return true
}

func (pc *ProbeCommand) probeUpload() error {
	upMode := pc.pbOption.upMode
	if upMode == "" {
		upMode = normalMode
	} else {
		if upMode != normalMode && upMode != appendMode && upMode != multipartMode {
			return fmt.Errorf("probeUpload errro,invalid mode flag:%s", upMode)
		}
	}

	if pc.pbOption.bucketName == "" {
		return fmt.Errorf("probeUpload error,bucketName is not exist")
	}

	endPoint, _ := pc.command.getEndpoint(pc.pbOption.bucketName)
	if endPoint == "" {
		return fmt.Errorf("probeUpload error,endpoint is not exist")
	}

	pSlice := strings.Split(endPoint, "//")
	if len(pSlice) == 1 {
		endPoint = pSlice[0]
	} else {
		endPoint = pSlice[1]
	}
	pingPath := endPoint

	objectName := pc.pbOption.objectName
	srcFileName, err := pc.getFileNameArg()
	if err != nil {
		return fmt.Errorf("probeUpload errro,getFileNameArg error:%s", err.Error())
	}

	var bDeleteLocalFile = false
	fileSize := int64(0)
	if srcFileName == "" {
		// it is absolute path
		currentDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("probeUpload errro,os.Getwd error:%s", err.Error())
		}
		uniqKey := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randStr(10)
		tempName := objectPrefex + uniqKey
		srcFileName = currentDir + string(os.PathSeparator) + tempName

		_, err = os.Stat(srcFileName)
		if err == nil {
			return fmt.Errorf("temp file exist:%s,please retry", srcFileName)
		}

		// prepare a local file
		var textBuffer bytes.Buffer
		for i := 0; i < 10240; i++ {
			textBuffer.WriteString("testossprobe")
		}

		err = ioutil.WriteFile(srcFileName, textBuffer.Bytes(), 0644)
		if err != nil {
			return fmt.Errorf("prepare temp file error,%s", err.Error())
		}
		bDeleteLocalFile = true
		fileSize = int64(textBuffer.Len())
	} else {
		fStat, err := os.Stat(srcFileName)
		if err != nil {
			return fmt.Errorf("%s not exist,stat error:%s", srcFileName, err.Error())
		}

		if fStat.IsDir() {
			return fmt.Errorf("%s is dir,not file", srcFileName)
		}

		fileSize = fStat.Size()
	}

	var bDeleteObject = false
	if objectName == "" {
		uniqKey := strconv.FormatInt(time.Now().UnixNano(), 10) + "-" + randStr(10)
		objectName = objectPrefex + uniqKey
		bDeleteObject = true
	} else {
		pc.pbOption.ulObject = objectName
	}

	// judge object is exist or not
	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return fmt.Errorf("probeUpload ossBucket error:%s", err.Error())
	}

	isExist, err := bucket.IsObjectExist(objectName)
	if err != nil {
		return fmt.Errorf("probeUpload IsObjectExist error:%s", err.Error())
	}

	if isExist {
		if bDeleteObject {
			return fmt.Errorf("oss temp object %s exist,please try again", objectName)
		} else {
			bConitnue := confirm(objectName)
			if !bConitnue {
				return nil
			}
		}
	}

	fmt.Printf("begin parse parameters and prepare file...[√]\n")

	fmt.Printf("begin network detection...")
	pc.ossNetDetection(pingPath)
	fmt.Printf("\rbegin network detection...[√]\n")
	fmt.Printf("begin upload file(%s)...", upMode)

	// begin upload
	startT := time.Now()
	if upMode == appendMode {
		err = pc.probeUploadFileAppend(srcFileName, objectName)
	} else if upMode == multipartMode {
		err = pc.probeUploadFileMultiPart(srcFileName, objectName)
	} else {
		err = pc.probeUploadFileNormal(srcFileName, objectName)
	}
	endT := time.Now()

	var logBuff bytes.Buffer
	if err == nil {
		fmt.Printf("\rbegin upload file(%s)...[√]\n", upMode)

		logBuff.WriteString("\n*************************  upload result  *************************\n")
		logBuff.WriteString("upload file:success\n")
		logBuff.WriteString(fmt.Sprintf("upload file size:%d(byte)\n", fileSize))
		logBuff.WriteString(fmt.Sprintf("upload time consuming:%d(ms)\n", endT.UnixNano()/1000/1000-startT.UnixNano()/1000/1000))
		logBuff.WriteString("(only the time consumed by probe command)\n\n")

		if pc.pbOption.ulObject != "" {
			logBuff.WriteString(fmt.Sprintf("upload object is %s\n", pc.pbOption.ulObject))
		}
	} else {
		fmt.Printf("\rbegin upload file(%s)...[x]\n\n", upMode)

		logBuff.WriteString("\n*************************  upload result  *************************\n")
		logBuff.WriteString("upload file:failure\n")

		logBuff.WriteString("\n*************************  error message  *************************\n")
		logBuff.WriteString(fmt.Sprintf("%s\n", err.Error()))
	}

	fmt.Printf("%s", logBuff.String())
	pc.pbOption.logFile.WriteString(logBuff.String())

	fmt.Printf("\n************************* report log info*************************\n")
	fmt.Printf("report log file:%s\n\n", pc.pbOption.logName)

	// delete oss temp object
	if bDeleteObject {
		pc.deleteObject(objectName)
	}

	// delete local file
	if bDeleteLocalFile {
		os.Remove(srcFileName)
	}

	return err
}

func (pc *ProbeCommand) probeUploadFileAppend(absFileName string, objectName string) error {
	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return fmt.Errorf("probeUploadFileAppend error:%s", err.Error())
	}

	var nextPos int64 = 0
	fd, err := os.Open(absFileName)
	if err != nil {
		return fmt.Errorf("probeUploadFileAppend,open %s error:%s", absFileName, err.Error())
	}
	defer fd.Close()
	nextPos, err = bucket.AppendObject(objectName, fd, nextPos)
	if err != nil {
		return fmt.Errorf("probeUploadFileAppend error:%s", err.Error())
	}
	return nil
}

func (pc *ProbeCommand) probeUploadFileMultiPart(absFileName string, objectName string) error {
	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return fmt.Errorf("probeUploadFileMultiPart error:%s", err.Error())
	}

	err = bucket.UploadFile(objectName, absFileName, 100*1024, oss.Routines(5), oss.Checkpoint(true, ""))
	if err != nil {
		return fmt.Errorf("probeUploadFileMultiPart error:%s", err.Error())
	}
	return nil
}

func (pc *ProbeCommand) probeUploadFileNormal(absFileName string, objectName string) error {
	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return err
	}
	err = bucket.PutObjectFromFile(objectName, absFileName)
	if err != nil {
		return fmt.Errorf("PutObjectFromFile error:%s", err.Error())
	}
	return nil
}

type downloadPart struct {
	Index int   // Part number, starting from 0
	Start int64 // Start index
	End   int64 // End index
}

type downloadWorkerArg struct {
	bucket *oss.Bucket
	key    string
}

func getPartEnd(begin int64, total int64, per int64) int64 {
	if begin+per > total {
		return total - 1
	}
	return begin + per - 1
}

func downloadWorker(id int, arg downloadWorkerArg, jobs <-chan downloadPart, results chan<- downloadPart, failed chan<- error, die <-chan bool, st *StatBandWidth) {
	for part := range jobs {
		var options []oss.Option
		r := oss.Range(part.Start, part.End)
		options = append(options, r, oss.Progress(st))
		options = append(options, oss.AcceptEncoding("identity"))
		result, err := arg.bucket.DoGetObject(&oss.GetObjectRequest{ObjectKey: arg.key}, options)
		if err != nil {
			fmt.Printf("GetObject error,%s", err.Error())
			return
		}
		_, err = io.Copy(ioutil.Discard, result.Response.Body)
		result.Response.Close()

		select {
		case <-die:
			return
		default:
		}

		if err != nil {
			failed <- err
			break
		}
		results <- part
	}
}

func downloadScheduler(jobs chan downloadPart, parts []downloadPart) {
	for _, part := range parts {
		jobs <- part
	}
	close(jobs)
}

func (pc *ProbeCommand) DetectDownloadTime() error {
	if pc.pbOption.bucketName == "" {
		return fmt.Errorf("--bucketname is empty")
	}

	bucket, err := pc.command.ossBucket(pc.pbOption.bucketName)
	if err != nil {
		return err
	}

	//if pc.pbOption.probeItem == "download-time" {
	if pc.pbOption.objectName == "" {
		return fmt.Errorf("--object is empty when probe-item is download-time")
	}

	bExist, err := bucket.IsObjectExist(pc.pbOption.objectName)
	if err != nil {
		return err
	}

	if !bExist {
		return fmt.Errorf("oss object is not exist,%s", pc.pbOption.objectName)
	}

	meta, err := bucket.GetObjectDetailedMeta(pc.pbOption.objectName)
	if err != nil {
		return err
	}

	objectSize, err := strconv.ParseInt(meta.Get(oss.HTTPHeaderContentLength), 10, 64)
	if err != nil {
		return err
	}
	var offset int64
	//var partSize int64
	parts := []downloadPart{}
	partSize, _ := GetInt(OptionPartSize, pc.command.options)
	parallel, _ := GetInt(OptionParallel, pc.command.options)
	if parallel <= 0 {
		parallel = 1
	}

	if partSize > 0 {
		i := 0
		for offset = 0; offset < objectSize; offset += partSize {
			part := downloadPart{}
			part.Index = i
			part.Start = offset
			part.End = getPartEnd(offset, objectSize, partSize)
			parts = append(parts, part)
			i++
		}
	} else {
		part := downloadPart{}
		part.Index = 0
		part.Start = 0
		part.End = objectSize - 1
		parts = append(parts, part)
	}

	//}
	jobs := make(chan downloadPart, len(parts))
	results := make(chan downloadPart, len(parts))
	failed := make(chan error)
	die := make(chan bool)
	routines := int(parallel)
	var statBandwidth StatBandWidth
	statBandwidth.Reset(int(parallel))

	//fmt.Printf("\nDetectDownloadTime, partSize :%v, objectSize:%v, parallel:%v\n", partSize, objectSize, parallel)

	arg := downloadWorkerArg{bucket, pc.pbOption.objectName}
	for w := 1; w <= routines; w++ {
		go downloadWorker(w, arg, jobs, results, failed, die, &statBandwidth)
	}

	// Download parts concurrently
	go downloadScheduler(jobs, parts)

	go func() {
		oldStat := statBandwidth.GetStat()
		for {
			time.Sleep(time.Duration(2) * time.Second)
			nowStat := statBandwidth.GetStat()
			nowTick := time.Now().UnixNano() / 1000 / 1000

			nowSpeed := float64(nowStat.TotalBytes-oldStat.TotalBytes) / 1024
			averSpeed := float64(nowStat.TotalBytes/1024) / float64((nowTick-nowStat.StartTick)/1000)
			maxSpeed := nowStat.MaxSpeed
			if nowSpeed > maxSpeed {
				maxSpeed = nowSpeed
				statBandwidth.SetMaxSpeed(maxSpeed)
			}
			oldStat = nowStat
			fmt.Printf("\rdownloading average speed:%.2f(KB/s),current speed:%.2f(KB/s),max speed:%.2f(KB/s)", averSpeed, nowSpeed, maxSpeed)
		}
	}()

	completed := 0
	for completed < len(parts) {
		select {
		case part := <-results:
			completed++
			_ = (part.End - part.Start + 1)
		case err := <-failed:
			close(die)
			return err
		}
		if completed >= len(parts) {
			break
		}
	}

	nowTick := time.Now().UnixNano() / 1000 / 1000
	nowStat := statBandwidth.GetStat()
	averSpeed := float64(nowStat.TotalBytes/1024) / float64((nowTick-nowStat.StartTick)/1000)
	//total := float64(objectSize)

	fmt.Printf("\ndownload-speed part-size:%v, parallel:%v total bytes:%v, cost:%.3f s, avg speed:%.2f(kB/s)\n", partSize, parallel, nowStat.TotalBytes, float64(nowTick-nowStat.StartTick)/1000, averSpeed)

	return nil
}
