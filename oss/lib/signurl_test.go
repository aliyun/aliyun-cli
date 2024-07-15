package lib

import (
	"fmt"
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

func (s *OssutilCommandSuite) TestSignUrlTraficLimit(c *C) {
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

func (s *OssutilCommandSuite) TestSignUrlWithDisableEncodePathUrl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := randLowStr(1024)
	uploadFileName := "ossutil-test-file-" + randLowStr(5)
	s.createFile(uploadFileName, data, c)

	object := randStr(5) + "/" + randStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	command := "sign"
	str := ""
	disableEncodePath := true
	timeOut := strconv.FormatInt(60, 10)
	options := OptionMapType{
		"endpoint":           &str,
		"accessKeyID":        &str,
		"accessKeySecret":    &str,
		"stsToken":           &str,
		"configFile":         &configFile,
		"disableEncodeSlash": &disableEncodePath,
		"timeout":            &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
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

func (s *OssutilCommandSuite) TestSignUrlWithRequestPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	data := randLowStr(1024)
	uploadFileName := "ossutil-test-file-" + randLowStr(5)
	s.createFile(uploadFileName, data, c)

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//signurl with requester payer
	command := "sign"
	str := ""
	payer := "requester"
	timeOut := strconv.FormatInt(60, 10)
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &payer,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, uploadFileName)
	args = []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

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
}

func (s *OssutilCommandSuite) TestSignHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"sign"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestSignWithInputPassword(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	str := ""
	bPassword := accessKeySecret
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"password":        &bPassword,
	}
	args := []string{CloudURLToString(bucketName, object)}
	_, err := cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithModeAk(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)
	command := "sign"
	str := ""
	timeOut := strconv.FormatInt(60, 10)
	mode := "AK"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"mode":            &mode,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)

}

func (s *OssutilCommandSuite) TestSignWithModeEcsRamRole(c *C) {
	accessKeyID = ""
	accessKeySecret = ""

	svr := startHttpServer(StsHttpHandlerOk)
	time.Sleep(time.Duration(1) * time.Second)

	//set endpoint emtpy
	cfile := randStr(10)
	ecsAk := "http://127.0.0.1:32915/latest/meta-data/Ram/security-credentials/EcsRamRoleTesting"
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "endpoint= " + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	s.createFile(cfile, configStr, c)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)
	command := "sign"
	str := ""
	timeOut := strconv.FormatInt(600, 10)
	mode := "EcsRamRole"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &cfile,
		"mode":            &mode,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)

	svr.Close()
}

