package lib

import (
	"os"
	"strconv"
	"strings"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestObjectTaggingSingleOperation(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	fileName := "ossutil-test-file-" + randLowStr(5)
	textBuffer := randStr(100)
	s.createFile(fileName, textBuffer, c)

	object := "ossutil-test-object-" + randLowStr(5)
	s.putObject(bucketName, object, fileName, c)

	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// put tag
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, object), tagInfo}
	_, err := cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag
	resultfileName := "ossutil-test-result-" + randLowStr(5)
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody := s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, true)
	c.Assert(strings.Contains(catBody, "value1"), Equals, true)

	// delete tag
	strMethod = "delete"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag and check again
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody = s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, false)
	c.Assert(strings.Contains(catBody, "value1"), Equals, false)

	os.Remove(resultfileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestObjectTaggingBatchOperation(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	bucketStr := CloudURLToString(bucketName, "")

	dir := "ossutil-test-dir-" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	var str string
	strMethod := "put"
	recursive := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"recursive":       &recursive,
		"routines":        &routines,
	}

	// put tag
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag
	resultfileName := "ossutil-test-result-" + randLowStr(5)
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody := s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, true)
	c.Assert(strings.Contains(catBody, "value1"), Equals, true)

	for _, filename := range filenames {
		c.Assert(strings.Contains(string(catBody), filename), Equals, true)
	}

	// delete tag
	strMethod = "delete"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag and check again
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody = s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, false)
	c.Assert(strings.Contains(catBody, "value1"), Equals, false)
	for _, filename := range filenames {
		c.Assert(strings.Contains(string(catBody), filename), Equals, false)
	}

	os.RemoveAll(dir)
	os.Remove(resultfileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestObjectTaggingVersionId(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	fileName := "ossutil-test-file-" + randLowStr(5)
	textBuffer := randStr(100)
	s.createFile(fileName, textBuffer, c)

	object := "ossutil-test-object-" + randLowStr(5)
	s.putObject(bucketName, object, fileName, c)

	// put again
	s.putObject(bucketName, object, fileName, c)

	// get stat
	objectStat := s.getStat(bucketName, object, c)
	versionId := objectStat["X-Oss-Version-Id"]

	var str string
	recursive := true
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"versionId":       &versionId,
		"recursive":       &recursive,
	}

	// put tag
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, object), tagInfo}
	_, err := cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	delete(options, "recursive")
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag
	resultfileName := "ossutil-test-result-" + randLowStr(5)
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody := s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, true)
	c.Assert(strings.Contains(catBody, "value1"), Equals, true)

	// delete tag
	strMethod = "delete"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get tag and check again
	testResultFile, _ = os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Stdout = oldStdout
	testResultFile.Close()

	// check file content
	catBody = s.readFile(resultfileName, c)
	c.Assert(strings.Contains(catBody, "key1"), Equals, false)
	c.Assert(strings.Contains(catBody, "value1"), Equals, false)

	os.Remove(resultfileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestObjectTaggingError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	fileName := "ossutil-test-file-" + randLowStr(5)
	textBuffer := randStr(100)
	s.createFile(fileName, textBuffer, c)

	object := "ossutil-test-object-" + randLowStr(5)
	s.putObject(bucketName, object, fileName, c)

	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// put tag,duplicate key
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, object), tagInfo, tagInfo}
	_, err := cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod = "putt"
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// method is empty
	strMethod = ""
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	//object is emtpy
	strMethod = "put"
	tagArgs = []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// tag is lost
	strMethod = "put"
	tagArgs = []string{CloudURLToString(bucketName, object)}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// request is error
	requester := "aaaaaaaa"
	options["payer"] = &requester
	strMethod = "put"
	tagArgs = []string{CloudURLToString(bucketName, object), tagInfo}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// tag value is error
	tagInfo = "key1 value1"
	delete(options, "payer")
	strMethod = "put"
	tagArgs = []string{CloudURLToString(bucketName, object), tagInfo}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestObjectTaggingPayer(c *C) {
	bucketName := payerBucket
	objectName := randStr(10)

	fileName := "ossutil-test-file-" + randLowStr(5)
	textBuffer := randStr(100)
	s.createFile(fileName, textBuffer, c)

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, objectName)}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	var str string
	strMethod := "put"
	requester := "requester"
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"method":          &strMethod,
		"payer":           &requester,
	}
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, objectName), tagInfo}
	_, err = cm.RunCommand("object-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	os.Remove(fileName)
}

func (s *OssutilCommandSuite) TestObjectTaggingHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"object-tagging"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
