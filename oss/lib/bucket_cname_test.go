package lib

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketCnamePutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

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

	cnameArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameGetSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	cnameArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameNotConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// get cname
	cnameDownName := "test-ossutil-cname-" + randLowStr(5)
	strMethod := "get"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// create file cnameDownName
	s.createFile(cnameDownName, "", c)

	cnameArgs := []string{CloudURLToString(bucketName, ""), cnameDownName}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)

	xmlBody, err := ioutil.ReadFile(cnameDownName)
	c.Assert(err, IsNil)
	c.Assert(len(xmlBody) == 0, Equals, true)

	os.Remove(cnameDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"bucket-cname"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
