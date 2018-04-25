package lib

import (
	"fmt"
	"net/url"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestStatErrArgc(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	command := "stat"
	args := []string{CloudURLToString(bucketName, ""), CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetBucketStat(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// get bucket stat
	bucketStat := s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatName], Equals, bucketName)
	c.Assert(bucketStat[StatLocation] != "", Equals, true)
	c.Assert(bucketStat[StatCreationDate] != "", Equals, true)
	c.Assert(bucketStat[StatExtranetEndpoint] != "", Equals, true)
	c.Assert(bucketStat[StatIntranetEndpoint] != "", Equals, true)
	c.Assert(bucketStat[StatACL], Equals, "private")
	c.Assert(bucketStat[StatOwner] != "", Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetStatNotExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	showElapse, err := s.rawGetStat(bucketName, "")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.putBucket(bucketName, c)

	showElapse, err = s.rawGetStat(bucketName, "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	object := "testobject_for_getstat_not_exist"
	showElapse, err = s.rawGetStat(bucketName, object)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	object = "testobject_exist"
	s.putObject(bucketName, object, uploadFileName, c)
	showElapse, err = s.rawGetStat(bucketName, object)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetStatRetryTimes(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	command := "stat"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	retryTimes := "1"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"retryTimes":      &retryTimes,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetStatErrSrc(c *C) {
	showElapse, err := s.rawGetStat("", "")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawGetStatWithArgs([]string{"../"})
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestStatIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "stat"
	str := ""
	args := []string{CloudURLToString(bucketName, "")}
	retryTimes := "1"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"retryTimes":      &retryTimes,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"retryTimes":      &retryTimes,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestStatURLEncoding(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "^M特殊字符 加上空格 test"
	s.putObject(bucketName, object, uploadFileName, c)

	urlObject := url.QueryEscape(object)

	_, err := s.rawGetStat(bucketName, urlObject)
	c.Assert(err, NotNil)

	command := "stat"
	args := []string{CloudURLToString(bucketName, urlObject)}
	str := ""
	retryTimes := "3"
	encodingType := URLEncodingType
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"retryTimes":      &retryTimes,
		"encodingType":    &encodingType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}
