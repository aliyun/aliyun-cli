package lib

import (
	"net/url"
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
	c.Assert(strings.Contains(err.Error(), "ErrorMessage=Request has expired"), Equals, true)
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
