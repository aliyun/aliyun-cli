package lib

import (
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestSetBucketMeta(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	showElapse, err := s.rawSetMeta(bucketName, "", "X-Oss-Meta-A:A", false, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetObjectMetaBasic(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "TestSetObjectMeta_testobject"
	s.putObject(bucketName, object, uploadFileName, c)

	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "default")
	_, ok := objectStat["X-Oss-Meta-A"]
	c.Assert(ok, Equals, false)

	// update
	s.setObjectMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A#Expires:2006-01-02T15:04:05Z", true, false, false, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "A")
	c.Assert(objectStat["Expires"], Equals, "Mon, 02 Jan 2006 15:04:05 GMT")

	// error expires
	showElapse, err := s.rawSetMeta(bucketName, object, "Expires:2006-01", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["Expires"], Equals, "Mon, 02 Jan 2006 15:04:05 GMT")

	// delete
	s.setObjectMeta(bucketName, object, "x-oss-object-acl#X-Oss-Meta-A", false, true, false, true, c)
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")
	_, ok = objectStat["X-Oss-Meta-A"]
	c.Assert(ok, Equals, false)

	s.setObjectMeta(bucketName, object, "X-Oss-Meta-A:A#x-oss-meta-B:b", true, false, false, true, c)

	s.setObjectMeta(bucketName, object, "X-Oss-Meta-c:c", false, false, false, true, c)
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")

	// without force
	s.setObjectMeta(bucketName, object, "x-oss-object-acl:public-read#X-Oss-Meta-A:A", true, false, false, false, c)

	// without update, delete and force
	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, LEnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// miss meta
	s.setObjectMeta(bucketName, object, "", true, false, false, true, c)

	showElapse, err = s.rawSetMeta(bucketName, object, "", true, false, false, true, EnglishLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	showElapse, err = s.rawSetMeta(bucketName, object, "", true, false, false, true, LEnglishLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// delete error meta
	showElapse, err = s.rawSetMeta(bucketName, object, "X-Oss-Meta-A:A", false, true, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// update error meta
	showElapse, err = s.rawSetMeta(bucketName, object, "a:b", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:private", true, false, false, true, DefaultLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//batch
	s.setObjectMeta(bucketName, "", "content-type:abc#X-Oss-Meta-Update:update", true, false, true, false, c)

	s.setObjectMeta(bucketName, "", "content-type:abc#X-Oss-Meta-Update:update", true, false, true, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["Content-Type"], Equals, "abc")
	c.Assert(objectStat["X-Oss-Meta-Update"], Equals, "update")

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-update", false, true, true, true, c)

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-A:A#x-oss-meta-B:b", true, false, true, true, c)

	s.setObjectMeta(bucketName, "nosetmeta", "X-Oss-Meta-M:c", false, false, true, true, c)

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-C:c", false, false, true, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Meta-C"], Equals, "c")

	showElapse, err = s.rawSetMeta(bucketName, "", "X-Oss-Meta-c:c", false, true, true, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, "", "a:b", true, false, true, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetNotExistObjectMeta(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "testobject-notexistone"
	// set meta of not exist object
	showElapse, err := s.rawSetMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// batch set meta of not exist objects
	s.setObjectMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A", true, false, true, true, c)

	showElapse, err = s.rawGetStat(bucketName, object)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	object = "testobject"
	s.putObject(bucketName, object, uploadFileName, c)

	s.setObjectMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A", true, false, false, true, c)

	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "A")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrBatchSetMeta(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put objects
	num := 10
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestErrBatchSetMeta_setmeta:%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// update without force
	meta := "content-type:abc#X-Oss-Meta-Update:update"
	args := []string{CloudURLToString(bucketName, ""), meta}
	command := "set-meta"
	str := ""
	str1 := "abc"
	ok := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str1,
		"accessKeyID":     &str1,
		"accessKeySecret": &str1,
		"stsToken":        &str,
		"update":          &ok,
		"recursive":       &ok,
		"force":           &ok,
		"routines":        &routines,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Update"]
		c.Assert(ok, Equals, false)
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrSetMeta(c *C) {
	args := []string{"os:/", ""}
	showElapse, err := s.rawSetMetaWithArgs(args, false, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta("", "", "", false, false, false, false, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "notexistobject"

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A", true, true, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMetaWithArgs([]string{CloudURLToString(bucketName, object)}, false, false, false, true, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMetaWithArgs([]string{CloudURLToString(bucketName, object)}, true, false, false, false, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMetaWithArgs([]string{CloudURLToString(bucketName, object)}, true, false, false, false, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A", false, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, "", "x-oss-object-acl:private#X-Oss-Meta-A:A", false, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMetaWithArgs([]string{"oss:///object", "x-oss-object-acl:private#X-Oss-Meta-A:A"}, false, false, true, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMetaWithArgs([]string{CloudURLToString("", "")}, true, false, false, false, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "unknown:a", true, false, false, true, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "Expires:a", true, false, false, true, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetOSSOption(c *C) {
	_, err := getOSSOption(headerOptionMap, "unknown", "a")
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSetMetaIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "testobject"
	s.putObject(bucketName, object, uploadFileName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "set-meta"
	str := ""
	args := []string{CloudURLToString(bucketName, object), "x-oss-object-acl:private#X-Oss-Meta-A:A#Expires:2006-01-02T15:04:05Z"}
	ok := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"update":          &ok,
		"force":           &ok,
		"routines":        &routines,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"update":          &ok,
		"force":           &ok,
		"routines":        &routines,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetMetaURLEncoding(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "^M特殊字符 加上空格 test"
	s.putObject(bucketName, object, uploadFileName, c)

	urlObject := url.QueryEscape(object)

	showElapse, err := s.rawSetMeta(bucketName, urlObject, "x-oss-object-acl:private#X-Oss-Meta-A:A#Expires:2006-01-02T15:04:05Z", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	command := "set-meta"
	str := ""
	args := []string{CloudURLToString(bucketName, urlObject), "x-oss-object-acl:private#X-Oss-Meta-A:A#Expires:2006-01-02T15:04:05Z"}
	ok := true
	routines := strconv.Itoa(Routines)
	encodingType := URLEncodingType
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &ok,
		"force":           &ok,
		"routines":        &routines,
		"encodingType":    &encodingType,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetMetaErrArgs(c *C) {
	object := randStr(20)
	meta := "x-oss-object-acl:public-read-write"

	err := s.initSetMetaWithArgs([]string{CloudURLToString("", object), meta}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setMetaCommand.RunCommand()
	c.Assert(err, NotNil)

	err = s.initSetMetaWithArgs([]string{CloudURLToString("", ""), meta}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setMetaCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchSetMetaNotExistBucket(c *C) {
	// set acl notexist bucket
	meta := "x-oss-object-acl:public-read-write"
	err := s.initSetMetaWithArgs([]string{CloudURLToString(bucketNamePrefix+randLowStr(10), ""), meta}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setMetaCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchSetMetaErrorContinue(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	meta := "x-oss-object-acl:public-read-write"

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("设置元信息object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	err := s.initSetMetaWithArgs([]string{CloudURLToString(bucketName, ""), meta}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	bucket, err := setMetaCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	c.Assert(bucket, NotNil)

	setMetaCommand.monitor.init("Setted meta on")
	setMetaCommand.smOption.ctnu = true

	// init reporter
	setMetaCommand.smOption.reporter, err = GetReporter(setMetaCommand.smOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer setMetaCommand.smOption.reporter.Clear()

	var routines int64
	routines = 3
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	chObjects <- objectNames[0]
	chObjects <- "notexistobject" + randStr(3)
	chObjects <- objectNames[1]
	chListError <- nil
	close(chObjects)

	headers := map[string]string{}
	headers[oss.HTTPHeaderOssObjectACL] = "public-read-write"

	for i := 0; int64(i) < routines; i++ {
		setMetaCommand.setObjectMetaConsumer(bucket, headers, false, false, chObjects, chError)
	}

	err = setMetaCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, IsNil)

	str := setMetaCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := setMetaCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.errNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(3))

	setMetaCommand.monitor.seekAheadEnd = true
	setMetaCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	setMetaCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	setMetaCommand.monitor.seekAheadEnd = true
	setMetaCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	setMetaCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, "public-read-write")
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchSetMetaErrorBreak(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	meta := "x-oss-object-acl:public-read-write"

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("设置元信息object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	err := s.initSetMetaWithArgs([]string{CloudURLToString(bucketName, ""), meta}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	// make error bucket with error id
	bucket := s.getErrorOSSBucket(bucketName, c)
	c.Assert(bucket, NotNil)

	setMetaCommand.monitor.init("Setted meta on")
	setMetaCommand.smOption.ctnu = true

	// init reporter
	setMetaCommand.smOption.reporter, err = GetReporter(setMetaCommand.smOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer setMetaCommand.smOption.reporter.Clear()

	var routines int64
	routines = 3
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	chObjects <- objectNames[0]
	chObjects <- objectNames[1]
	chListError <- nil
	close(chObjects)

	headers := map[string]string{}
	headers[oss.HTTPHeaderOssObjectACL] = "public-read-write"

	for i := 0; int64(i) < routines; i++ {
		setMetaCommand.setObjectMetaConsumer(bucket, headers, false, false, chObjects, chError)
	}

	err = setMetaCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, NotNil)

	str := setMetaCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = setMetaCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := setMetaCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	setMetaCommand.monitor.seekAheadEnd = true
	setMetaCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	setMetaCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	setMetaCommand.monitor.seekAheadEnd = true
	setMetaCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	setMetaCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setMetaCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, "default")
	}

	s.removeBucket(bucketName, true, c)
}

// Test: --include '*.txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNormalInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc1"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	// e.g., ossutil set-meta oss://tempb4/ "content-type:abc#X-Oss-Meta-test:with-filter" -rf --include "*.txt"
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*.txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.txt")
	exFiles := filterStrsWithExclude(objs, "*.txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*2??txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithMarkInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc2"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	// e.g., ossutil set-meta oss://tempb4/ "content-type:abc#X-Oss-Meta-test:with-filter" -rf --include "*.txt"
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*2??txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*2??txt")
	exFiles := filterStrsWithExclude(objs, "*2??txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestSetObjectMetaWithSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc3"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*[0-9]?jpg"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[0-9]?jpg")
	exFiles := filterStrsWithExclude(objs, "*[0-9]?jpg")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNonSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc4"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*[^0-3]?txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[^0-3]?txt")
	exFiles := filterStrsWithExclude(objs, "*[^0-3]?txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNonSequenceIncludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc4-ex"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*[!0-3]?txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	inFiles := filterStrsWithInclude(objs, filters[0].pattern)
	exFiles := filterStrsWithExclude(objs, filters[0].pattern)

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --include '*.jpg'
func (s *OssutilCommandSuite) TestSetObjectMetaWithRepeatedInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc5"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*.jpg", "--include", "*.jpg"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.jpg")
	exFiles := filterStrsWithExclude(objs, "*.jpg")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*'
func (s *OssutilCommandSuite) TestSetObjectMetaWithFullInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc6"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:abc#X-Oss-Meta-test:with-filter-include"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*")
	exFiles := filterStrsWithExclude(objs, "*")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "abc")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-include")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "abc", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*.txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNormalExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	// e.g., ossutil set-meta oss://tempb4/ "content-type:abc#X-Oss-Meta-test:with-filter" -rf --exclude "*.txt"
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*.txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.txt")
	exFiles := filterStrsWithExclude(objs, "*.txt")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*2??txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithMarkExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-mark"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	// e.g., ossutil set-meta oss://tempb4/ "content-type:abc#X-Oss-Meta-test:with-filter" -rf --include "*.txt"
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*2??txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*2??txt")
	exFiles := filterStrsWithExclude(objs, "*2??txt")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestSetObjectMetaWithSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-sequence"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*[0-9]?jpg"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[0-9]?jpg")
	exFiles := filterStrsWithExclude(objs, "*[0-9]?jpg")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNonSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-nonsequence"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*[^0-3]?txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[^0-3]?txt")
	exFiles := filterStrsWithExclude(objs, "*[^0-3]?txt")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectMetaWithNonSequenceExcludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-nonsequence-ex"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*[!0-3]?txt"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	inFiles := filterStrsWithInclude(objs, filters[0].pattern)
	exFiles := filterStrsWithExclude(objs, filters[0].pattern)

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --exclude '*.jpg'
func (s *OssutilCommandSuite) TestSetObjectMetaWithRepeatedExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-repeated"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*.jpg", "--exclude", "*.jpg"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.jpg")
	exFiles := filterStrsWithExclude(objs, "*.jpg")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*'
func (s *OssutilCommandSuite) TestSetObjectMetaWithFullExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-exclude"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*")
	exFiles := filterStrsWithExclude(objs, "*")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-exclude")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
		_, ok := objectStat["X-Oss-Meta-Test"]
		c.Assert(ok, Equals, false)
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestSetObjectMetaWithMultiNormalIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-inc-exc"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*.txt", "--exclude", "*2*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-inc-exc")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
			_, ok := objectStat["X-Oss-Meta-Test"]
			c.Assert(ok, Equals, false)
		}
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test repeated: --include '*.txt' --exclude "*2*" --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestSetObjectMetaWithMultiRepeatedIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-inc-exc"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*.txt", "--exclude", "*2*", "--include", "*.txt", "--exclude", "*2*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}, {"--include", "*.txt"}, {"--exclude", "*2*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-inc-exc")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
			_, ok := objectStat["X-Oss-Meta-Test"]
			c.Assert(ok, Equals, false)
		}
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*' --exclude "*"
func (s *OssutilCommandSuite) TestSetObjectMetaWithMultiFullIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-inc-exc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-inc-exc"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--include", "*", "--exclude", "*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*"}, {"--exclude", "*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-inc-exc")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
			_, ok := objectStat["X-Oss-Meta-Test"]
			c.Assert(ok, Equals, false)
		}
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*' --include "*"
func (s *OssutilCommandSuite) TestSetObjectMetaWithMultiFullExcludeInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-exc-inc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set meta
	meta := "content-type:xyz#X-Oss-Meta-test:with-filter-inc-exc"
	args := []string{bucketStr, meta}
	cmdline := []string{"ossutil", "set-meta", bucketStr, meta, "-rf", "--exclude", "*", "--include", "*"}

	showElapse, err := s.rawSetMetaWithFilter(args, true, false, true, true, DefaultLanguage, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--exclude", "*"}, {"--include", "*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["Content-Type"], Equals, "xyz")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-filter-inc-exc")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["Content-Type"] != "xyz", Equals, true)
			_, ok := objectStat["X-Oss-Meta-Test"]
			c.Assert(ok, Equals, false)
		}
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Nagetive test
func (s *OssutilCommandSuite) TestSetObjectMetaWithInvalidIncExc(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setmeta-invalid"
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	subdir := "dir1"
	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	// e.g., ossutil set-meta oss://tempb4 -f --exclude '*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-f", "--exclude", "*.txt"}
	showElapse, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	// e.g., ossutil set-meta oss://tempb4 -f --include '*.txt' --exclude "*2*"
	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include", "*.txt", "--exclude", "/*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}


func (s *OssutilCommandSuite) TestSetObjectMetaVersionBasic(c *C) {
	bucketName := bucketNamePrefix + "-set-meta-" + randLowStr(10)
	s.putBucket(bucketName, c)

	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	object := "TestSetObjectMeta_testobject"
	s.putObject(bucketName, object, uploadFileName, c)

	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "default")
	_, ok := objectStat["X-Oss-Meta-A"]
	c.Assert(ok, Equals, false)

	// update
	s.setObjectMeta(bucketName, object, "x-oss-object-acl:private#X-Oss-Meta-A:A#Expires:2006-01-02T15:04:05Z", true, false, false, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "A")
	c.Assert(objectStat["Expires"], Equals, "Mon, 02 Jan 2006 15:04:05 GMT")

	// error expires
	showElapse, err := s.rawSetMeta(bucketName, object, "Expires:2006-01", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["Expires"], Equals, "Mon, 02 Jan 2006 15:04:05 GMT")

	// delete
	s.setObjectMeta(bucketName, object, "x-oss-object-acl#X-Oss-Meta-A", false, true, false, true, c)
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")
	_, ok = objectStat["X-Oss-Meta-A"]
	c.Assert(ok, Equals, false)

	s.setObjectMeta(bucketName, object, "X-Oss-Meta-A:A#x-oss-meta-B:b", true, false, false, true, c)

	s.setObjectMeta(bucketName, object, "X-Oss-Meta-c:c", false, false, false, true, c)
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "private")

	// without force
	s.setObjectMeta(bucketName, object, "x-oss-object-acl:public-read#X-Oss-Meta-A:A", true, false, false, false, c)

	// without update, delete and force
	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, EnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:default#X-Oss-Meta-A:A", false, false, false, false, LEnglishLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// miss meta
	s.setObjectMeta(bucketName, object, "", true, false, false, true, c)

	showElapse, err = s.rawSetMeta(bucketName, object, "", true, false, false, true, EnglishLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	showElapse, err = s.rawSetMeta(bucketName, object, "", true, false, false, true, LEnglishLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// delete error meta
	showElapse, err = s.rawSetMeta(bucketName, object, "X-Oss-Meta-A:A", false, true, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// update error meta
	showElapse, err = s.rawSetMeta(bucketName, object, "a:b", true, false, false, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, object, "x-oss-object-acl:private", true, false, false, true, DefaultLanguage)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//batch
	s.setObjectMeta(bucketName, "", "content-type:abc#X-Oss-Meta-Update:update", true, false, true, false, c)

	s.setObjectMeta(bucketName, "", "content-type:abc#X-Oss-Meta-Update:update", true, false, true, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["Content-Type"], Equals, "abc")
	c.Assert(objectStat["X-Oss-Meta-Update"], Equals, "update")

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-update", false, true, true, true, c)

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-A:A#x-oss-meta-B:b", true, false, true, true, c)

	s.setObjectMeta(bucketName, "nosetmeta", "X-Oss-Meta-M:c", false, false, true, true, c)

	s.setObjectMeta(bucketName, "", "X-Oss-Meta-C:c", false, false, true, true, c)

	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Meta-C"], Equals, "c")

	showElapse, err = s.rawSetMeta(bucketName, "", "X-Oss-Meta-c:c", false, true, true, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawSetMeta(bucketName, "", "a:b", true, false, true, true, DefaultLanguage)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetObjectMetaWithVersion(c *C) {
	bucketName := bucketNamePrefix + "-set-meta-" + randLowStr(10)
	objectName := randStr(12)
	
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// put object v1
	textBufferV1 := randStr(100)
	s.createFile(uploadFileName, textBufferV1, c)
	s.putObject(bucketName, objectName, uploadFileName, c)
	objectStat := s.getStat(bucketName, objectName, c)
	versionIdV1 := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionIdV1) > 0, Equals, true)
	_, ok := objectStat["X-Oss-Meta-A"]
	c.Assert(ok, Equals, false)

	// update object to v2 
	s.setObjectMeta(bucketName, objectName, "x-oss-object-acl:private#X-Oss-Meta-A:A#X-Oss-Meta-B:B", false, false, false, true, c)
	objectStat = s.getStat(bucketName, objectName, c)
	versionIdV2 := objectStat["X-Oss-Version-Id"]
	c.Assert(objectStat[StatACL], Equals, "private")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "A")
	c.Assert(objectStat["X-Oss-Meta-B"], Equals, "B")

	// update object to v3
	s.setObjectMeta(bucketName, objectName, "x-oss-object-acl:public-read#X-Oss-Meta-A:C#X-Oss-Meta-B:D", false, false, false, true, c)
	objectStat = s.getStat(bucketName, objectName, c)
	c.Assert(objectStat[StatACL], Equals, "public-read")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "C")
	c.Assert(objectStat["X-Oss-Meta-B"], Equals, "D")

	// begin set-meta from v2
	var str string
	update := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"update":          &update,
		"versionId":       &versionIdV2,
	}

	args := []string{CloudURLToString(bucketName, objectName), "x-oss-object-acl:public-read-write#X-Oss-Meta-A:123#X-Oss-Meta-C:456"}
	_, err := cm.RunCommand("set-meta", args, options)
	c.Assert(err, IsNil)

	objectStat = s.getStat(bucketName, objectName, c)
	c.Assert(objectStat[StatACL], Equals, "public-read-write")
	c.Assert(objectStat["X-Oss-Meta-A"], Equals, "123")
	c.Assert(objectStat["X-Oss-Meta-B"], Equals, "B")
	c.Assert(objectStat["X-Oss-Meta-C"], Equals, "456")

	s.removeBucket(bucketName, true, c)
}


func (s *OssutilCommandSuite) TestSetObjectMetaWithInvalidVersionArgs(c *C) {
	bucketName := bucketNamePrefix + "-set-alc-" + randLowStr(10)
	objectName := randStr(12)
	versionId := "test"

	var str string
	flag := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"allVersions":     &flag,
	}

	args := []string{CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("set-meta", args, options)
	c.Assert(strings.Contains(err.Error(), "the command does not support option: \"allVersions\""), Equals, true)

	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &flag,
		"versionId":       &versionId,
	}

	args = []string{CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("set-meta", args, options)
	c.Assert(strings.Contains(err.Error(), "--version-id only work on single object"), Equals, true)
}