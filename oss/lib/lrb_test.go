package lib

import (
	"os"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestListRegionBucketByEndpoint(c *C) {
	// create bucket in shenzhen
	bucketName_shenzhen := bucketNamePrefix + randLowStr(10)
	endpoint_shenzhen := "oss-cn-shenzhen.aliyuncs.com"
	command := "mb"
	args := []string{CloudURLToString(bucketName_shenzhen, "")}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &endpoint_shenzhen,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// create bucket in shanghai
	bucketName_shanghai := bucketNamePrefix + randLowStr(10)
	endpoint_shanghai := "oss-cn-shanghai.aliyuncs.com"
	command = "mb"
	args = []string{CloudURLToString(bucketName_shanghai, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shanghai,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// list shenzhen bucket
	command = "lrb"
	args = []string{}
	options = OptionMapType{
		"endpoint":        &endpoint_shenzhen,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// find shenzhen bucket ok
	bFindShenzhen := false
	bFindShanghai := false
	for _, result := range lrbCommand.listResult {
		for _, bucket := range result.Buckets {
			if bucket.Name == bucketName_shenzhen {
				bFindShenzhen = true
			}

			if bucket.Name == bucketName_shanghai {
				bFindShanghai = true
			}
		}
	}
	lrbCommand.listResult = []oss.ListBucketsResult{}

	c.Assert(bFindShenzhen, Equals, true)
	c.Assert(bFindShanghai, Equals, false)

	// rm shenzhen bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName_shenzhen, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shenzhen,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
		"allType":         &ok,
		"recursive":       &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// rm shanghai bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName_shanghai, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shanghai,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
		"allType":         &ok,
		"recursive":       &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestListRegionBucketByConfigFile(c *C) {
	// create bucket in shenzhen
	bucketName_shenzhen := bucketNamePrefix + randLowStr(10)
	endpoint_shenzhen := "oss-cn-shenzhen.aliyuncs.com"
	command := "mb"
	args := []string{CloudURLToString(bucketName_shenzhen, "")}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &endpoint_shenzhen,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// create bucket in shanghai
	bucketName_shanghai := bucketNamePrefix + randLowStr(10)
	endpoint_shanghai := "oss-cn-shanghai.aliyuncs.com"
	command = "mb"
	args = []string{CloudURLToString(bucketName_shanghai, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shanghai,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// create conf
	confName := "test_file_lrb_" + randLowStr(3)
	confText := "oss-cn-shenzhen.aliyuncs.com\n" + "oss-cn-shanghai.aliyuncs.com"
	s.createFile(confName, confText, c)

	// list shenzhen bucket
	command = "lrb"
	args = []string{confName}
	options = OptionMapType{
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// find shenzhen bucket ok
	bFindShenzhen := false
	bFindShanghai := false
	for _, result := range lrbCommand.listResult {
		for _, bucket := range result.Buckets {
			if bucket.Name == bucketName_shenzhen {
				bFindShenzhen = true
			}

			if bucket.Name == bucketName_shanghai {
				bFindShanghai = true
			}
		}
	}
	lrbCommand.listResult = []oss.ListBucketsResult{}

	c.Assert(bFindShenzhen, Equals, true)
	c.Assert(bFindShanghai, Equals, true)

	// rm shenzhen bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName_shenzhen, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shenzhen,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
		"allType":         &ok,
		"recursive":       &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// rm shanghai bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName_shanghai, "")}
	options = OptionMapType{
		"endpoint":        &endpoint_shanghai,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
		"allType":         &ok,
		"recursive":       &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	os.Remove(confName)
}

func (s *OssutilCommandSuite) TestListRegionBucketConfigFileNotexist(c *C) {

	// create conf
	confName := "test_file_lrb_" + randLowStr(3)

	// list shenzhen bucket
	command := "lrb"
	args := []string{confName}
	str := ""
	options := OptionMapType{
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestListRegionBucketHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"lrb"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestListRegionBucketByConfigFileInvalidEndpoint(c *C) {
	// create conf
	confName := "test_file_lrb_" + randLowStr(3)
	confText := "#oss-cn-shenzhen.aliyuncs.com\n" + "oss-cn-test.aliyuncs.com"
	s.createFile(confName, confText, c)

	// list shenzhen bucket
	str := ""
	command := "lrb"
	args := []string{confName}
	options := OptionMapType{
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	os.Remove(confName)
}
