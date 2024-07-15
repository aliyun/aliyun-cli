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

func (s *OssutilCommandSuite) TestSetBucketACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// set acl
	for _, acl := range []string{"private", "public-read", "public-read-write"} {
		s.setBucketACL(bucketName, acl, c)
		s.getStat(bucketName, "", c)
	}
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetBucketACLBucketNotExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	acl := "private"
	_, err := s.rawSetBucketACL(bucketName, acl, false)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSetBucketErrorACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	for _, acl := range []string{"default", "def", "erracl", "私有"} {
		showElapse, err := s.rawSetBucketACL(bucketName, acl, false)
		c.Assert(err, NotNil)
		c.Assert(showElapse, Equals, false)

		showElapse, err = s.rawSetBucketACL(bucketName, acl, true)
		c.Assert(err, NotNil)
		c.Assert(showElapse, Equals, false)

		bucketStat := s.getStat(bucketName, "", c)
		c.Assert(bucketStat[StatACL], Equals, "private")
	}
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetNotExistBucketACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	// set bucket acl will be invalid when bucket not exist
	showElapse, err := s.rawSetBucketACL(bucketName, "public-read", true)
	c.Assert(err, NotNil)

	// invalid bucket name
	bucketName = "a"
	showElapse, err = s.rawSetBucketACL(bucketName, "public-read", true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawGetStat(bucketName, "")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestSetObjectACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "TestSetObjectACL"

	// set acl to not exist object
	showElapse, err := s.rawSetObjectACL(bucketName, object, "default", false, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	object = "setacl-oldobject"
	s.putObject(bucketName, object, uploadFileName, c)

	//get object acl
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "default")

	object = "setacl-newobject"
	s.putObject(bucketName, object, uploadFileName, c)

	// set acl
	for _, acl := range []string{"default"} {
		s.setObjectACL(bucketName, object, acl, false, true, c)
		objectStat = s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, acl)
	}

	s.setObjectACL(bucketName, object, "private", false, true, c)

	// set error acl
	for _, acl := range []string{"public_read", "erracl", "私有", ""} {
		showElapse, err = s.rawSetObjectACL(bucketName, object, acl, false, false)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, NotNil)
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchSetObjectACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put objects
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestBatchSetObjectACL_setacl%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// without --force option
	s.setObjectACL(bucketName, "", "public-read-write", true, false, c)

	s.setObjectACL(bucketName, "TestBatchSetObjectACL_setacl", "public-read", true, true, c)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, "public-read")
	}

	showElapse, err := s.rawSetObjectACL(bucketName, "TestBatchSetObjectACL_setacl", "erracl", true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrSetACL(c *C) {
	acl := "private"
	args := []string{"os://", acl}
	showElapse, err := s.rawSetACLWithArgs(args, false, false, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	args = []string{"oss://", acl}
	showElapse, err = s.rawSetACLWithArgs(args, false, false, true)

	// error set bucket acl
	bucketName := bucketNamePrefix + randLowStr(10)
	object := "testobject"
	args = []string{CloudURLToString(bucketName, object), acl}
	showElapse, err = s.rawSetACLWithArgs(args, false, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	args = []string{CloudURLToString(bucketName, ""), acl}
	showElapse, err = s.rawSetACLWithArgs(args, true, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// miss acl
	args = []string{CloudURLToString(bucketName, "")}
	showElapse, err = s.rawSetACLWithArgs(args, false, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	args = []string{CloudURLToString(bucketName, object)}
	showElapse, err = s.rawSetACLWithArgs(args, false, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	args = []string{CloudURLToString(bucketName, object)}
	showElapse, err = s.rawSetACLWithArgs(args, true, false, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// miss object
	args = []string{CloudURLToString(bucketName, ""), acl}
	showElapse, err = s.rawSetACLWithArgs(args, false, false, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// bad prefix
	showElapse, err = s.rawSetObjectACL(bucketName, "/object", acl, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestErrBatchSetACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put objects
	num := 10
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("TestErrBatchSetACL_setacl:%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	command := "set-acl"
	str := ""
	str1 := "abc"
	args := []string{CloudURLToString(bucketName, ""), "public-read-write"}
	routines := strconv.Itoa(Routines)
	ok := true
	options := OptionMapType{
		"endpoint":        &str1,
		"accessKeyID":     &str1,
		"accessKeySecret": &str1,
		"stsToken":        &str,
		"routines":        &routines,
		"recursive":       &ok,
		"force":           &ok,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, "default")
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetACLIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "set-acl"
	str := ""
	args := []string{CloudURLToString(bucketName, ""), "public-read"}
	routines := strconv.Itoa(Routines)
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"routines":        &routines,
		"bucket":          &ok,
		"force":           &ok,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"routines":        &routines,
		"bucket":          &ok,
		"force":           &ok,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetACLURLEncoding(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "^M特殊字符 加上空格 test"
	s.putObject(bucketName, object, uploadFileName, c)

	urlObject := url.QueryEscape(object)

	showElapse, err := s.rawSetObjectACL(bucketName, urlObject, "default", false, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	command := "set-acl"
	str := ""
	args := []string{CloudURLToString(bucketName, urlObject), "public-read"}
	routines := strconv.Itoa(Routines)
	ok := true
	encodingType := URLEncodingType
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"force":           &ok,
		"encodingType":    &encodingType,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetACLErrArgs(c *C) {
	object := randStr(20)

	err := s.initSetACLWithArgs([]string{CloudURLToString("", object), "private"}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setACLCommand.RunCommand()
	c.Assert(err, NotNil)

	err = s.initSetACLWithArgs([]string{CloudURLToString("", ""), "private"}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setACLCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchSetACLNotExistBucket(c *C) {
	// set acl notexist bucket
	err := s.initSetACLWithArgs([]string{CloudURLToString(bucketNamePrefix+randLowStr(10), ""), "private"}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = setACLCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchSetACLErrorContinue(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("设置权限object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// set acl prepare
	acl := oss.ACLPrivate

	err := s.initSetACLWithArgs([]string{CloudURLToString(bucketName, ""), string(acl)}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	bucket, err := setACLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	c.Assert(bucket, NotNil)

	setACLCommand.monitor.init("Setted acl on")
	setACLCommand.saOption.ctnu = true

	// init reporter
	setACLCommand.saOption.reporter, err = GetReporter(setACLCommand.saOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer setACLCommand.saOption.reporter.Clear()

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

	for i := 0; int64(i) < routines; i++ {
		setACLCommand.setObjectACLConsumer(bucket, acl, chObjects, chError)
	}

	err = setACLCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, IsNil)

	str := setACLCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := setACLCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.errNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(3))

	setACLCommand.monitor.seekAheadEnd = true
	setACLCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	setACLCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	setACLCommand.monitor.seekAheadEnd = true
	setACLCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	setACLCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, string(acl))
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchSetACLErrorBreak(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("设置权限object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// prepare
	acl := oss.ACLPrivate

	err := s.initSetACLWithArgs([]string{CloudURLToString(bucketName, ""), string(acl)}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	// make error bucket with error id
	bucket := s.getErrorOSSBucket(bucketName, c)
	c.Assert(bucket, NotNil)

	setACLCommand.monitor.init("Setted acl on")
	setACLCommand.saOption.ctnu = true

	// init reporter
	setACLCommand.saOption.reporter, err = GetReporter(setACLCommand.saOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer setACLCommand.saOption.reporter.Clear()

	var routines int64
	routines = 3
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	chObjects <- objectNames[0]
	chObjects <- objectNames[1]
	chListError <- nil
	close(chObjects)

	for i := 0; int64(i) < routines; i++ {
		setACLCommand.setObjectACLConsumer(bucket, acl, chObjects, chError)
	}

	err = setACLCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, NotNil)

	str := setACLCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = setACLCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := setACLCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	setACLCommand.monitor.seekAheadEnd = true
	setACLCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	setACLCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	setACLCommand.monitor.seekAheadEnd = true
	setACLCommand.monitor.seekAheadError = nil
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	setACLCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(setACLCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat[StatACL], Equals, "default")
	}

	s.removeBucket(bucketName, true, c)
}

// Test: --include '*.txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNormalInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc1"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set acl
	// e.g., ossutil set-acl oss://tempb4/ public-read -rf --include "*.txt"
	acl := "public-read"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*.txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.txt")
	exFiles := filterStrsWithExclude(objs, "*.txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*2??txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithMarkInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc2"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	// Set acl
	// e.g., ossutil set-acl oss://tempb4/ public-read -rf --include "*2??txt"
	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*2??txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*2??txt")
	exFiles := filterStrsWithExclude(objs, "*2??txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestSetObjectAclWithSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc3"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*[0-9]?jpg"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[0-9]?jpg")
	exFiles := filterStrsWithExclude(objs, "*[0-9]?jpg")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNonSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc4"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*[^0-3]?txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[^0-3]?txt")
	exFiles := filterStrsWithExclude(objs, "*[^0-3]?txt")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNonSequenceIncludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc4-ex"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*[!0-3]?txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	inFiles := filterStrsWithInclude(objs, filters[0].pattern)
	exFiles := filterStrsWithExclude(objs, filters[0].pattern)

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --include '*.jpg'
func (s *OssutilCommandSuite) TestSetObjectAclWithRepeatedInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc5"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*.jpg", "--include", "*.jpg"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.jpg")
	exFiles := filterStrsWithExclude(objs, "*.jpg")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*'
func (s *OssutilCommandSuite) TestSetObjectAclWithFullInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc6"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read-write"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*")
	exFiles := filterStrsWithExclude(objs, "*")

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read-write")
	}

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*.txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNormalExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*.txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.txt")
	exFiles := filterStrsWithExclude(objs, "*.txt")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*2??txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithMarkExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-mark"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*2??txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*2??txt")
	exFiles := filterStrsWithExclude(objs, "*2??txt")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestSetObjectAclWithSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-sequence"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*[0-9]?jpg"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[0-9]?jpg")
	exFiles := filterStrsWithExclude(objs, "*[0-9]?jpg")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNonSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-nonsequence"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*[^0-3]?txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*[^0-3]?txt")
	exFiles := filterStrsWithExclude(objs, "*[^0-3]?txt")
	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestSetObjectAclWithNonSequenceExcludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-nonsequence-ex"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*[!0-3]?txt"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	inFiles := filterStrsWithInclude(objs, filters[0].pattern)
	exFiles := filterStrsWithExclude(objs, filters[0].pattern)

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --exclude '*.jpg'
func (s *OssutilCommandSuite) TestSetObjectAclWithRepeatedExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-repeated"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*.jpg", "--exclude", "*.jpg"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*.jpg")
	exFiles := filterStrsWithExclude(objs, "*.jpg")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*'
func (s *OssutilCommandSuite) TestSetObjectAclWithFullExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "private"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	inFiles := filterStrsWithInclude(objs, "*")
	exFiles := filterStrsWithExclude(objs, "*")

	for _, obj := range exFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "private")
	}

	for _, obj := range inFiles {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "default")
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestSetObjectAclWithMultiNormalIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*.txt", "--exclude", "*2*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["ACL"], Equals, "default")
		}
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test repeated: --include '*.txt' --exclude "*2*" --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestSetObjectAclWithMultiRepeatedIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc-exc-normal"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*.txt", "--exclude", "*2*", "--include", "*.txt", "--exclude", "*2*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}, {"--include", "*.txt"}, {"--exclude", "*2*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["ACL"], Equals, "default")
		}
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*' --exclude "*"
func (s *OssutilCommandSuite) TestSetObjectAclWithMultiFullIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-inc-exc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--include", "*", "--exclude", "*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--include", "*"}, {"--exclude", "*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["ACL"], Equals, "default")
		}
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*' --include "*"
func (s *OssutilCommandSuite) TestSetObjectAclWithMultiFullExcludeInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-exc-inc-full"
	subdir := "subdir"
	objs := s.createTestObjects(dir, subdir, bucketStr, c)

	acl := "public-read"
	args := []string{bucketStr, acl}
	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-rf", "--exclude", "*", "--include", "*"}

	showElapse, err := s.rawSetAclWithFilter(args, true, true, cmdline)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fts := []filterOptionType{{"--exclude", "*"}, {"--include", "*"}}
	matchedObjs := matchFiltersForStrs(objs, fts)

	for _, obj := range matchedObjs {
		objectStat := s.getStat(bucketName, obj, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
	}

	for _, obj := range objs {
		if !containsInStrsSlice(matchedObjs, obj) {
			objectStat := s.getStat(bucketName, obj, c)
			c.Assert(objectStat["ACL"], Equals, "default")
		}
	}

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Nagetive test
func (s *OssutilCommandSuite) TestSetObjectAclWithInvalidIncExc(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "test-setacl-invalid"
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	subdir := "dir1"
	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	// e.g., ossutil set-acl oss://tempb4 public-read -f --exclude '*.txt'
	acl := "public-read"
	args := []string{bucketStr, acl}

	cmdline := []string{"ossutil", "set-acl", bucketStr, acl, "-f", "--exclude", "*.txt"}
	showElapse, err := s.rawSetAclWithFilter(args, false, true, cmdline)
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	// e.g., ossutil set-acl oss://tempb4 private -f --include '*.txt' --exclude "*2*"
	cmdline = []string{"ossutil", "set-acl", bucketStr, acl, "-f", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawSetAclWithFilter(args, false, true, cmdline)
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "set-acl", bucketStr, acl, "-f", "--include", "/*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawSetAclWithFilter(args, false, true, cmdline)
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSetObjectAclWithVersion(c *C) {
	bucketName := bucketNamePrefix + "-set-alc-" + randLowStr(10)
	objectName := randStr(12)

	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// put object v1
	textBufferV1 := randStr(100)
	s.createFile(uploadFileName, textBufferV1, c)
	s.putObject(bucketName, objectName, uploadFileName, c)
	s.setObjectACL(bucketName, objectName, "public-read", false, true, c)
	objectStat := s.getStat(bucketName, objectName, c)
	versionIdV1 := objectStat["X-Oss-Version-Id"]
	objectACLV1 := objectStat[StatACL]
	c.Assert(len(versionIdV1) > 0, Equals, true)
	c.Assert(objectACLV1, Equals, "public-read")

	// put object v2
	textBufferV2 := randStr(200)
	s.createFile(uploadFileName, textBufferV2, c)
	s.putObject(bucketName, objectName, uploadFileName, c)
	s.setObjectACL(bucketName, objectName, "public-read-write", false, true, c)
	objectStat = s.getStat(bucketName, objectName, c)
	versionIdV2 := objectStat["X-Oss-Version-Id"]
	objectACLV2 := objectStat[StatACL]
	c.Assert(len(versionIdV2) > 0, Equals, true)
	c.Assert(objectACLV2, Equals, "public-read-write")
	c.Assert(strings.Contains(versionIdV1, versionIdV2), Equals, false)

	// begin set-acl with versionidV1
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionIdV1,
	}

	args := []string{CloudURLToString(bucketName, objectName), "private"}
	_, err := cm.RunCommand("set-acl", args, options)
	c.Assert(err, IsNil)

	//get without versionId
	objectStat = s.getStat(bucketName, objectName, c)
	objectACLV2 = objectStat[StatACL]
	c.Assert(len(versionIdV2) > 0, Equals, true)
	c.Assert(objectACLV2, Equals, "public-read-write")

	//stat with version id v1
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testResultFile

	args = []string{CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("stat", args, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout
	objectStat = s.getStatResults(c)
	objectACL := objectStat[StatACL]
	c.Assert(objectACL, Equals, "private")
}

func (s *OssutilCommandSuite) TestSetObjectAclWithInvalidVersionArgs(c *C) {
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

	args := []string{CloudURLToString(bucketName, objectName), "private"}
	_, err := cm.RunCommand("set-acl", args, options)
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

	args = []string{CloudURLToString(bucketName, objectName), "private"}
	_, err = cm.RunCommand("set-acl", args, options)
	c.Assert(strings.Contains(err.Error(), "--version-id only work on single object"), Equals, true)
}
