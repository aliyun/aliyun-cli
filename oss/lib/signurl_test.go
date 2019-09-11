package lib

import (
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

/* Case: normal access
 * Put object and sign url, download by signed url, compare content
 */
func (s *OssutilCommandSuite) TestNormalSignurl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := "签名url"
	s.createFile(uploadFileName, data, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	cmdline := CloudURLToString(bucketName, object)
	str := s.signURL(cmdline, URLEncodingType, DefaultTimeout, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)
	c.Assert(strings.Contains(str, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
}

/* Case: object name contains speical char
 * put object, sign url, download by signed url, compare content
 */
func (s *OssutilCommandSuite) TestObjectNameWithSpeicalChar(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "test a+b"
	objurl := url.QueryEscape(object)
	c.Assert(object != objurl, Equals, true)

	data := randStr(100)
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, object, uploadFileName, c)

	cmdline := CloudURLToString(bucketName, objurl)
	str := s.signURL(cmdline, URLEncodingType, 300, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
}

/* Case: object not exist
 * sign url, download by signed url
 */
func (s *OssutilCommandSuite) TestObjectNotExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "中文"
	objurl := url.QueryEscape(object)
	c.Assert(object != objurl, Equals, true)

	cmdline := CloudURLToString(bucketName, objurl)
	str := s.signURL(cmdline, "url", DefaultTimeout, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(strings.Contains(err.Error(), "ErrorCode=NoSuchKey"), Equals, true)
}

/* Case: object name with Chinese
 * Put object and sign url, download by signed url, compare content
 */
func (s *OssutilCommandSuite) TestObjectNameWithChinese(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "中文"
	objurl := url.QueryEscape(object)
	c.Assert(object != objurl, Equals, true)

	data := randStr(100)
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, object, uploadFileName, c)

	// sign url
	cmdline := CloudURLToString(bucketName, objurl)
	str := s.signURL(cmdline, URLEncodingType, DefaultTimeout, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
}

/* Case: object name with unreadable code
 * Put object and sign url, download by signed url, compare content
 */
func (s *OssutilCommandSuite) TestObjectNameUnreadable(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "寰堝睂"
	objurl := url.QueryEscape(object)
	c.Assert(object != objurl, Equals, true)

	data := "寰堝睂xyz中文"
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, object, uploadFileName, c)

	cmdline := CloudURLToString(bucketName, object)
	str := s.signURL(cmdline, URLEncodingType, DefaultTimeout, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)
}

/* Case: expire
 * sign url with timeout, sleep, download by signed url
 */
func (s *OssutilCommandSuite) TestExpire(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := randStr(200)
	s.putObject(bucketName, object, uploadFileName, c)

	cmdline := CloudURLToString(bucketName, object)
	str := s.signURL(cmdline, URLEncodingType, 3, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)
	c.Assert(strings.Contains(str, object), Equals, true)
	time.Sleep(4 * time.Second)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(strings.Contains(err.Error(), "Request has expired"), Equals, true)
}

func (s *OssutilCommandSuite) TestSignurlErr(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := "签名url"
	s.createFile(uploadFileName, data, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	cmdline := CloudURLToString("", object)
	err := s.initSignURL(cmdline, URLEncodingType, DefaultTimeout)
	c.Assert(err, IsNil)
	err = signURLCommand.RunCommand()
	c.Assert(err, NotNil)

	cmdline = CloudURLToString(bucketName, "")
	err = s.initSignURL(cmdline, URLEncodingType, DefaultTimeout)
	c.Assert(err, IsNil)
	err = signURLCommand.RunCommand()
	c.Assert(err, NotNil)

	// invalid timeout
	cmdline = CloudURLToString(bucketName, object)
	err = s.initSignURL(cmdline, URLEncodingType, -1)
	c.Assert(err, IsNil)
	err = signURLCommand.RunCommand()
	c.Assert(strings.Contains(err.Error(), "invalid expires"), Equals, true)
}

/* Case: invalid encoding type
 * put object and sign url with invalid encoding type, download by signed url
 */
func (s *OssutilCommandSuite) TestInvalidOptions(c *C) {
	str := "-1"
	options := OptionMapType{OptionTimeout: &str}
	err := checkOption(options)
	c.Assert(strings.Contains(err.Error(), "invalid option value of timeout"), Equals, true)

	str = "xxxx"
	options = OptionMapType{OptionEncodingType: &str}
	err = checkOption(options)
	c.Assert(strings.Contains(err.Error(), "invalid option value of encodingType"), Equals, true)
}

/* Case: version
 * put object and sign url with versionId, download by signed url
 */
func (s *OssutilCommandSuite) TestSignurlWithVersion(c *C) {
	bucketName := bucketNamePrefix + "-sign-" + randLowStr(10)
	objectName := randStr(12)

	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// put object v1
	textBufferV1 := randStr(100)
	s.createFile(uploadFileName, textBufferV1, c)
	s.putObject(bucketName, objectName, uploadFileName, c)
	bucketStat := s.getStat(bucketName, objectName, c)
	versionIdV1 := bucketStat["X-Oss-Version-Id"]
	c.Assert(len(versionIdV1) > 0, Equals, true)

	// put object v2
	textBufferV2 := randStr(200)
	s.createFile(uploadFileName, textBufferV2, c)
	s.putObject(bucketName, objectName, uploadFileName, c)
	bucketStat = s.getStat(bucketName, objectName, c)
	versionIdV2 := bucketStat["X-Oss-Version-Id"]
	c.Assert(len(versionIdV2) > 0, Equals, true)
	c.Assert(strings.Contains(versionIdV1, versionIdV2), Equals, false)

	// get object without versionid
	cmdline := CloudURLToString(bucketName, objectName)
	str := s.signURL(cmdline, URLEncodingType, DefaultTimeout, c)
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)
	c.Assert(strings.Contains(str, objectName), Equals, true)
	c.Assert(strings.Contains(str, "versionId"), Equals, false)

	// get object with url
	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, textBufferV2)

	// get object with versionid V1
	var str1 string
	t := strconv.FormatInt(DefaultTimeout, 10)
	args := []string{CloudURLToString(bucketName, objectName)}
	options := OptionMapType{
		"endpoint":        &str1,
		"accessKeyID":     &str1,
		"accessKeySecret": &str1,
		"stsToken":        &str1,
		"configFile":      &configFile,
		"timeout":         &t,
		"versionId":       &versionIdV1,
	}

	out := os.Stdout
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testResultFile
	_, err = cm.RunCommand("sign", args, options)
	testResultFile.Close()
	os.Stdout = out
	c.Assert(err, IsNil)

	body := s.getFileResult(resultPath, c)
	str = body[0]
	c.Assert(strings.Contains(str, "Expires"), Equals, true)
	c.Assert(strings.Contains(str, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(str, "Signature"), Equals, true)
	c.Assert(strings.Contains(str, bucketName), Equals, true)
	c.Assert(strings.Contains(str, objectName), Equals, true)
	c.Assert(strings.Contains(str, "versionId"), Equals, true)

	// get object with url
	err = bucket.GetObjectToFileWithURL(str, downloadFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, textBufferV1)
}

func (s *OssutilCommandSuite) TestTraficLimitSignUrl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := randLowStr(1024)
	uploadFileName := "ossutil-test-file-" + randLowStr(5)
	s.createFile(uploadFileName, data, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	command := "sign"
	str := ""
	trafficLimit := strconv.FormatInt(1024*1024*8, 10)
	timeOut := strconv.FormatInt(60, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"trafficLimit":    &trafficLimit,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "x-oss-traffic-limit"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, data)

	os.Remove(uploadFileName)
	os.Remove(downFileName)
	s.removeBucket(bucketName, true, c)
}
