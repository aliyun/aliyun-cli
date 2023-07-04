package lib

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
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
	c.Assert(bucketStat[StatAccessMonitor] != "", Equals, true)
	c.Assert(bucketStat[StatLocation] != "", Equals, true)
	c.Assert(bucketStat[StatCreationDate] != "", Equals, true)
	c.Assert(bucketStat[StatExtranetEndpoint] != "", Equals, true)
	c.Assert(bucketStat[StatIntranetEndpoint] != "", Equals, true)
	c.Assert(bucketStat[StatACL], Equals, "private")
	c.Assert(bucketStat[StatOwner] != "", Equals, true)
	c.Assert(bucketStat[StatTransferAcceleration], Equals, "Disabled")
	c.Assert(bucketStat[StatCrossRegionReplication], Equals, "Disabled")

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

func (s *OssutilCommandSuite) TestStatVersioning(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	// create file and put object
	objectName := "ossutil-test-object" + randLowStr(5)
	data := randLowStr(200)
	testFileName := "ossutil-test-file" + randLowStr(5)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)

	// get stat
	objectStat := s.getStat(bucketName, objectName, c)
	versionId1 := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId1) > 0, Equals, true)
	etag1 := objectStat["Etag"]

	// overwirte object
	os.Remove(testFileName)
	data = randLowStr(100)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)

	// get stat again
	objectStat = s.getStat(bucketName, objectName, c)
	versionId2 := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId1) > 0, Equals, true)
	etag2 := objectStat["Etag"]

	// check
	c.Assert(versionId1 != versionId2, Equals, true)
	c.Assert(etag1 != etag2, Equals, true)

	// get stat with versionid
	testOutFileName := "ossutil-test-file" + randLowStr(5)
	testResultFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldOut := os.Stdout
	os.Stdout = testResultFile

	command := "stat"
	args := []string{CloudURLToString(bucketName, objectName)}
	str := ""
	retryTimes := "1"
	encodingType := URLEncodingType
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"retryTimes":      &retryTimes,
		"encodingType":    &encodingType,
		"versionId":       &versionId1,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(string(fileBody), versionId1), Equals, true)
	c.Assert(strings.Contains(string(fileBody), etag1), Equals, true)

	os.Stdout = oldOut

	os.Remove(testOutFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestStatObjectWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// stat with payer
	command := "stat"
	args = []string{CloudURLToString(bucketName, uploadFileName)}
	str := ""
	requester := "requester"
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}

	resultfileName := "ossutil-test-result-" + randLowStr(5)
	testResultFile, _ := os.OpenFile(resultfileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testResultFile

	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout

	statBody := s.readFile(resultfileName, c)
	c.Assert(strings.Contains(statBody, "X-Oss-Hash-Crc64ecma"), Equals, true)
	os.Remove(resultfileName)
}
