package lib

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestUploadProgressBar(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// single file
	udir := randStr(11)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)
	object := randStr(11)

	num := 2
	len := 0
	for i := 0; i < num; i++ {
		filePath := randStr(10)
		s.createFile(udir+string(os.PathSeparator)+filePath, randStr((i+3)*30*num), c)
		len += (i + 3) * 30 * num
	}
	time.Sleep(sleepTime)

	// init copyCommand
	err = s.initCopyCommand(udir, CloudURLToString(bucketName, object), true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	str := copyCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = copyCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = copyCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = copyCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(len) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(num) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(len))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(len))
	c.Assert(snap.fileNum, Equals, int64(num))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(num))
	c.Assert(snap.dealNum, Equals, int64(num))
	c.Assert(copyCommand.monitor.getPrecent(snap) == 100 || copyCommand.monitor.getPrecent(snap) == 0, Equals, true)

	time.Sleep(time.Second)
	str = strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", num)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)

	// mkdir a subdir in dir
	subdir := udir + string(os.PathSeparator) + "subdir"
	subdir1 := udir + string(os.PathSeparator) + "subdir1"
	os.RemoveAll(subdir)
	os.RemoveAll(subdir1)
	err = os.MkdirAll(subdir, 0755)
	c.Assert(err, IsNil)
	err = os.MkdirAll(subdir1, 0755)
	c.Assert(err, IsNil)

	// put file to subdir
	num1 := 2
	len1 := 0
	for i := 0; i < num1; i++ {
		filePath := randStr(10)
		s.createFile(subdir+string(os.PathSeparator)+filePath, randStr((i+1)*20*num1), c)
		len1 += (i + 1) * 20 * num1
	}

	// init copyCommand
	err = s.initCopyCommand(udir, CloudURLToString(bucketName, object), true, true, true, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// copy with update
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	str = copyCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(len+len1) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(num+num1+2) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(len1))
	c.Assert(snap.skipSize, Equals, int64(len))
	c.Assert(snap.dealSize, Equals, int64(len+len1))
	c.Assert(snap.fileNum, Equals, int64(num1))
	c.Assert(snap.dirNum, Equals, int64(2))
	c.Assert(snap.skipNum, Equals, int64(num))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(num+num1+2))
	c.Assert(snap.dealNum, Equals, int64(num+num1+2))
	c.Assert(copyCommand.monitor.getPrecent(snap) == 100 || copyCommand.monitor.getPrecent(snap) == 0, Equals, true)

	time.Sleep(time.Second)
	str = strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", snap.dealNum)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, true)
	c.Assert(strings.Contains(str, "directories"), Equals, true)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, true)
	c.Assert(strings.Contains(str, "directories"), Equals, true)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	os.RemoveAll(udir)

	snap = copyCommand.monitor.getSnapshot()
	snap.skipSize = 1
	snap.transferSize = 0
	str = strings.ToLower(copyCommand.monitor.getSizeDetail(snap))
	c.Assert(strings.Contains(str, "skip size:"), Equals, true)
	c.Assert(strings.Contains(str, "transfer size:"), Equals, false)
	snap.transferSize = 1
	str = strings.ToLower(copyCommand.monitor.getSizeDetail(snap))
	c.Assert(strings.Contains(str, "ok size:"), Equals, true)
	c.Assert(strings.Contains(str, "transfer"), Equals, true)
	c.Assert(strings.Contains(str, "skip"), Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDownloadProgressBar(c *C) {
	s.createFile(uploadFileName, "", c)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	// normal download single object of content length 0
	err := s.initCopyCommand(CloudURLToString(bucketName, object), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	str := copyCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(0))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))
	c.Assert(copyCommand.monitor.getPrecent(snap) == 100 || copyCommand.monitor.getPrecent(snap) == 0, Equals, true)

	time.Sleep(time.Second)
	str = strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 1)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCopyProgressBar(c *C) {
	s.createFile(uploadFileName, randStr(15), c)
	srcBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(srcBucket, c)
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)
	num := 2
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestCopyProgressBar%d", i)
		s.putObject(srcBucket, object, uploadFileName, c)
	}

	// normal download single object of content length 0
	err := s.initCopyCommand(CloudURLToString(srcBucket, "TestCopyProgressBar"), CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	str := copyCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(30))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(30))
	c.Assert(snap.fileNum, Equals, int64(2))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))
	c.Assert(copyCommand.monitor.getPrecent(snap) == 100 || copyCommand.monitor.getPrecent(snap) == 0, Equals, true)

	time.Sleep(time.Second)
	str = strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 2)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	s.removeBucket(srcBucket, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestProgressBarStatisticErr(c *C) {
	// test batch download err
	s.createFile(uploadFileName, "TestProgressBarStatisticErr", c)
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	num := 2
	for i := 0; i < num; i++ {
		object := randStr(10)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(configFile, data, c)

	err := s.initCopyCommand(CloudURLToString(bucketName, ""), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	os.Remove(configFile)
	configFile = cfile

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(0))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))

	time.Sleep(time.Second)
	str := strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("scanned num: %d", snap.dealNum)), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, false)
	c.Assert(strings.Contains(str, "progress"), Equals, false)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, false)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "succeed"), Equals, false)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, false)
	c.Assert(strings.Contains(str, "when error happens"), Equals, true)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, false)

	str1 := strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str1)), Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestProgressBarContinueErr(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	udir := randStr(11)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)

	num := 2
	filePaths := []string{}
	for i := 0; i < num; i++ {
		filePath := randStr(10)
		s.createFile(udir+"/"+filePath, fmt.Sprintf("测试文件：%d内容", i), c)
		filePaths = append(filePaths, filePath)
	}

	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", "abc", accessKeyID, accessKeySecret)
	s.createFile(configFile, data, c)

	err = s.initCopyCommand(udir, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	os.Remove(configFile)
	configFile = cfile

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "error"), Equals, true)
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(0))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(num))
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num))

	time.Sleep(time.Second)
	str := strings.ToLower(copyCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", snap.dealNum)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	c.Assert(strings.Contains(str, "skip"), Equals, false)
	c.Assert(strings.Contains(str, "directories"), Equals, false)

	str = strings.ToLower(copyCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror"), Equals, true)
	c.Assert(strings.Contains(str, "succeed"), Equals, false)
	c.Assert(strings.Contains(str, fmt.Sprintf("error num: %d", snap.errNum)), Equals, true)

	os.RemoveAll(udir)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSingleFileProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(10)
	destObject := object + randStr(10)

	// single large file
	data := strings.Repeat("a", 10240)
	s.createFile(uploadFileName, data, c)

	for threshold := range []int64{1024, DefaultBigFileThreshold} {
		// init copyCommand
		err := s.initCopyCommand(uploadFileName, CloudURLToString(bucketName, object), false, true, false, int64(threshold), CheckpointDir, DefaultOutputDir)
		c.Assert(err, IsNil)
		copyCommand.monitor.init(operationTypePut)

		snap := copyCommand.monitor.getSnapshot()
		c.Assert(snap.transferSize, Equals, int64(0))
		c.Assert(snap.skipSize, Equals, int64(0))
		c.Assert(snap.dealSize, Equals, int64(0))
		c.Assert(snap.fileNum, Equals, int64(0))
		c.Assert(snap.dirNum, Equals, int64(0))
		c.Assert(snap.skipNum, Equals, int64(0))
		c.Assert(snap.errNum, Equals, int64(0))
		c.Assert(snap.okNum, Equals, int64(0))
		c.Assert(snap.dealNum, Equals, int64(0))

		time.Sleep(time.Second)
		str := strings.ToLower(copyCommand.monitor.getProgressBar())
		c.Assert(strings.Contains(str, "total num"), Equals, false)
		c.Assert(strings.Contains(str, "scanned"), Equals, true)
		c.Assert(strings.Contains(str, "error"), Equals, false)
		c.Assert(strings.Contains(str, "progress"), Equals, false)
		c.Assert(strings.Contains(str, "skip"), Equals, false)
		c.Assert(strings.Contains(str, "directories"), Equals, false)
		c.Assert(strings.Contains(str, "upload"), Equals, false)
		c.Assert(strings.Contains(str, "download"), Equals, false)
		c.Assert(strings.Contains(str, "copy"), Equals, false)

		// check output
		testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
		out := os.Stdout
		os.Stdout = testResultFile
		err = copyCommand.RunCommand()
		c.Assert(err, IsNil)
		os.Stdout = out
		pstr := strings.ToLower(s.readFile(resultPath, c))
		c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
		c.Assert(strings.Contains(pstr, "error"), Equals, false)

		snap = copyCommand.monitor.getSnapshot()
		c.Assert(snap.transferSize, Equals, int64(10240))
		c.Assert(snap.skipSize, Equals, int64(0))
		c.Assert(snap.dealSize, Equals, int64(10240))
		c.Assert(snap.fileNum, Equals, int64(1))
		c.Assert(snap.dirNum, Equals, int64(0))
		c.Assert(snap.skipNum, Equals, int64(0))
		c.Assert(snap.errNum, Equals, int64(0))
		c.Assert(snap.okNum, Equals, int64(1))
		c.Assert(snap.dealNum, Equals, int64(1))

		time.Sleep(time.Second)
		str = strings.ToLower(copyCommand.monitor.getProgressBar())
		c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 1)), Equals, true)
		c.Assert(strings.Contains(str, "error"), Equals, false)
		c.Assert(strings.Contains(str, "skip"), Equals, false)
		c.Assert(strings.Contains(str, "directories"), Equals, false)
		c.Assert(strings.Contains(str, "upload"), Equals, true)
		c.Assert(strings.Contains(str, "download"), Equals, false)
		c.Assert(strings.Contains(str, "copy"), Equals, false)
		time.Sleep(sleepTime)

		// download
		err = s.initCopyCommand(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1024, CheckpointDir, DefaultOutputDir)
		c.Assert(err, IsNil)
		copyCommand.monitor.init(operationTypeGet)

		testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
		out = os.Stdout
		os.Stdout = testResultFile
		err = copyCommand.RunCommand()
		c.Assert(err, IsNil)
		os.Stdout = out
		pstr = strings.ToLower(s.readFile(resultPath, c))
		c.Assert(strings.Contains(pstr, "error"), Equals, false)

		snap = copyCommand.monitor.getSnapshot()
		c.Assert(snap.transferSize, Equals, int64(10240))
		c.Assert(snap.skipSize, Equals, int64(0))
		c.Assert(snap.dealSize, Equals, int64(10240))
		c.Assert(snap.fileNum, Equals, int64(1))
		c.Assert(snap.dirNum, Equals, int64(0))
		c.Assert(snap.skipNum, Equals, int64(0))
		c.Assert(snap.errNum, Equals, int64(0))
		c.Assert(snap.okNum, Equals, int64(1))
		c.Assert(snap.dealNum, Equals, int64(1))

		time.Sleep(time.Second)
		str = strings.ToLower(copyCommand.monitor.getProgressBar())
		c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 1)), Equals, true)
		c.Assert(strings.Contains(str, "error"), Equals, false)
		c.Assert(strings.Contains(str, "skip"), Equals, false)
		c.Assert(strings.Contains(str, "directories"), Equals, false)
		c.Assert(strings.Contains(str, "upload"), Equals, false)
		c.Assert(strings.Contains(str, "download"), Equals, true)
		c.Assert(strings.Contains(str, "copy"), Equals, false)

		// copy
		err = s.initCopyCommand(CloudURLToString(bucketName, object), CloudURLToString(bucketName, destObject), false, true, false, 1024, CheckpointDir, DefaultOutputDir)
		c.Assert(err, IsNil)
		copyCommand.monitor.init(operationTypeCopy)

		testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
		out = os.Stdout
		os.Stdout = testResultFile
		err = copyCommand.RunCommand()
		c.Assert(err, IsNil)
		os.Stdout = out
		pstr = strings.ToLower(s.readFile(resultPath, c))
		c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
		c.Assert(strings.Contains(pstr, "error"), Equals, false)

		snap = copyCommand.monitor.getSnapshot()
		c.Assert(snap.transferSize, Equals, int64(10240))
		c.Assert(snap.skipSize, Equals, int64(0))
		c.Assert(snap.dealSize, Equals, int64(10240))
		c.Assert(snap.fileNum, Equals, int64(1))
		c.Assert(snap.dirNum, Equals, int64(0))
		c.Assert(snap.skipNum, Equals, int64(0))
		c.Assert(snap.errNum, Equals, int64(0))
		c.Assert(snap.okNum, Equals, int64(1))
		c.Assert(snap.dealNum, Equals, int64(1))

		time.Sleep(time.Second)
		str = strings.ToLower(copyCommand.monitor.getProgressBar())
		c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 1)), Equals, true)
		c.Assert(strings.Contains(str, "error"), Equals, false)
		c.Assert(strings.Contains(str, "skip"), Equals, false)
		c.Assert(strings.Contains(str, "directories"), Equals, false)
		c.Assert(strings.Contains(str, "upload"), Equals, false)
		c.Assert(strings.Contains(str, "download"), Equals, false)
		c.Assert(strings.Contains(str, "copy"), Equals, true)

		// copy skip
		err = s.initCopyCommand(CloudURLToString(bucketName, object), CloudURLToString(bucketName, destObject), false, true, true, 1024, CheckpointDir, DefaultOutputDir)
		c.Assert(err, IsNil)
		copyCommand.monitor.init(operationTypeCopy)

		testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
		out = os.Stdout
		os.Stdout = testResultFile
		err = copyCommand.RunCommand()
		c.Assert(err, IsNil)
		os.Stdout = out
		pstr = strings.ToLower(s.readFile(resultPath, c))
		c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
		c.Assert(strings.Contains(pstr, "error"), Equals, false)

		snap = copyCommand.monitor.getSnapshot()
		c.Assert(snap.transferSize, Equals, int64(0))
		c.Assert(snap.skipSize, Equals, int64(10240))
		c.Assert(snap.dealSize, Equals, int64(10240))
		c.Assert(snap.fileNum, Equals, int64(0))
		c.Assert(snap.dirNum, Equals, int64(0))
		c.Assert(snap.skipNum, Equals, int64(1))
		c.Assert(snap.errNum, Equals, int64(0))
		c.Assert(snap.okNum, Equals, int64(1))
		c.Assert(snap.dealNum, Equals, int64(1))

		time.Sleep(time.Second)
		str = strings.ToLower(copyCommand.monitor.getProgressBar())
		c.Assert(strings.Contains(str, fmt.Sprintf("num: %d", 1)), Equals, true)
		c.Assert(strings.Contains(str, "error"), Equals, false)
		c.Assert(strings.Contains(str, "skip"), Equals, true)
		c.Assert(strings.Contains(str, "directories"), Equals, false)
		c.Assert(strings.Contains(str, "upload"), Equals, false)
		c.Assert(strings.Contains(str, "download"), Equals, false)
	}
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetACLProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestSetACLProgress%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}
	time.Sleep(time.Second)

	// set object acl without -r -> no progress
	err := s.initSetACL(bucketName, objectNames[0], "private", false, false, true)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = setACLCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap := setACLCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))

	// batch set object acl -> progress
	err = s.initSetACL(bucketName, "TestSetACLProgress", "private", true, false, true)
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = setACLCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap = setACLCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(num))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num))
	c.Assert(setACLCommand.monitor.getPrecent(snap) == 100 || setACLCommand.monitor.getPrecent(snap) == 0, Equals, true)

	str := strings.ToLower(setACLCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", 2)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	str = strings.ToLower(setACLCommand.monitor.progressBar(true, normalExit))
	c.Assert(str, Equals, "")

	// batch set acl list error
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", endpoint, accessKeyID, "")
	s.createFile(configFile, data, c)

	err = s.initSetACL(bucketName, "TestSetACLProgress", "private", true, false, true)
	c.Assert(err, IsNil)

	os.Remove(configFile)
	configFile = cfile

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = setACLCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	snap = setACLCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))

	str = strings.ToLower(setACLCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, false)

	setACLCommand.monitor.init("Setted acl on")
	setACLCommand.command.updateMonitor(err, &setACLCommand.monitor)
	c.Assert(setACLCommand.monitor.errNum, Equals, int64(1))
	c.Assert(setACLCommand.monitor.okNum, Equals, int64(0))

	str = strings.ToLower(setACLCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "when error happens"), Equals, true)
	c.Assert(strings.Contains(str, "setted acl on 0 objects"), Equals, true)

	setACLCommand.monitor.init("Setted acl on")
	snap = setACLCommand.monitor.getSnapshot()
	c.Assert(setACLCommand.monitor.getPrecent(snap) == 0, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetMetaProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	num := 2
	objectNames := []string{}
	prefix := randLowStr(10)
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("%s%d", prefix, i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// set object meta without -r -> no progress
	err := s.initSetMeta(bucketName, objectNames[0], "x-oss-object-acl:default#X-Oss-Meta-A:A", true, false, false, true, DefaultLanguage)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = setMetaCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap := setMetaCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))

	// batch set object acl -> progress
	err = s.initSetMeta(bucketName, prefix, "x-oss-object-acl:default#X-Oss-Meta-A:A", true, false, true, true, DefaultLanguage)
	c.Assert(err, IsNil)

	setMetaCommand.monitor.init("Setted meta on")

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = setMetaCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap = setMetaCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(num))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num))
	c.Assert(setMetaCommand.monitor.getPrecent(snap) == 100 || setMetaCommand.monitor.getPrecent(snap) == 0, Equals, true)

	str := strings.ToLower(setMetaCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", 2)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	// batch set acl list error
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", endpoint, accessKeyID, "")
	s.createFile(configFile, data, c)

	err = s.initSetMeta(bucketName, prefix, "x-oss-object-acl:default#X-Oss-Meta-A:A", true, false, true, true, DefaultLanguage)
	c.Assert(err, IsNil)

	os.Remove(configFile)
	configFile = cfile

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = setMetaCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	snap = setMetaCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))

	str = strings.ToLower(setMetaCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, false)

	setMetaCommand.monitor.init("Setted meta on")
	setMetaCommand.command.updateMonitor(err, &setMetaCommand.monitor)
	c.Assert(setMetaCommand.monitor.errNum, Equals, int64(1))
	c.Assert(setMetaCommand.monitor.okNum, Equals, int64(0))

	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "when error happens"), Equals, true)
	c.Assert(strings.Contains(str, "setted meta on 0 objects"), Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveSingleProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// remove single not exist object
	object := randStr(10)
	err := s.initRemove(bucketName, object, "rm -f")
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, fmt.Sprintf("%d objects", 0)), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(objectType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap := removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(0))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.getPrecent(snap) == 100 || removeCommand.monitor.getPrecent(snap) == 0, Equals, true)

	str := strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("total %d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	str = strings.ToLower(removeCommand.monitor.progressBar(true, normalExit))
	c.Assert(str, Equals, "")

	// remove single exist object
	s.putObject(bucketName, object, uploadFileName, c)

	err = s.initRemove(bucketName, object, "rm -f")
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, fmt.Sprintf("%d objects", 1)), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(1))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(1))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.getPrecent(snap) == 100 || removeCommand.monitor.getPrecent(snap) == 0, Equals, true)

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", 1)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("total %d objects", 1)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", 1)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchRemoveProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// batch remove not exist objects
	err := s.initRemove(bucketName, "TestBatchRemoveProgresssss", "rm -rf")
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, fmt.Sprintf("%d objects", 0)), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(objectType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap := removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(0))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))

	str := strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("total %d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", 0)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	// remove single exist object
	num := 2
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestBatchRemoveProgress%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	err = s.initRemove(bucketName, "TestBatchRemoveProgress", "rm -rf")
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(num))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.getPrecent(snap) == 100 || removeCommand.monitor.getPrecent(snap) == 0, Equals, true)

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("%d objects", num)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("total %d objects", num)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", num)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	removeCommand.monitor.init()
	removeCommand.updateObjectMonitor(0, 1)
	c.Assert(removeCommand.monitor.objectNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.uploadIdNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.errObjectNum, Equals, int64(1))
	c.Assert(removeCommand.monitor.errUploadIdNum, Equals, int64(0))

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.TrimSpace(str), Equals, "")

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.TrimSpace(str), Equals, "")

	removeCommand.monitor.setOP(objectType)
	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "when error happens"), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects", 0)), Equals, true)

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("scanned %d objects", 1)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	c.Assert(strings.Contains(str, "progress"), Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveUploadIdProgress(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucket, _ := removeCommand.command.ossBucket(bucketName)

	// rm -marf
	err := s.initRemove(bucketName, "", "rm -marf")
	c.Assert(err, IsNil)
	removeCommand.RunCommand()

	// rm -m without object, error
	err = s.initRemove(bucketName, "", "rm -m")
	c.Assert(err, IsNil)
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)
	c.Assert(strings.Contains(pstr, fmt.Sprintf("total %d objects", 0)), Equals, false)

	// rm -a without object, error
	err = s.initRemove(bucketName, "", "rm -a")
	c.Assert(err, IsNil)
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	object := randStr(10)
	num := 10
	for i := 0; i < num; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}
	// put object
	s.putObject(bucketName, object, uploadFileName, c)

	// rm -mb, error
	err = s.initRemove(bucketName, "", "rm -mb")
	c.Assert(err, IsNil)
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)

	// rm -ab, error
	err = s.initRemove(bucketName, "", "rm -ab")
	c.Assert(err, IsNil)
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)

	// rm -m single object
	err = s.initRemove(bucketName, object, "rm -m")
	c.Assert(err, IsNil)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(multipartType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap := removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(0))
	c.Assert(snap.uploadIdNum, Equals, int64(num))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num))
	c.Assert(snap.errNum, Equals, int64(0))

	s.getObject(bucketName, object, downloadFileName, c)

	str := strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", num)), Equals, true)
	c.Assert(strings.Contains(str, "objects"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, "objects"), Equals, false)
	c.Assert(strings.Contains(str, fmt.Sprintf("%d uploadids", num)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", num)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	// rm -a
	for i := 0; i < num; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}
	// put object
	object1 := object + "1"
	s.putObject(bucketName, object1, uploadFileName, c)
	for i := 0; i < num; i++ {
		_, err = bucket.InitiateMultipartUpload(object1)
		c.Assert(err, IsNil)
	}

	err = s.initRemove(bucketName, object, "rm -a")
	c.Assert(err, IsNil)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(allType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(1))
	c.Assert(snap.uploadIdNum, Equals, int64(num))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num+1))
	c.Assert(snap.errNum, Equals, int64(0))

	s.getObject(bucketName, object1, downloadFileName, c)
	lmr, e := bucket.ListMultipartUploads(oss.Prefix(object1))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, num)

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects, %d uploadids", 1, num)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d objects, %d uploadids", 1, num)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	// rm -arf
	err = s.initRemove(bucketName, object, "rm -arf")
	c.Assert(err, IsNil)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, true)
	c.Assert(strings.Contains(pstr, "error"), Equals, false)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(allType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(1))
	c.Assert(snap.uploadIdNum, Equals, int64(num))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(num+1))
	c.Assert(snap.errNum, Equals, int64(0))

	// progress
	removeCommand.monitor.init()
	removeCommand.monitor.setOP(multipartType)
	removeCommand.monitor.updateUploadIdNum(2)
	removeCommand.monitor.updateErrUploadIdNum(1)
	c.Assert(removeCommand.monitor.objectNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.uploadIdNum, Equals, int64(2))
	c.Assert(removeCommand.monitor.errObjectNum, Equals, int64(0))
	c.Assert(removeCommand.monitor.errUploadIdNum, Equals, int64(1))

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("scanned %d uploadids", 3)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", 2)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	c.Assert(strings.Contains(str, "progress"), Equals, false)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "when error happens"), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", 2)), Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveBucketProgress(c *C) {
	// remove not exist bucket
	err := s.initRemove(bucketNameNotExist, "", "rm -bf")
	c.Assert(err, IsNil)

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "succeed"), Equals, false)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	bucket, _ := removeCommand.command.ossBucket(bucketName)
	err = s.initRemove(bucketName, "", "rm -marf")
	c.Assert(err, IsNil)
	removeCommand.RunCommand()

	// rm -mrb
	object := "TestRemoveBucketProgress"
	s.putObject(bucketName, object, uploadFileName, c)
	num := 10
	for i := 0; i < num; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}
	object1 := "another_object"
	s.putObject(bucketName, object1, uploadFileName, c)
	for i := 0; i < num; i++ {
		_, err = bucket.InitiateMultipartUpload(object1)
		c.Assert(err, IsNil)
	}

	err = s.initRemove(bucketName, "", "rm -mrbf")
	c.Assert(err, IsNil)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, NotNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "error"), Equals, true)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(multipartType|bucketType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap := removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(0))
	c.Assert(snap.uploadIdNum, Equals, int64(2*num))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(2*num))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.removedBucket, Equals, "")

	str := strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", 2*num)), Equals, true)

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, fmt.Sprintf("total %d uploadids", 2*num)), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("removed %d uploadids", 2*num)), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)

	// rm -marf
	err = s.initRemove(bucketName, "", "rm -marf")
	c.Assert(err, IsNil)
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)

	c.Assert(int64(removeCommand.monitor.op), Equals, int64(allType))
	c.Assert(removeCommand.monitor.removedBucket, Equals, "")

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(snap.objectNum, Equals, int64(2))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(2))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.removedBucket, Equals, "")

	// rm -bf
	err = s.initRemove(bucketName, "", "rm -bf")
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out = os.Stdout
	os.Stdout = testResultFile
	err = removeCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out
	pstr = strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, fmt.Sprintf("removed bucket: %s", bucketName)), Equals, true)

	snap = removeCommand.monitor.getSnapshot()
	c.Assert(int64(removeCommand.monitor.op), Equals, int64(bucketType))
	c.Assert(snap.objectNum, Equals, int64(0))
	c.Assert(snap.uploadIdNum, Equals, int64(0))
	c.Assert(snap.errObjectNum, Equals, int64(0))
	c.Assert(snap.errUploadIdNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.removedBucket, Equals, bucketName)

	c.Assert(removeCommand.monitor.getOKInfo(snap), Equals, "")

	str = strings.ToLower(removeCommand.monitor.getProgressBar())
	c.Assert(strings.TrimSpace(str), Equals, "")

	str = strings.ToLower(removeCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(pstr, fmt.Sprintf("removed bucket: %s", bucketName)), Equals, true)
	c.Assert(strings.Contains(strings.TrimSpace(pstr), strings.TrimSpace(str)), Equals, true)
}

