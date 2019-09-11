package lib

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketTaggingPutSuccess(c *C) {
	// put tagging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// tagging command test
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

	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err := cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// check,get tag
	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(bucketTagCommand.tagResult.Tags), Equals, 1)
	c.Assert(bucketTagCommand.tagResult.Tags[0].Key, Equals, "key1")
	c.Assert(bucketTagCommand.tagResult.Tags[0].Value, Equals, "value1")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketTaggingDeleteSuccess(c *C) {
	// put tagging
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// tagging command test
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

	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err := cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// check,get tag
	strMethod = "get"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(bucketTagCommand.tagResult.Tags), Equals, 1)
	c.Assert(bucketTagCommand.tagResult.Tags[0].Key, Equals, "key1")
	c.Assert(bucketTagCommand.tagResult.Tags[0].Value, Equals, "value1")

	// delete bucket tagging
	strMethod = "delete"
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)

	// get bucket tagging again:error
	strMethod = "get"
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(bucketTagCommand.tagResult.Tags), Equals, 0)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketTaggingError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// bucket-tagging command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// method is empty
	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err := cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod := "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	// args is empty
	strMethod = "put"
	tagArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	//value is error
	tagInfo = "key1:value1"
	tagArgs = []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)

}

func (s *OssutilCommandSuite) TestBucketTaggingPutEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// bucket-tagging command test
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

	// oss client error
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	tagInfo := "key1#value1"
	tagArgs := []string{CloudURLToString(bucketName, ""), tagInfo}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketTaggingGetEmptyEndpoint(c *C) {
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

	// oss client error
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	tagArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-tagging", tagArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketTaggingHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"bucket-tagging"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
