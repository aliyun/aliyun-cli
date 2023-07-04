package lib

import (
	"encoding/xml"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketLogPutSuccess(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// cors command test
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

	logArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, "")}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// check,get logging
	logDownName := randLowStr(12) + "-log-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	logArgs = []string{CloudURLToString(bucketName, ""), logDownName}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check logging
	_, err = os.Stat(logDownName)
	c.Assert(err, IsNil)

	logBody := s.readFile(logDownName, c)

	logXml := oss.LoggingXML{}
	err = xml.Unmarshal([]byte(logBody), &logXml)
	c.Assert(err, IsNil)
	c.Assert(logXml.LoggingEnabled.TargetBucket, Equals, bucketLog)

	os.Remove(logDownName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogDeleteSuccess(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// cors command test
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

	prefixName := "test"
	logArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, prefixName)}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check,get logging
	logDownName := randLowStr(12) + "-log-down"
	strMethod = "get"
	logArgs = []string{CloudURLToString(bucketName, ""), logDownName}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check logging
	_, err = os.Stat(logDownName)
	c.Assert(err, IsNil)

	logBody := s.readFile(logDownName, c)
	logXml := oss.LoggingXML{}
	err = xml.Unmarshal([]byte(logBody), &logXml)
	c.Assert(err, IsNil)
	c.Assert(logXml.LoggingEnabled.TargetBucket, Equals, bucketLog)
	c.Assert(logXml.LoggingEnabled.TargetPrefix, Equals, prefixName)

	// then delete logging
	strMethod = "delete"
	logArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check again
	strMethod = "get"
	os.Remove(logDownName)
	logArgs = []string{CloudURLToString(bucketName, ""), logDownName}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check logging
	_, err = os.Stat(logDownName)
	c.Assert(err, IsNil)

	logBody = s.readFile(logDownName, c)

	logXml = oss.LoggingXML{}
	err = xml.Unmarshal([]byte(logBody), &logXml)
	c.Assert(err, IsNil)
	c.Assert(logXml.LoggingEnabled.TargetBucket, Equals, "")
	c.Assert(logXml.LoggingEnabled.TargetPrefix, Equals, "")

	os.Remove(logDownName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogGetToStdout(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// cors command test
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

	prefixName := "test"
	logArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, prefixName)}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check,get logging
	strMethod = "get"
	logArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogGetConfirm(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// cors command test
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

	logArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, "")}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check,get logging
	logDownName := randLowStr(12) + "-log-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	logArgs = []string{CloudURLToString(bucketName, ""), logDownName}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	// check logging
	_, err = os.Stat(logDownName)
	c.Assert(err, IsNil)

	logBody := s.readFile(logDownName, c)

	logXml := oss.LoggingXML{}
	err = xml.Unmarshal([]byte(logBody), &logXml)
	c.Assert(err, IsNil)
	c.Assert(logXml.LoggingEnabled.TargetBucket, Equals, bucketLog)

	// get again,test confirm
	logArgs = []string{CloudURLToString(bucketName, ""), logDownName}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, IsNil)

	os.Remove(logDownName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogPutError(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// method is empty
	var str string
	strMethod := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	logArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, "")}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// method is invalid
	strMethod = "head"
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// cloud url is error
	strMethod = "put"
	logArgs = []string{"http:///", CloudURLToString(bucketLog, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// put log missing parameter
	logArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// dest cloud url is error
	logArgs = []string{CloudURLToString(bucketName, ""), "http:////"}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// src bucket is not exist
	logArgs = []string{CloudURLToString(bucketName+"test", ""), CloudURLToString(bucketLog, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	//accessKeySecret error
	strKey := "aaa"
	options["accessKeySecret"] = &strKey
	logArgs = []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	// delete accessKeySecret
	delete(bucketLogCommand.command.options, OptionAccessKeySecret)
	logArgs = []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketLog, "")}
	_, err = cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogDeleteError(c *C) {
	// put logging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	bucketLog := bucketName + "-log"
	s.putBucket(bucketLog, c)

	// accessKeySecret error
	var str string
	strMethod := "delete"
	strKey := "aaa"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &strKey,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}
	logArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("logging", logArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketLog, true, c)
}

func (s *OssutilCommandSuite) TestBucketLogHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"logging"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