func (s *OssutilCommandSuite) TestSnapshot(c *C) {
	// upload with snapshot
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	data := randStr(20)
	s.createFile(uploadFileName, data, c)
	object := randStr(10)
	spath := "ossutil.snapshot-dir" + randStr(6)
	os.RemoveAll(spath)

	err := s.initCopyWithSnapshot(uploadFileName, CloudURLToString(bucketName, object), false, false, false, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	c.Assert(copyCommand.monitor.fileNum, Equals, int64(1))
	c.Assert(copyCommand.monitor.dirNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.skipNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.errNum, Equals, int64(0))

	s.getObject(bucketName, object, downloadFileName, c)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	_, err = os.Stat(spath)
	c.Assert(err, IsNil)

	// upload again
	err = s.initCopyWithSnapshot(uploadFileName, CloudURLToString(bucketName, object), false, false, false, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	c.Assert(copyCommand.monitor.fileNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.dirNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.skipNum, Equals, int64(1))
	c.Assert(copyCommand.monitor.errNum, Equals, int64(0))

	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	_, err = os.Stat(spath)
	c.Assert(err, IsNil)

	// modify local and upload again
	time.Sleep(time.Second)
	data = randStr(21)
	s.createFile(uploadFileName, data, c)

	err = s.initCopyWithSnapshot(uploadFileName, CloudURLToString(bucketName, object), false, false, false, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	c.Assert(copyCommand.monitor.fileNum, Equals, int64(1))
	c.Assert(copyCommand.monitor.dirNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.skipNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.errNum, Equals, int64(0))

	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	_, err = os.Stat(spath)
	c.Assert(err, IsNil)

	// -u --snapshot-path
	time.Sleep(time.Second)
	s.createFile(uploadFileName, data, c)
	err = s.initCopyWithSnapshot(uploadFileName, CloudURLToString(bucketName, object), false, true, true, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	c.Assert(copyCommand.monitor.fileNum, Equals, int64(1))
	c.Assert(copyCommand.monitor.dirNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.skipNum, Equals, int64(0))
	c.Assert(copyCommand.monitor.errNum, Equals, int64(0))

	// download with snapshot:success
	err = s.initCopyWithSnapshot(CloudURLToString(bucketName, object), downloadFileName, false, false, false, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)

	// copy with snapshot:error
	err = s.initCopyWithSnapshot(CloudURLToString(bucketName, object), CloudURLToString(bucketNameDest, object), false, false, false, DefaultBigFileThreshold, spath)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)

	os.RemoveAll(spath)

	// snapshot path exist and invalid
	err = s.initCopyWithSnapshot(uploadFileName, CloudURLToString(bucketName, object), false, false, false, DefaultBigFileThreshold, uploadFileName)
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRangeGet(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := randStr(20)
	s.createFile(uploadFileName, data, c)

	// put object
	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	// test range put
	err := s.initCopyWithRange(uploadFileName, CloudURLToString(bucketName, object), false, true, false, DefaultBigFileThreshold, "1-2")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)

	// test range copy
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), CloudURLToString(bucketName, object+"dest"), false, true, false, DefaultBigFileThreshold, "1-2")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)

	// test range get
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	snap := copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(20) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(1) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(20))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(20))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "abc")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, ",1-2")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "1-2,")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[1:3])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "1-2,a-c")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[1:3])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "1-2,abc")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[1:3])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, ",abc,1-2")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "1-5")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[1:6])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(5))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(5))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "4-20")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "4-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[4:])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(16))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(16))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "19-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[19:])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "20-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "-6")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[14:])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(6))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(6))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	os.Remove(downloadFileName)
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "-0")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)
	_, err = os.Stat(downloadFileName)
	c.Assert(err, NotNil)

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize, Equals, int64(0))
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(0))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(1))
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(1))

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "--0")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, DefaultBigFileThreshold, "3-8")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[3:9])

	// skip download
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, true, DefaultBigFileThreshold, "10-15")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[3:9])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(6))
	c.Assert(snap.dealSize, Equals, int64(6))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(1))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	// test bigfile range get
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "-")
	/*c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
	*/

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "abc")
	/*c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
	*/

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, ",1-2")
	/*
		c.Assert(err, IsNil)
		err = copyCommand.RunCommand()
		c.Assert(err, IsNil)
		str = s.readFile(downloadFileName, c)
		c.Assert(str, Equals, data)
	*/

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "1-5")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[1:6])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(5))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(5))
	c.Assert(snap.fileNum, Equals, int64(1))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "4-20")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "4-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[4:])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "19-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[19:])

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "20-")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "-5")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[15:])

	os.Remove(downloadFileName)
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "-0")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, NotNil)
	_, err = os.Stat(downloadFileName)
	c.Assert(err, NotNil)

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "--0")
	/*c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
	*/

	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, "3-8")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[3:9])

	// skip download
	err = s.initCopyWithRange(CloudURLToString(bucketName, object), downloadFileName, false, true, true, 1, "10-15")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data[3:9])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(6) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(1) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(6))
	c.Assert(snap.dealSize, Equals, int64(6))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(1))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(1))

	// test batch download with range get
	data1 := randStr(30)
	s.createFile(uploadFileName, data1, c)

	object1 := randStr(10)
	s.putObject(bucketName, object1, uploadFileName, c)

	dir := randStr(10)

	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, false, 1, "3-9")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(dir+string(os.PathSeparator)+object, c)
	c.Assert(str, Equals, data[3:10])
	str = s.readFile(dir+string(os.PathSeparator)+object1, c)
	c.Assert(str, Equals, data1[3:10])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(14) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(2) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(14))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(14))
	c.Assert(snap.fileNum, Equals, int64(2))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, false, DefaultBigFileThreshold, "3-20")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(dir+string(os.PathSeparator)+object, c)
	c.Assert(str, Equals, data)
	str = s.readFile(dir+string(os.PathSeparator)+object1, c)
	c.Assert(str, Equals, data1[3:21])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(38) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(2) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(38))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(38))
	c.Assert(snap.fileNum, Equals, int64(2))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, false, DefaultBigFileThreshold, "-5")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(dir+string(os.PathSeparator)+object, c)
	c.Assert(str, Equals, data[15:])
	str = s.readFile(dir+string(os.PathSeparator)+object1, c)
	c.Assert(str, Equals, data1[25:])

	fmt.Println(bucketName)
	fmt.Println(dir)
	fmt.Println(object)
	fmt.Println(object1)

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(10) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(2) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(10))
	c.Assert(snap.skipSize, Equals, int64(0))
	c.Assert(snap.dealSize, Equals, int64(10))
	c.Assert(snap.fileNum, Equals, int64(2))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, false, DefaultBigFileThreshold, "-20")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(dir+string(os.PathSeparator)+object, c)
	c.Assert(str, Equals, data)
	str = s.readFile(dir+string(os.PathSeparator)+object1, c)
	c.Assert(str, Equals, data1[10:])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(snap.transferSize, Equals, int64(40))
	c.Assert(snap.dealSize, Equals, int64(40))
	c.Assert(snap.fileNum, Equals, int64(2))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	// batch download with skip
	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, true, DefaultBigFileThreshold, "10-15")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	str = s.readFile(dir+string(os.PathSeparator)+object, c)
	c.Assert(str, Equals, data)
	str = s.readFile(dir+string(os.PathSeparator)+object1, c)
	c.Assert(str, Equals, data1[10:])

	snap = copyCommand.monitor.getSnapshot()
	c.Assert(copyCommand.monitor.totalSize == int64(12) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.totalNum == int64(2) || !copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(snap.transferSize, Equals, int64(0))
	c.Assert(snap.skipSize, Equals, int64(12))
	c.Assert(snap.dealSize, Equals, int64(12))
	c.Assert(snap.fileNum, Equals, int64(0))
	c.Assert(snap.dirNum, Equals, int64(0))
	c.Assert(snap.skipNum, Equals, int64(2))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	os.RemoveAll(dir)
	err = s.initCopyWithRange(CloudURLToString(bucketName, ""), dir, true, true, false, DefaultBigFileThreshold, "-0")
	c.Assert(err, IsNil)
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	_, err = os.Stat(dir + string(os.PathSeparator) + object)
	c.Assert(err, NotNil)
	_, err = os.Stat(dir + string(os.PathSeparator) + object1)
	c.Assert(err, NotNil)

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadObjectForProgressBarShowSpeed(c *C) {
	oldSecondCount := processTickInterval
	processTickInterval = 2

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// single dir
	udir := randStr(11)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)

	var logBuffer bytes.Buffer
	for i := 0; i < 200*1024; i++ {
		logBuffer.WriteString("hellow world.!")
	}

	tempFile1 := randStr(10) + "1"
	tempFile2 := randStr(10) + "2"
	s.createFile(udir+string(os.PathSeparator)+tempFile1, logBuffer.String(), c)
	s.createFile(udir+string(os.PathSeparator)+tempFile2, logBuffer.String(), c)

	// init copyCommand
	err = s.initCopyCommand(udir, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir, DefaultOutputDir)
	c.Assert(err, IsNil)

	// check output
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = copyCommand.RunCommand()
	c.Assert(err, IsNil)
	os.Stdout = out

	// check progress bar file
	pstr := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(pstr, "upload 2 files"), Equals, true)

	err = os.Remove(udir + string(os.PathSeparator) + tempFile1)
	if err != nil {
		fmt.Printf("Remove error:%s.\n", err.Error())
	}

	err = os.Remove(udir + string(os.PathSeparator) + tempFile2)
	if err != nil {
		fmt.Printf("Remove error:%s.\n", err.Error())
	}

	err = os.RemoveAll(udir)
	if err != nil {
		fmt.Printf("Remove error:%s.\n", err.Error())
	}

	s.removeBucket(bucketName, true, c)
	processTickInterval = oldSecondCount
}