func (s *OssutilCommandSuite) TestSignWithModeStsToken(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	client := NewClient(stsAccessID, stsAccessKeySecret, stsARN, "sts_test")

	resp, err := client.AssumeRole(3600, "https://sts.cn-hangzhou.aliyuncs.com")
	c.Assert(err, IsNil)

	temAccessID := resp.Credentials.AccessKeyId
	temAccessKeySecret := resp.Credentials.AccessKeySecret
	temSTSToken := resp.Credentials.SecurityToken

	c.Assert(temAccessID, Not(Equals), "")
	c.Assert(temAccessKeySecret, Not(Equals), "")
	c.Assert(temSTSToken, Not(Equals), "")

	str := ""
	mode := "StsToken"
	timeOut := strconv.FormatInt(3600, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &temAccessID,
		"accessKeySecret": &temAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"stsToken":        &temSTSToken,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err = cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithModeRamRoleArn(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	str := ""
	mode := "RamRoleArn"
	timeOut := strconv.FormatInt(3600, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"ramRoleArn":      &stsARN,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithModeRamRoleArnTokenTimeOut(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	str := ""
	mode := "RamRoleArn"
	timeOut := strconv.FormatInt(3600, 10)
	tokenTimeout := 20
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"ramRoleArn":      &stsARN,
		"timeout":         &timeOut,
		"tokenTimeout":    &tokenTimeout,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithModeRamRoleArnStsRegion(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	str := ""
	mode := "RamRoleArn"
	timeOut := strconv.FormatInt(3600, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"ramRoleArn":      &stsARN,
		"stsRegion":       &stsRegion,
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithRoleSessionName(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)

	str := ""
	mode := "RamRoleArn"
	timeOut := strconv.FormatInt(3600, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"ramRoleArn":      &stsARN,
		"roleSessionName": "demo-role",
		"timeout":         &timeOut,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand("sign", args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignWithUserAgent(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	s.putObject(bucketName, object, fileName, c)
	command := "sign"
	str := ""
	timeOut := strconv.FormatInt(3600, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"timeout":         &timeOut,
		"userAgent":       &str,
	}

	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(strings.Contains(signURLCommand.signUrl, "Expires"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "OSSAccessKeyId"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, "Signature"), Equals, true)
	c.Assert(strings.Contains(signURLCommand.signUrl, object), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5)
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str, Equals, objectContext)
	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSignUrlWithQueryProcess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := randLowStr(1024)
	uploadFileName := "ossutil-test-file-" + randLowStr(5)
	s.createFile(uploadFileName, data, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	command := "sign"
	query := []string{
		"x-oss-traffic-limit:800000",
	}
	timeOut := strconv.FormatInt(600, 10)
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"queryParam":      &query,
		"timeout":         &timeOut,
	}
	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	signUrl, err := url.QueryUnescape(signURLCommand.signUrl)
	testLogger.Print(signUrl)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(signUrl, "x-oss-traffic-limit=800000"), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5) + ".txt"
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str := s.readFile(downFileName, c)
	c.Assert(str != "", Equals, true)
	os.Remove(downFileName)

	// test many query
	query = []string{
		"x-oss-traffic-limit:2160000",
		"response-content-type:text/txt",
	}
	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"queryParam":      &query,
		"timeout":         &timeOut,
	}
	srcUrl = CloudURLToString(bucketName, object)
	args = []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	signUrl, err = url.QueryUnescape(signURLCommand.signUrl)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(signUrl, "x-oss-traffic-limit=2160000"), Equals, true)
	c.Assert(strings.Contains(signUrl, "response-content-type=text/txt"), Equals, true)
	downFileName = "ossutil-test-file" + randStr(5) + ".txt"
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str = s.readFile(downFileName, c)
	c.Assert(str != "", Equals, true)
	os.Remove(downFileName)
}

func (s *OssutilCommandSuite) TestSignUrlWithSignV4(c *C) {
	var err error
	if region == "" {
		err = fmt.Errorf("region can not be empty")
	}
	c.Assert(err, IsNil)
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := randLowStr(1024)
	uploadFileName := "ossutil-test-file-" + randLowStr(5)
	s.createFile(uploadFileName, data, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	command := "sign"
	timeOut := strconv.FormatInt(600, 10)
	authV4 := "v4"
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"timeout":         &timeOut,
		OptionSignVersion: &authV4,
	}
	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "please enter the region"), Equals, true)

	options[OptionRegion] = &region
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	signUrl, err := url.QueryUnescape(signURLCommand.signUrl)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(signUrl, object), Equals, true)
	c.Assert(strings.Contains(signUrl, "x-oss-credential="), Equals, true)
	c.Assert(strings.Contains(signUrl, "x-oss-date="), Equals, true)
	c.Assert(strings.Contains(signUrl, "x-oss-signature="), Equals, true)
	c.Assert(strings.Contains(signUrl, "x-oss-expires=600"), Equals, true)
	c.Assert(strings.Contains(signUrl, "x-oss-signature-version=OSS4-HMAC-SHA256"), Equals, true)

	bucket, err := signURLCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	// get object with url
	downFileName := "ossutil-test-file" + randStr(5) + ".txt"
	err = bucket.GetObjectToFileWithURL(signURLCommand.signUrl, downFileName)
	c.Assert(err, IsNil)
	str := s.readFile(downFileName, c)
	c.Assert(str != "", Equals, true)
	os.Remove(downFileName)
}
