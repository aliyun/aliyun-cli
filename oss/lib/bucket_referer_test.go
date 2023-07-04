package lib

import (
	"encoding/xml"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketRefererPutSuccess(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
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

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check,get referer
	refererDownName := randLowStr(12) + "-referer-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	refererArgs = []string{CloudURLToString(bucketName, ""), refererDownName}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check referer
	_, err = os.Stat(refererDownName)
	c.Assert(err, IsNil)

	refererBody := s.readFile(refererDownName, c)
	referXml := oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)

	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, true)

	if referXml.RefererList[0] == refererDomainA {
		c.Assert(referXml.RefererList[1], Equals, refererDomainB)
	} else {
		c.Assert(referXml.RefererList[1], Equals, refererDomainA)
	}

	os.Remove(refererDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererDisableEmpty(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
	var str string
	strMethod := "put"
	disableEmptyRefer := true
	options := OptionMapType{
		"endpoint":            &str,
		"accessKeyID":         &str,
		"accessKeySecret":     &str,
		"stsToken":            &str,
		"configFile":          &configFile,
		"method":              &strMethod,
		"disableEmptyReferer": &disableEmptyRefer,
	}

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check,get referer
	refererDownName := randLowStr(12) + "-referer-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	refererArgs = []string{CloudURLToString(bucketName, ""), refererDownName}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check referer
	_, err = os.Stat(refererDownName)
	c.Assert(err, IsNil)

	refererBody := s.readFile(refererDownName, c)
	referXml := oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)

	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, false)

	if referXml.RefererList[0] == refererDomainA {
		c.Assert(referXml.RefererList[1], Equals, refererDomainB)
	} else {
		c.Assert(referXml.RefererList[1], Equals, refererDomainA)
	}

	os.Remove(refererDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererGetConfirm(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
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

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// check,get referer
	refererDownName := randLowStr(12) + "-referer-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	refererArgs = []string{CloudURLToString(bucketName, ""), refererDownName}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check referer
	_, err = os.Stat(refererDownName)
	c.Assert(err, IsNil)

	refererBody := s.readFile(refererDownName, c)
	referXml := oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)

	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, true)

	if referXml.RefererList[0] == refererDomainA {
		c.Assert(referXml.RefererList[1], Equals, refererDomainB)
	} else {
		c.Assert(referXml.RefererList[1], Equals, refererDomainA)
	}

	// get again
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	os.Remove(refererDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererGetToStdout(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
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

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// output to file
	fileName := "test-file-" + randLowStr(5)
	testResultFile, _ = os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	refererArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("referer", refererArgs, options)
	testResultFile.Close()
	os.Stdout = oldStdout

	refererBody := s.readFile(fileName, c)
	referXml := oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)

	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, true)

	if referXml.RefererList[0] == refererDomainA {
		c.Assert(referXml.RefererList[1], Equals, refererDomainB)
	} else {
		c.Assert(referXml.RefererList[1], Equals, refererDomainA)
	}

	c.Assert(err, IsNil)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererBucketError(c *C) {
	// referer command test
	var str string
	strMethod := "put"
	disableEmptyRefer := true
	options := OptionMapType{
		"endpoint":            &str,
		"accessKeyID":         &str,
		"accessKeySecret":     &str,
		"stsToken":            &str,
		"configFile":          &configFile,
		"method":              &strMethod,
		"disableEmptyReferer": &disableEmptyRefer,
	}

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{"oss://///", refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBucketRefererDeleteSuccess(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
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

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// check,get referer
	refererDownName := randLowStr(12) + "-referer-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	refererArgs = []string{CloudURLToString(bucketName, ""), refererDownName}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check referer
	_, err = os.Stat(refererDownName)
	c.Assert(err, IsNil)

	refererBody := s.readFile(refererDownName, c)
	referXml := oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)

	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, true)

	if referXml.RefererList[0] == refererDomainA {
		c.Assert(referXml.RefererList[1], Equals, refererDomainB)
	} else {
		c.Assert(referXml.RefererList[1], Equals, refererDomainA)
	}

	// delete referer
	strMethod = "delete"
	refererArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// get again
	os.Remove(refererDownName)
	strMethod = "get"
	refererArgs = []string{CloudURLToString(bucketName, ""), refererDownName}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, IsNil)

	// check referer
	_, err = os.Stat(refererDownName)
	c.Assert(err, IsNil)

	refererBody = s.readFile(refererDownName, c)
	referXml = oss.RefererXML{}
	err = xml.Unmarshal([]byte(refererBody), &referXml)
	c.Assert(err, IsNil)
	c.Assert(referXml.AllowEmptyReferer, Equals, true)
	c.Assert(len(referXml.RefererList), Equals, 0)

	os.Remove(refererDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererError(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// method is empty
	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod := "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)

	// args is empty
	strMethod = "put"
	refererArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)

}

func (s *OssutilCommandSuite) TestBucketRefererPutEmptyEndpoint(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// referer command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
	}

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererGetEmptyEndpoint(c *C) {
	// put referer
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// referer command test
	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
	}

	refererDomainA := "sina.com"
	refererDomainB := "baidu.com"
	refererArgs := []string{CloudURLToString(bucketName, ""), refererDomainA, refererDomainB}
	_, err := cm.RunCommand("referer", refererArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketRefererHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"referer"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
