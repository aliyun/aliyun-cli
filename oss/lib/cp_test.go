package lib

import (
	"fmt"
	"hash/fnv"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestCPObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	// dest bucket is not exist
	destBucket := bucketName + "-dest"

	// put object
	s.createFile(uploadFileName, content, c)
	object := randStr(12)
	s.putObject(bucketName, object, uploadFileName, c)
	// get object
	s.getObject(bucketName, object, downloadFileName, c)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, content)

	// modify the content of the file
	data := "欢迎使用ossutil"
	s.createFile(uploadFileName, data, c)
	// overwrite the object
	s.putObject(bucketName, object, uploadFileName, c)
	// get object
	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	// get object to the current directory
	s.getObject(bucketName, object, ".", c)
	str = s.readFile(object, c)
	c.Assert(str, Equals, data)
	os.Remove(object)

	// put object without specify the dest object
	data1 := "put object without specify the dest object"
	s.createFile(uploadFileName, data1, c)
	s.putObject(bucketName, "", uploadFileName, c)
	s.getObject(bucketName, uploadFileName, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data1)

	// get object to file in directory that does not exist
	notexistdir := "NOTEXISTDIR_" + randStr(5)
	s.getObject(bucketName, object, notexistdir+string(os.PathSeparator)+downloadFileName, c)
	str = s.readFile(notexistdir+string(os.PathSeparator)+downloadFileName, c)
	c.Assert(str, Equals, data)
	os.RemoveAll(notexistdir)

	// copy object
	destObject := "TestCPObject_destObject"
	s.copyObject(bucketName, object, bucketName, destObject, c)
	objectStat := s.getStat(bucketName, destObject, c)
	c.Assert(objectStat[StatACL], Equals, "default")
	// get object
	filePath := downloadFileName + randStr(5)
	s.getObject(bucketName, destObject, filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data)
	os.Remove(filePath)

	// put object to non-existent bucket
	showElapse, err := s.rawCP(uploadFileName, CloudURLToString(destBucket, object), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// get object from non-existent bucket
	showElapse, err = s.rawCP(CloudURLToString(destBucket, object), downloadFileName, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// get object from non-existent object
	showElapse, err = s.rawCP(CloudURLToString(bucketName, "notexistobject"), downloadFileName, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// copy object to non-existent bucket
	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, destObject), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// copy object
	s.putBucket(destBucket, c)
	s.copyObject(bucketName, object, destBucket, destObject, c)
	s.getObject(destBucket, destObject, filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data)
	os.Remove(filePath)

	// copy single object in directory, verify the path of dest object
	srcObject := "a/b/c/d/e"
	s.putObject(bucketName, srcObject, uploadFileName, c)
	s.copyObject(bucketName, srcObject, destBucket, "", c)
	s.getObject(destBucket, "e", filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data1)
	os.Remove(filePath)

	s.copyObject(bucketName, srcObject, destBucket, "a/", c)
	s.getObject(destBucket, "a/e", filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data1)
	os.Remove(filePath)

	s.copyObject(bucketName, srcObject, destBucket, "a", c)
	s.getObject(destBucket, "a", filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data1)
	os.Remove(filePath)

	// copy without specify dest object
	s.copyObject(bucketName, uploadFileName, destBucket, "", c)
	s.getObject(destBucket, uploadFileName, filePath, c)
	str = s.readFile(filePath, c)
	c.Assert(str, Equals, data1)
	os.Remove(filePath)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestErrorCP(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// error src_url
	showElapse, err := s.rawCP(uploadFileName, CloudURLToString("", ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(uploadFileName, CloudURLToString("", bucketName), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString("", bucketName), downloadFileName, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString("", ""), downloadFileName, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(uploadFileName, "a", true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// miss argc
	showElapse, err = s.rawCP(CloudURLToString("", bucketName), "", true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// copy self
	object := "testobject"
	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(bucketName, object), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(bucketName, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), CloudURLToString(bucketName, object), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// err checkpoint dir, conflict with config file
	showElapse, err = s.rawCP(uploadFileName, CloudURLToString(bucketName, object), false, true, true, DefaultBigFileThreshold, configFile)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetInvalidSrcURL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	s.createFile(uploadFileName, content, c)
	dirObject := randLowStr(5) + "/"
	object := dirObject + randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	//get object, the src URL is a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, dirObject), downloadDir, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetSrcURLMissDelimiter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	s.createFile(uploadFileName, content, c)
	object := randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	//get object with --recusive, the src URL is not a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, object), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPutInvalidSrcURL(c *C) {
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	dir := randLowStr(5) + "/"
	fileName := randLowStr(5)
	filePath := dir + fileName
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)
	s.createFile(filePath, content, c)

	//put object, the src URL is a directory
	showElapse, err := s.rawCP(dir, CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(destBucket, true, c)
	os.RemoveAll(dir)
}

func (s *OssutilCommandSuite) TestPutSrcURLMissDelimiter(c *C) {
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)
	s.createFile(uploadFileName, content, c)

	//put object with --recursive, the src URL is not a directory, it is invalid
	showElapse, err := s.rawCP(uploadFileName, CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestCopyInvalidSrcURL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	s.createFile(uploadFileName, content, c)
	dirObject := randLowStr(5) + "/"
	object := dirObject + randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	//copy object, the src URL is a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, dirObject), CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestCopySrcURLMissDelimiter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	s.createFile(uploadFileName, content, c)
	object := randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	//copy object with --recusive, the src URL is not a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestGetDestURLMissDelimiter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	s.createFile(uploadFileName, content, c)
	dirObject := randLowStr(5) + "/"
	object := dirObject + randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	downloadPath := downloadDir + "/" + randLowStr(10)

	//get object with --recusive, the dest URL is not a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, dirObject), downloadPath, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPutDestURLMissDelimiter(c *C) {
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	dirPath := randLowStr(5) + "/"
	filePath := dirPath + randLowStr(5)
	err := os.MkdirAll(dirPath, 0755)
	c.Assert(err, IsNil)
	s.createFile(filePath, content, c)

	destObject := randLowStr(5)

	showElapse, err := s.rawCP(dirPath, CloudURLToString(destBucket, destObject), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(destBucket, true, c)
	os.Remove(filePath)
}

func (s *OssutilCommandSuite) TestCopyDestURLMissDelimiter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	s.createFile(uploadFileName, content, c)
	dirObject := randLowStr(5) + "/"
	object := dirObject + randLowStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	destObject := randLowStr(5)

	//get object with --recusive, the dest URL is not a directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, dirObject), CloudURLToString(destBucket, destObject), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetMultiLevelSrcURL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	s.createFile(uploadFileName, content, c)
	suffix := randLowStr(5)
	multiLevelDir := randLowStr(5) + "/" + suffix
	object := randLowStr(5)
	multiLevelObj := multiLevelDir + "/" + object
	s.putObject(bucketName, multiLevelObj, uploadFileName, c)

	//get object, the src object is in multi-level directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, multiLevelObj), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(downloadDir + "/" + object)
	c.Assert(err, IsNil)
	err = os.Remove(downloadDir + "/" + object)
	c.Assert(err, IsNil)

	object = randLowStr(10)
	multiLevelObj = multiLevelDir + "/" + object
	s.putObject(bucketName, multiLevelObj, uploadFileName, c)

	//get object with --recursive, the src dir object is not end with "/"
	showElapse, err = s.rawCP(CloudURLToString(bucketName, multiLevelDir), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(downloadDir + "/" + suffix + "/" + object)
	c.Assert(err, IsNil)
	err = os.Remove(downloadDir + "/" + suffix + "/" + object)
	c.Assert(err, IsNil)

	//get object with --recursive, the src dir object is multi-level directory
	showElapse, err = s.rawCP(CloudURLToString(bucketName, multiLevelDir+"/"), downloadDir, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(downloadDir + "/" + object)
	c.Assert(err, IsNil)
	err = os.Remove(downloadDir + "/" + object)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPutMultiLevelSrcURL(c *C) {
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	multiLevelDir := randLowStr(5) + "/" + randLowStr(5)
	fileName := randLowStr(5)
	filePath := multiLevelDir + "/" + fileName
	err := os.MkdirAll(multiLevelDir, 0755)
	c.Assert(err, IsNil)
	s.createFile(filePath, content, c)

	//put object, the src file is in multi-level directory
	showElapse, err := s.rawCP(filePath, CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	s.getStat(destBucket, fileName, c)

	fileName = randLowStr(10)
	filePath = multiLevelDir + "/" + fileName
	s.createFile(filePath, content, c)

	//put object with --recursive, the src dir is multi-level directory
	showElapse, err = s.rawCP(multiLevelDir, CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	s.getStat(destBucket, fileName, c)

	s.removeBucket(destBucket, true, c)
	os.Remove(filePath)
}

func (s *OssutilCommandSuite) TestCopyMultiLevelSrcURL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	destBucket := bucketName + "-dest"
	s.putBucket(destBucket, c)

	fileName := randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)

	suffix := randLowStr(5)
	multiLevelDir := randLowStr(5) + "/" + suffix
	object := randLowStr(5)
	multiLevelObj := multiLevelDir + "/" + object
	s.putObject(bucketName, multiLevelObj, fileName, c)

	//copy object, the src object is in multi-level directory
	showElapse, err := s.rawCP(CloudURLToString(bucketName, multiLevelObj), CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	s.getStat(destBucket, object, c)

	object = randLowStr(10)
	multiLevelObj = multiLevelDir + "/" + object
	s.putObject(bucketName, multiLevelObj, fileName, c)

	//copy object with --recursive, the src dir is multi-level directory
	showElapse, err = s.rawCP(CloudURLToString(bucketName, multiLevelDir), CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	s.getStat(destBucket, suffix+"/"+object, c)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, multiLevelDir+"/"), CloudURLToString(destBucket, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	s.getStat(destBucket, object, c)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestPutWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//put object resumble, with --payer=requester
	args = []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err = s.rawCPWithPayer(args, false, true, false, 1, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestPutWithPayerInvalidPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket
	invalidPayer := randStr(10)

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, invalidPayer)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestGetWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket
	destObject := randStr(10)

	//at first, put object to bucket for test
	args := []string{uploadFileName, CloudURLToString(bucketName, destObject)}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//get object, with --payer=requester
	args = []string{CloudURLToString(bucketName, destObject), downloadFileName}
	showElapse, err = s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//get object resumble, with --payer=requester
	showElapse, err = s.rawCPWithPayer(args, false, true, false, 1, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestCopyWithPayer(c *C) {
	//copy from payer bucket
	srcBucket := payerBucket
	destBucket := bucketNamePrefix + randLowStr(5)
	s.putBucketWithEndPoint(destBucket, payerBucketEndPoint, c)
	s.createFile(uploadFileName, content, c)

	//at first, put object to bucket for test
	args := []string{uploadFileName, CloudURLToString(srcBucket, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	args = []string{CloudURLToString(srcBucket, uploadFileName), CloudURLToString(destBucket, "")}
	showElapse, err = s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	showElapse, err = s.rawCPWithPayer(args, false, true, false, 1, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	//copy to payer bucket
	srcBucket = destBucket
	destBucket = payerBucket
	args = []string{CloudURLToString(srcBucket, uploadFileName), CloudURLToString(destBucket, "")}
	showElapse, err = s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	showElapse, err = s.rawCPWithPayer(args, false, true, false, 1, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestUploadErrSrc(c *C) {
	srcBucket := bucketNamePrefix + randLowStr(10)
	destBucket := bucketNamePrefix + randLowStr(10)
	command := "cp"
	args := []string{uploadFileName, CloudURLToString(srcBucket, ""), CloudURLToString(destBucket, "")}
	str := ""
	ok := true
	cpDir := CheckpointDir
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"force":            &ok,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestBatchCPObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// create local dir
	dir := randStr(10)
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	// upload empty dir miss recursive
	showElapse, err := s.rawCP(dir, CloudURLToString(bucketName, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// upload empty dir
	showElapse, err = s.rawCP(dir, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// head object
	showElapse, err = s.rawGetStat(bucketName, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawGetStat(bucketName, dir+"/")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	os.RemoveAll(dir)

	// create dir in dir
	dir = randStr(10)
	subdir := randStr(10)
	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	// upload dir
	showElapse, err = s.rawCP(dir, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// remove object
	s.removeObjects(bucketName, subdir+"/", false, true, c)

	// create file in dir
	num := 3
	filePaths := []string{subdir + "/"}
	for i := 0; i < num; i++ {
		filePath := fmt.Sprintf("TestBatchCPObject_%d", i)
		s.createFile(dir+"/"+filePath, fmt.Sprintf("测试文件：%d内容", i), c)
		filePaths = append(filePaths, filePath)
	}

	// upload files
	showElapse, err = s.rawCP(dir, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// get files
	downDir := "下载目录"
	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), downDir, true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	for _, filePath := range filePaths {
		_, err := os.Stat(downDir + "/" + filePath)
		c.Assert(err, IsNil)
	}

	_, err = os.Stat(downDir)
	c.Assert(err, IsNil)

	// get to exist files
	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), downDir, true, false, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	_, err = os.Stat(downDir)
	c.Assert(err, IsNil)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), downDir, true, false, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	_, err = os.Stat(downDir)
	c.Assert(err, IsNil)

	// copy files
	destBucket := bucketName + "-" + randLowStr(4)
	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), CloudURLToString(destBucket, "123"), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.putBucket(destBucket, c)
	time.Sleep(4 * time.Second)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), CloudURLToString(destBucket, "123"), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	for _, filePath := range filePaths {
		s.getStat(destBucket, "123/"+filePath, c)
	}

	// remove dir
	os.RemoveAll(dir)
	os.RemoveAll(downDir)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUpdate(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// create older file and newer file
	oldData := "old data"
	oldFile := "oldFile" + randStr(5)
	newData := "new data"
	newFile := "newFile" + randStr(5)
	s.createFile(oldFile, oldData, c)
	s.createFile(newFile, newData, c)

	// put newer object
	object := "testobject"
	s.putObject(bucketName, object, newFile, c)

	// get object
	s.getObject(bucketName, object, downloadFileName, c)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, newData)

	// put old object with update
	showElapse, err := s.rawCP(oldFile, CloudURLToString(bucketName, object), false, false, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, newData)

	showElapse, err = s.rawCP(oldFile, CloudURLToString(bucketName, object), false, true, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, newData)

	showElapse, err = s.rawCP(oldFile, CloudURLToString(bucketName, object), false, false, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(bucketName, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, newData)

	// get object with update
	// modify downloadFile
	time.Sleep(time.Second * 1)
	downData := "download file has been modified locally"
	s.createFile(downloadFileName, downData, c)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), downloadFileName, false, false, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, downData)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), downloadFileName, false, true, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, downData)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), downloadFileName, false, false, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, downData)

	// copy object with update
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	destData := "data for dest bucket"
	destFile := "destFile" + randStr(5)
	s.createFile(destFile, destData, c)
	s.putObject(destBucket, object, destFile, c)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, object), false, false, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(destBucket, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, destData)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, object), false, true, true, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(destBucket, object, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, destData)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, object), false, false, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), CloudURLToString(destBucket, ""), true, false, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(oldFile)
	os.Remove(newFile)
	os.Remove(destFile)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestResumeCPObject(c *C) {
	var threshold int64
	threshold = 1
	cpDir := "checkpoint目录"

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	data := "resume cp"
	s.createFile(uploadFileName, data, c)

	// put object
	object := "object"
	showElapse, err := s.rawCP(uploadFileName, CloudURLToString(bucketName, object), false, true, false, threshold, cpDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// get object
	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), downloadFileName, false, true, false, threshold, cpDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	s.createFile(downloadFileName, "-------long file which must be truncated by cp file------", c)
	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), downloadFileName, false, true, false, threshold, cpDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	// copy object
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	destObject := "destObject"

	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), CloudURLToString(destBucket, destObject), false, true, false, threshold, cpDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.getObject(destBucket, destObject, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestCPMulitSrc(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// upload multi file
	file1 := uploadFileName + "1"
	s.createFile(file1, file1, c)
	file2 := uploadFileName + "2"
	s.createFile(file2, file2, c)
	showElapse, err := s.rawCPWithArgs([]string{file1, file2, CloudURLToString(bucketName, "")}, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	os.Remove(file1)
	os.Remove(file2)

	// download multi objects
	object1 := "object1"
	s.putObject(bucketName, object1, uploadFileName, c)
	object2 := "object2"
	s.putObject(bucketName, object2, uploadFileName, c)
	showElapse, err = s.rawCPWithArgs([]string{CloudURLToString(bucketName, object1), CloudURLToString(bucketName, object2), "../"}, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// copy multi objects
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)
	showElapse, err = s.rawCPWithArgs([]string{CloudURLToString(bucketName, object1), CloudURLToString(bucketName, object2), CloudURLToString(destBucket, "")}, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestErrUpload(c *C) {
	// src file not exist
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	showElapse, err := s.rawCP("notexistfile", CloudURLToString(bucketName, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// create local dir
	dir := randStr(3) + "上传目录"
	err = os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)
	cpDir := dir + string(os.PathSeparator) + CheckpointDir
	showElapse, err = s.rawCP(dir, CloudURLToString(bucketName, ""), true, true, true, DefaultBigFileThreshold, cpDir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// err object name
	showElapse, err = s.rawCP(uploadFileName, CloudURLToString(bucketName, "/object"), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(uploadFileName, CloudURLToString(bucketName, "/object"), false, true, false, 1, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	subdir := dir + string(os.PathSeparator) + "subdir"
	err = os.MkdirAll(subdir, 0755)
	c.Assert(err, IsNil)

	showElapse, err = s.rawCP(subdir, CloudURLToString(bucketName, "/object"), false, true, false, 1, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	os.RemoveAll(dir)
	os.RemoveAll(subdir)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrDownload(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "object"
	s.putObject(bucketName, object, uploadFileName, c)

	// download to dir, but dir exist as a file
	showElapse, err := s.rawCP(CloudURLToString(bucketName, object), configFile+string(os.PathSeparator), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// batch download without -r
	showElapse, err = s.rawCP(CloudURLToString(bucketName, ""), downloadFileName, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// download to file in not exist dir
	showElapse, err = s.rawCP(CloudURLToString(bucketName, object), configFile+string(os.PathSeparator)+downloadFileName, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrCopy(c *C) {
	srcBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(srcBucket, c)

	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	// batch copy without -r
	showElapse, err := s.rawCP(CloudURLToString(srcBucket, ""), CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// error src object name
	showElapse, err = s.rawCP(CloudURLToString(srcBucket, "/object"), CloudURLToString(destBucket, ""), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// err dest object
	object := "object"
	s.putObject(srcBucket, object, uploadFileName, c)
	showElapse, err = s.rawCP(CloudURLToString(srcBucket, object), CloudURLToString(destBucket, "/object"), false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(srcBucket, object), CloudURLToString(destBucket, "/object"), false, true, false, 1, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawCP(CloudURLToString(srcBucket, ""), CloudURLToString(destBucket, "/object"), true, true, false, 1, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(srcBucket, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestPreparePartOption(c *C) {
	partSize, routines := copyCommand.preparePartOption(0)
	c.Assert(partSize, Equals, int64(oss.MinPartSize))
	c.Assert(routines, Equals, 1)

	partSize, routines = copyCommand.preparePartOption(1)
	c.Assert(partSize, Equals, int64(oss.MinPartSize))
	c.Assert(routines, Equals, 1)

	partSize, routines = copyCommand.preparePartOption(300000)
	c.Assert(partSize, Equals, int64(oss.MinPartSize))
	c.Assert(routines, Equals, 2)

	partSize, routines = copyCommand.preparePartOption(20121443)
	c.Assert(partSize, Equals, int64(2560000))
	c.Assert(routines, Equals, 4)

	partSize, routines = copyCommand.preparePartOption(80485760)
	c.Assert(partSize, Equals, int64(2560000))
	c.Assert(routines, Equals, 8)

	partSize, routines = copyCommand.preparePartOption(500000000)
	c.Assert(partSize, Equals, int64(2561480))
	c.Assert(routines, Equals, 8)

	partSize, routines = copyCommand.preparePartOption(100000000000)
	c.Assert(partSize, Equals, int64(250000000))
	c.Assert(routines, Equals, 10)

	partSize, routines = copyCommand.preparePartOption(100000000000000)
	c.Assert(partSize, Equals, int64(10000000000))
	c.Assert(routines, Equals, 12)

	partSize, routines = copyCommand.preparePartOption(MaxInt64)
	c.Assert(partSize, Equals, int64(922337203685478))
	c.Assert(routines, Equals, 12)

	p := 7
	parallel := strconv.Itoa(p)
	copyCommand.command.options = make(OptionMapType, len(OptionMap))
	copyCommand.command.options[OptionParallel] = &parallel
	partSize, routines = copyCommand.preparePartOption(1)
	c.Assert(routines, Equals, p)

	p = 6
	parallel = strconv.Itoa(p)
	ps := 100000
	psStr := strconv.Itoa(ps)
	copyCommand.command.options[OptionParallel] = &parallel
	copyCommand.command.options[OptionPartSize] = &psStr
	partSize, routines = copyCommand.preparePartOption(1)
	c.Assert(routines, Equals, p)
	c.Assert(partSize, Equals, int64(ps))

	str := ""
	copyCommand.command.options[OptionParallel] = &str
}

func (s *OssutilCommandSuite) TestResumeDownloadRetry(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	bucket, err := copyCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	err = copyCommand.ossResumeDownloadRetry(bucket, "", "", 0, 0)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestCPIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "testobject"

	ufile := randStr(12)
	data := "欢迎使用ossutil"
	s.createFile(ufile, data, c)

	cfile := randStr(10)
	data = fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "cp"
	str := ""
	args := []string{ufile, CloudURLToString(bucketName, object)}
	ok := true
	routines := strconv.Itoa(Routines)
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	cpDir := CheckpointDir
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &cfile,
		"force":            &ok,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":         &endpoint,
		"accessKeyID":      &accessKeyID,
		"accessKeySecret":  &accessKeySecret,
		"stsToken":         &str,
		"configFile":       &cfile,
		"force":            &ok,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"partSize":         &partSize,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(ufile)
	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadOutputDir(c *C) {
	dir := randStr(10)
	os.RemoveAll(dir)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(10)
	ufile := randStr(12)
	data := "content"
	s.createFile(ufile, data, c)

	// normal copy -> no outputdir
	showElapse, err := s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// NoSuchBucket err copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketNameNotExist, object), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// SignatureDoesNotMatch err copy -> no outputdir
	cfile := configFile
	configFile = randStr(10)
	data = fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s", endpoint, accessKeyID, "abc", bucketName, endpoint)
	s.createFile(configFile, data, c)

	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	data = fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Cname]\n%s=%s", endpoint, accessKeyID, accessKeySecret, bucketName, "abc")
	s.createFile(configFile, data, c)

	// err copy without -r -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), false, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy with -r -> outputdir
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, dir)
	os.Stdout = out
	str := s.readFile(resultPath, c)
	c.Assert(strings.Contains(str, "Error occurs"), Equals, true)
	c.Assert(strings.Contains(str, "See more information in file"), Equals, true)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, IsNil)

	os.Remove(configFile)
	configFile = cfile

	// get file list of outputdir
	fileList, err := s.getFileList(dir)
	c.Assert(err, IsNil)
	c.Assert(len(fileList), Equals, 1)

	// get report file content
	result := s.getReportResult(fmt.Sprintf("%s%s%s", dir, string(os.PathSeparator), fileList[0]), c)
	c.Assert(len(result), Equals, 1)

	os.Remove(ufile)
	os.RemoveAll(dir)

	// err list with -C -> no outputdir
	udir := randStr(10)
	err = os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)
	showElapse, err = s.rawCPWithOutputDir(udir, CloudURLToString(bucketName, object), false, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	os.RemoveAll(udir)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchUploadOutputDir(c *C) {
	udir := randStr(10)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)

	num := 2
	filePaths := []string{}
	for i := 0; i < num; i++ {
		filePath := randStr(10)
		s.createFile(udir+"/"+filePath, fmt.Sprintf("测试文件：%d内容", i), c)
		filePaths = append(filePaths, filePath)
	}

	dir := randStr(10)
	os.RemoveAll(dir)
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// normal copy -> no outputdir
	showElapse, err := s.rawCPWithOutputDir(udir, CloudURLToString(bucketName, udir+"/"), true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy without -r -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(udir, CloudURLToString(bucketName, udir+"/"), false, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy -> outputdir
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", "abc", accessKeyID, accessKeySecret)
	s.createFile(configFile, data, c)
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	showElapse, err = s.rawCPWithOutputDir(udir, CloudURLToString(bucketName, udir+"/"), true, true, false, 1, dir)
	os.Stdout = out
	str := s.readFile(resultPath, c)
	c.Assert(strings.Contains(str, "Error occurs"), Equals, true)
	c.Assert(strings.Contains(str, "See more information in file"), Equals, true)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, IsNil)

	// get file list of outputdir
	fileList, err := s.getFileList(dir)
	c.Assert(err, IsNil)
	c.Assert(len(fileList), Equals, 1)

	// get report file content
	result := s.getReportResult(fmt.Sprintf("%s%s%s", dir, string(os.PathSeparator), fileList[0]), c)
	c.Assert(len(result), Equals, num)

	os.Remove(configFile)
	configFile = cfile
	eError := os.RemoveAll(dir)
	if eError != nil {
		fmt.Printf("remove error:%s %s.\n", dir, eError.Error())
	}

	// NoSuchBucket err copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(udir, CloudURLToString(bucketNameNotExist, udir+"/"), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	os.RemoveAll(udir)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDownloadOutputDir(c *C) {
	dir := randStr(10)
	os.RemoveAll(dir)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	// normal copy without -r -> no outputdir
	showElapse, err := s.rawCPWithOutputDir(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// normal copy with -r -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(bucketName, object), downloadDir, true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(bucketNameNotExist, object), downloadFileName, true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy without -r -> no outputdir
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Cname]\n%s=%s", endpoint, accessKeyID, accessKeySecret, bucketName, "abc")
	s.createFile(configFile, data, c)

	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(bucketName, object), downloadFileName, false, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// list err copy with -r -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(bucketName, object), downloadDir, true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	os.RemoveAll(dir)
	os.Remove(configFile)
	configFile = cfile

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCopyOutputDir(c *C) {
	dir := randStr(10)
	os.RemoveAll(dir)

	srcBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(srcBucket, c)
	destBucket := srcBucket + "-dest"
	s.putBucket(destBucket, c)

	object := randStr(10)
	s.putObject(srcBucket, object, uploadFileName, c)

	// normal copy -> no outputdir
	showElapse, err := s.rawCPWithOutputDir(CloudURLToString(srcBucket, object), CloudURLToString(destBucket, object), true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// err copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, object), CloudURLToString(bucketNameNotExist, object), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(bucketNameNotExist, object), CloudURLToString(destBucket, object), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// list err copy without -r -> no outputdir
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Cname]\n%s=%s", endpoint, accessKeyID, accessKeySecret, srcBucket, "abc")
	s.createFile(configFile, data, c)
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, object), CloudURLToString(destBucket, object), false, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, object), CloudURLToString(destBucket, object), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	os.Remove(configFile)
	configFile = cfile
	os.RemoveAll(dir)

	s.removeBucket(srcBucket, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestBatchCopyOutputDir(c *C) {
	dir := randStr(10)
	os.RemoveAll(dir)

	srcBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(srcBucket, c)
	destBucket := bucketNamePrefix + randLowStr(10)
	s.putBucket(destBucket, c)

	objectList := []string{}
	num := 3
	for i := 0; i < num; i++ {
		object := randStr(10)
		s.putObject(srcBucket, object, uploadFileName, c)
		objectList = append(objectList, object)
	}

	showElapse, err := s.rawCPWithOutputDir(CloudURLToString(srcBucket, objectList[0]), CloudURLToString(destBucket, ""), true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.RemoveAll(dir)

	// normal copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, ""), CloudURLToString(destBucket, ""), true, true, false, 1, dir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// bucketNameNotExist err copy -> no outputdir
	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, ""), CloudURLToString(bucketNameNotExist, ""), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// test objectStatistic err
	cfile := configFile
	configFile = randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", srcBucket, "abc", srcBucket, "abc")
	s.createFile(configFile, data, c)

	showElapse, err = s.rawCPWithOutputDir(CloudURLToString(srcBucket, ""), CloudURLToString(destBucket, ""), true, true, false, 1, dir)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	os.Remove(configFile)
	configFile = cfile
	os.RemoveAll(dir)

	s.removeBucket(srcBucket, true, c)
	s.removeBucket(destBucket, true, c)
}

func (s *OssutilCommandSuite) TestConfigOutputDir(c *C) {
	// test default outputdir
	edir := ""
	dir := randStr(10) + "testconfigoutputdir"
	dir1 := dir + "another"
	os.RemoveAll(DefaultOutputDir)
	os.RemoveAll(dir)
	os.RemoveAll(dir1)

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(10)
	ufile := randStr(12)
	data := "content"
	s.createFile(ufile, data, c)

	// err copy -> outputdir
	cfile := configFile
	configFile = randStr(10)
	data = fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Cname]\n%s=%s", endpoint, accessKeyID, accessKeySecret, bucketName, "abc")
	s.createFile(configFile, data, c)

	showElapse, err := s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, edir)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(DefaultOutputDir)
	c.Assert(err, IsNil)

	// get file list of outputdir
	fileList, err := s.getFileList(DefaultOutputDir)
	c.Assert(err, IsNil)
	c.Assert(len(fileList), Equals, 1)

	os.RemoveAll(DefaultOutputDir)

	// config outputdir
	data = fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\noutputDir=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", endpoint, accessKeyID, accessKeySecret, dir, bucketName, endpoint, bucketName, "abc")
	s.createFile(configFile, data, c)

	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir)
	c.Assert(err, IsNil)
	_, err = os.Stat(DefaultOutputDir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// get file list of outputdir
	fileList, err = s.getFileList(dir)
	c.Assert(err, IsNil)
	c.Assert(len(fileList), Equals, 1)

	err = os.RemoveAll(dir)
	c.Assert(err, IsNil)

	err = os.RemoveAll(DefaultOutputDir)
	c.Assert(err, IsNil)

	// option and config
	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, dir1)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
	_, err = os.Stat(dir1)
	c.Assert(err, IsNil)
	_, err = os.Stat(dir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)
	_, err = os.Stat(DefaultOutputDir)
	c.Assert(err, NotNil)
	c.Assert(os.IsNotExist(err), Equals, true)

	// get file list of outputdir
	fileList, err = s.getFileList(dir1)
	c.Assert(err, IsNil)
	c.Assert(len(fileList), Equals, 1)

	os.Remove(configFile)
	configFile = cfile
	os.RemoveAll(dir1)
	os.RemoveAll(dir)
	os.RemoveAll(DefaultOutputDir)

	s.createFile(uploadFileName, content, c)
	showElapse, err = s.rawCPWithOutputDir(ufile, CloudURLToString(bucketName, object), true, true, false, 1, uploadFileName)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestInitReportError(c *C) {
	s.createFile(uploadFileName, content, c)
	report, err := GetReporter(false, DefaultOutputDir, "")
	c.Assert(err, IsNil)
	c.Assert(report, IsNil)

	report, err = GetReporter(true, uploadFileName, "")
	c.Assert(err, NotNil)
	c.Assert(report, IsNil)
}

func (s *OssutilCommandSuite) TestCopyFunction(c *C) {
	// test fileStatistic
	copyCommand.monitor.init(operationTypePut)
	storageURL, err := StorageURLFromString("&~", "")
	c.Assert(err, IsNil)
	copyCommand.fileStatistic([]StorageURLer{storageURL})
	c.Assert(copyCommand.monitor.seekAheadEnd, Equals, true)
	c.Assert(copyCommand.monitor.seekAheadError, NotNil)

	// test fileProducer
	chFiles := make(chan fileInfoType, ChannelBuf)
	chListError := make(chan error, 1)
	storageURL, err = StorageURLFromString("&~", "")
	c.Assert(err, IsNil)
	copyCommand.fileProducer([]StorageURLer{storageURL}, chFiles, chListError)
	err = <-chListError
	c.Assert(err, NotNil)

	// test put object error
	bucketName := bucketNameNotExist
	bucket, err := copyCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	err = copyCommand.ossPutObjectRetry(bucket, "object", "")
	c.Assert(err, NotNil)

	// test formatResultPrompt
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	out := os.Stdout
	os.Stdout = testResultFile
	err = fmt.Errorf("test error")
	copyCommand.cpOption.ctnu = true
	err = copyCommand.formatResultPrompt(err)
	c.Assert(err, IsNil)
	os.Stdout = out
	str := strings.ToLower(s.readFile(resultPath, c))
	c.Assert(strings.Contains(str, "succeed"), Equals, true)
	c.Assert(strings.Contains(str, "error"), Equals, false)

	// test download file error
	err = copyCommand.ossDownloadFileRetry(bucket, "object", downloadFileName)
	c.Assert(err, NotNil)

	// test truncateFile
	err = copyCommand.truncateFile("ossutil_notexistfile", 1)
	c.Assert(err, NotNil)
	s.createFile(uploadFileName, "abc", c)
	err = copyCommand.truncateFile(uploadFileName, 1)
	c.Assert(err, IsNil)
	str = s.readFile(uploadFileName, c)
	c.Assert(str, Equals, "a")
}

// test fileProducer
func (s *OssutilCommandSuite) TestFileProducer(c *C) {
	chFiles := make(chan fileInfoType, ChannelBuf)
	chListError := make(chan error, 1)
	storageURL, err := StorageURLFromString("&~", "")
	c.Assert(err, IsNil)
	copyCommand.fileProducer([]StorageURLer{storageURL}, chFiles, chListError)
	err = <-chListError
	c.Assert(err, NotNil)

	select {
	case _, ok := <-chFiles:
		testLogger.Printf("chFiles channel has closed")
		c.Assert(ok, Equals, false)
	}

	chFiles2 := make(chan fileInfoType, ChannelBuf)
	chListError2 := make(chan error, 1)
	storageURL, err = StorageURLFromString("cp_test.go", "")
	c.Assert(err, IsNil)
	copyCommand.fileProducer([]StorageURLer{storageURL}, chFiles2, chListError2)
	err = <-chListError2
	c.Assert(err, IsNil)

	select {
	case i, ok := <-chFiles2:
		testLogger.Printf("%#v\n", i)
		c.Assert(ok, Equals, true)
		c.Assert(i, Equals, fileInfoType{filePath: "cp_test.go", dir: ""})
	}

	select {
	case _, ok := <-chFiles:
		testLogger.Printf("chFiles channel has closed")
		c.Assert(ok, Equals, false)
	}
}

// test objectProducer
func (s *OssutilCommandSuite) TestCpObjectProducer(c *C) {
	chObjects := make(chan objectInfoType, ChannelBuf)
	chListError := make(chan error, 1)
	cloudURL, err := CloudURLFromString(CloudURLToString(bucketNameNotExist, "demo.txt"), "")
	c.Assert(err, IsNil)
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)
	bucket, err := client.Bucket(bucketNameNotExist)
	c.Assert(err, IsNil)
	copyCommand.objectProducer(bucket, cloudURL, chObjects, chListError)
	err = <-chListError
	c.Assert(err, NotNil)
	select {
	case _, ok := <-chObjects:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}

	chObjects2 := make(chan objectInfoType, ChannelBuf)
	chListError2 := make(chan error, 1)
	storageURL2, err := CloudURLFromString(CloudURLToString(bucketNameExist, ""), "")
	c.Assert(err, IsNil)
	bucket2, err := client.Bucket(bucketNameExist)
	c.Assert(err, IsNil)
	copyCommand.objectProducer(bucket2, storageURL2, chObjects2, chListError2)
	err = <-chListError2
	c.Assert(err, IsNil)
	select {
	case _, ok := <-chObjects:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}
}

func (s *OssutilCommandSuite) TestCPURLEncoding(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	specialStr := "中文测试" + randStr(5)
	encodedStr := url.QueryEscape(specialStr)
	s.createFile(specialStr, content, c)

	args := []string{encodedStr, CloudURLToString(bucketName, encodedStr)}
	command := "cp"
	str := ""
	thre := strconv.FormatInt(1, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	encodingType := URLEncodingType
	cpDir := CheckpointDir
	outputDir := DefaultOutputDir
	ok := true
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"stsToken":         &str,
		"configFile":       &configFile,
		"force":            &ok,
		"bigfileThreshold": &thre,
		"checkpointDir":    &cpDir,
		"outputDir":        &outputDir,
		"routines":         &routines,
		"partSize":         &partSize,
		"encodingType":     &encodingType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	objects := s.listLimitedMarker(bucketName, encodedStr, "ls --encoding-type url", -1, "", "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, specialStr)

	objects = s.listLimitedMarker(bucketName, specialStr, "ls ", -1, "", "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, specialStr)

	// get object
	downloadFileName := "下载文件" + randLowStr(3)
	args = []string{CloudURLToString(bucketName, encodedStr), url.QueryEscape(downloadFileName)}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	downStr := strings.ToLower(s.readFile(downloadFileName, c))
	c.Assert(downStr, Equals, content)

	// copy object
	destObject := "拷贝文件" + randLowStr(3)
	args = []string{CloudURLToString(bucketName, encodedStr), CloudURLToString(bucketName, url.QueryEscape(destObject))}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// get object
	s.getStat(bucketName, destObject, c)

	os.Remove(specialStr)
	os.Remove(downloadFileName)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPFunction(c *C) {
	var srcURL CloudURL
	srcURL.bucket = ""
	var destURL CloudURL
	err := copyCommand.checkCopyArgs([]StorageURLer{srcURL}, destURL, operationTypePut)
	c.Assert(err, NotNil)

	err = copyCommand.getFileListStatistic("notexistdir")
	c.Assert(err, NotNil)

	chFiles := make(chan fileInfoType, 100)
	err = copyCommand.getFileList("notexistdir", chFiles)

	bucketName := bucketNamePrefix + randLowStr(10)
	bucket, err := copyCommand.command.ossBucket(bucketName)
	destURL.bucket = bucketName
	destURL.object = "abc"
	var fileInfo fileInfoType
	fileInfo.filePath = "a"
	fileInfo.dir = "notexistdir"
	_, err, _, _, _ = copyCommand.uploadFile(bucket, destURL, fileInfo)
	c.Assert(err, NotNil)
}

// Test: --include '*.txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNormalInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*.txt"
	downdir := "testdownload-inc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*.txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt") and use these for verification
	files := filterStrsWithInclude(filenames, "*.txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*2??txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithMarkInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc2" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*2??txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*2??txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*2??txt"
	downdir := "testdownload-inc2" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*2??txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*2??txt") and use these for verification
	files := filterStrsWithInclude(filenames, "*2??txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestBatchCPObjectWithSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc3" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*[0-9]?jpg'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*[0-9]?jpg"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*2??txt"
	downdir := "testdownload-inc3" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*[0-9]?jpg"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*[0-9]?jpg") and use these for verification
	files := filterStrsWithInclude(filenames, "*[0-9]?jpg")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNonSequenceInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc4" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*[^0-3]?txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*[^0-3]?txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*[^0-3]?txt""
	downdir := "testdownload-inc4" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*[^0-3]?txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*[0-9]?jpg") and use these for verification
	files := filterStrsWithInclude(filenames, "*[^0-3]?txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNonSequenceIncludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc4-ex" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*[!0-3]?txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*[!0-3]?txt""
	downdir := "testdownload-inc4-ex" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*[!0-3]?txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	files := filterStrsWithInclude(filenames, filters[0].pattern)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --include '*.jpg'
func (s *OssutilCommandSuite) TestBatchCPObjectWithRepeatedInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc5" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// Test: repeated --include
	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*.jpg' --include '*.jpg'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.jpg", "--include", "*.jpg"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*.jpg" --include "*.jpg"
	downdir := "testdownload-inc5" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*.jpg", "--include", "*.jpg"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.jpg" --include "*.jpg") and use these for verification
	files := filterStrsWithInclude(filenames, "*.jpg")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*'
func (s *OssutilCommandSuite) TestBatchCPObjectWithFullInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc6" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --include '*'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --include "*"
	downdir := "testdownload-inc6" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*.txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNormalExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-exc/ oss://tempb4 -rf --exclude '*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*.txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --exclude uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*.txt"
	downdir := "testdownload-exc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*.txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*.txt") and use these for verification
	files := filterStrsWithExclude(filenames, "*.txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*2??txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithMarkExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc2" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-exc/ oss://tempb4 -rf --exclude '*2??txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*2??txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --exclude uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*2??txt"
	downdir := "testdownload-exc2" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*2??txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*2??txt") and use these for verification
	files := filterStrsWithExclude(filenames, "*2??txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[0-9]?jpg'
func (s *OssutilCommandSuite) TestBatchCPObjectWithSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc3" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-exc/ oss://tempb4 -rf --exclude '*[0-9]?jpg'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*[0-9]?jpg"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --exclude uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*2??txt"
	downdir := "testdownload-exc3" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*[0-9]?jpg"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*[0-9]?jpg") and use these for verification
	files := filterStrsWithExclude(filenames, "*[0-9]?jpg")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[^0-3]?txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNonSequenceExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc4" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-exc/ oss://tempb4 -rf --exclude '*[^0-3]?txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*[^0-3]?txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --exclude uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*2??txt"
	downdir := "testdownload-exc4" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*[^0-3]?txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*[^0-3]?txt") and use these for verification
	files := filterStrsWithExclude(filenames, "*[^0-3]?txt")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*[!0-3]?txt'
func (s *OssutilCommandSuite) TestBatchCPObjectWithNonSequenceExcludeEx(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc4-ex" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-exc/ oss://tempb4 -rf --exclude '*[!0-3]?txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*[!0-3]?txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --exclude uploaded
	downdir := "testdownload-exc4" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*[!0-3]?txt"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*[^0-3]?txt") and use these for verification
	res, filters := getFilter(cmdline)
	c.Assert(res, Equals, true)
	files := filterStrsWithExclude(filenames, filters[0].pattern)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: repeated --exclude '*.jpg'
func (s *OssutilCommandSuite) TestBatchCPObjectWithRepeatedExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc5" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --exclude '*.jpg' --exclude '*.jpg'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*.jpg", "--exclude", "*.jpg"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*.jpg" --exclude "*.jpg"
	downdir := "testdownload-exc5" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*.jpg", "--exclude", "*.jpg"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*.jpg" --exclude "*.jpg") and use these for verification
	files := filterStrsWithExclude(filenames, "*.jpg")

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*'
func (s *OssutilCommandSuite) TestBatchCPObjectWithFullExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-exc6" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --exclude '*'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download above files with --include uploaded
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf --exclude "*"
	downdir := "testdownload-exc6" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --exclude "*[^0-3]?txt") and use these for verification
	files := filterStrsWithExclude(filenames, "*")
	c.Assert(len(files), Equals, 0)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		winErrorFile := strings.Contains(err.Error(), "system cannot find the file specified")
		winErrorPath := strings.Contains(err.Error(), "system cannot find the path specified")
		linuxError := strings.Contains(err.Error(), "no such file or directory")
		c.Assert((winErrorFile || winErrorPath || linuxError), Equals, true)
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiNormalIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include '*.txt' --exclude "*2*"
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}}
	files := matchFiltersForStrs(filenames, fts)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test repeated: --include '*.txt' --exclude "*2*" --include '*.txt' --exclude "*2*"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiRepeatedIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc-repeated" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include '*.txt' --exclude "*2*" --include '*.txt' --exclude "*2*"
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.txt", "--exclude", "*2*", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc-repeated" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--include", "*.txt", "--exclude", "*2*", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--include", "*.txt"}, {"--exclude", "*2*"}, {"--include", "*.txt"}, {"--exclude", "*2*"}}
	files := matchFiltersForStrs(filenames, fts)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --include '*' --exclude "*"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiFullIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc2" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include '*' --exclude "*"
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*", "--exclude", "*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc2" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--include", "*"}, {"--exclude", "*"}}
	files := matchFiltersForStrs(filenames, fts)
	c.Assert(len(files), Equals, 0)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)

		winErrorFile := strings.Contains(err.Error(), "system cannot find the file specified")
		winErrorPath := strings.Contains(err.Error(), "system cannot find the path specified")
		linuxError := strings.Contains(err.Error(), "no such file or directory")
		c.Assert((winErrorFile || winErrorPath || linuxError), Equals, true)
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude '*' --include "*"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiFullExcludeInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc3" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --exclude '*' --include "*"
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude", "*", "--include", "*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc3" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude", "*", "--include", "*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--exclude", "*"}, {"--include", "*"}}
	files := matchFiltersForStrs(filenames, fts)
	c.Assert(len(files), Equals, len(filenames))

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Nagetive test
func (s *OssutilCommandSuite) TestBatchCPObjectWithInvalidIncludeExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	//0. Create dirs
	dir := "testdir-invalid" + randLowStr(5)
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	// testdir-invalid/dir1
	subdir := "dir1"
	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	// upload files
	// e.g., ossutil cp testdir-invalid/ oss://tempb4 -f --exclude '*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-f", "--exclude", "*.txt"}
	showElapse, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	// e.g., ossutil cp testdir-invalid/ oss://tempb4 -rf --include '*.txt' --exclude "*2*"
	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include", "/*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	// download
	// e.g., ossutil cp oss://tempb4/ testdownload/ -f --exclude "*.txt"
	downdir := "testdownload-invalid" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--exclude", "*.txt"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--include", "*.txt", "--exclude", "*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include", "*.txt", "--exclude", "/usr/*/*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	// download test with --meta, --acl
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--meta", "Cache-Control:no-cache"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache", "")
	c.Assert(err, IsNil)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--acl", "public-read"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "public-read")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "No need to set ACL for download", Equals, true)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--meta", "Cache-Control:no-cache"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache", "")
	c.Assert(err, IsNil)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--acl", "public-read"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "public-read")
	c.Assert(err, NotNil)

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: --exclude='*' --include="*"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiFullExcludeIncludeEqual(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc3" + randLowStr(5)
	subdir := "dir1" + randLowStr(5)
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --exclude='*' --include="*"
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude=*", "--include=*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc3" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--exclude=*", "--include=*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--exclude", "*"}, {"--include", "*"}}
	files := matchFiltersForStrs(filenames, fts)
	c.Assert(len(files), Equals, len(filenames))

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Nagetive test
func (s *OssutilCommandSuite) TestBatchCPObjectWithInvalidIncludeExcludeEqual(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	//0. Create dirs
	dir := "testdir-invalid" + randLowStr(5)
	err := os.MkdirAll(dir, 0755)

	// testdir-invalid/dir1
	subdir := "dir1" + randLowStr(5)
	err = os.MkdirAll(dir+string(os.PathSeparator)+subdir, 0755)
	c.Assert(err, IsNil)

	// upload files
	// e.g., ossutil cp testdir-invalid/ oss://tempb4 -f --exclude='*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-f", "--exclude=*.txt"}
	showElapse, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	// e.g., ossutil cp testdir-invalid/ oss://tempb4 -rf --include=*.txt --exclude=*2*
	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include=*.txt", "--exclude=*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include=/*.txt", "--exclude=*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	// download
	// e.g., ossutil cp oss://tempb4/ testdownload/ -f --exclude "*.txt"
	downdir := "testdownload-invalid" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--exclude=*.txt"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--include=*.txt", "--exclude=*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude only work with --recursive", Equals, true)

	cmdline = []string{"ossutil", "cp", dir, bucketStr, "-f", "--include=*.txt", "--exclude=/usr/*/*2*"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "--include or --exclude does not support format containing dir info", Equals, true)

	// download test with --meta, --acl
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--meta", "Cache-Control:no-cache"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache", "")
	c.Assert(showElapse, Equals, true)
	//c.Assert(err.Error() == "No need to set meta for download", Equals, true)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf", "--acl", "public-read"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "public-read")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "No need to set ACL for download", Equals, true)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--meta", "Cache-Control:no-cache"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache", "")
	c.Assert(err, NotNil)

	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-f", "--acl", "public-read"}
	showElapse, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "public-read")
	c.Assert(showElapse, Equals, false)
	c.Assert(err.Error() == "No need to set ACL for download", Equals, true)

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: only --include="*.txt"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiNormalOnlyIncludeEqual(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include='*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include=*.txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include="*.txt") and use these for verification
	fts := []filterOptionType{{"--include", "*.txt"}}
	files := matchFiltersForStrs(filenames, fts)

	ftsJpg := []filterOptionType{{"--include", "*.jpg"}}
	filesJpg := matchFiltersForStrs(filenames, ftsJpg)
	c.Assert(len(filesJpg) > 0, Equals, true)

	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	// exist file check
	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// not exist file check
	for _, filename := range filesJpg {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, NotNil)
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: only --exclude="*.txt"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiNormalOnlyExcludeEqual(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --exclude='*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--exclude=*.txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	ftsTxt := []filterOptionType{{"--include", "*.txt"}}
	filesTxt := matchFiltersForStrs(filenames, ftsTxt)

	ftsJpg := []filterOptionType{{"--include", "*.jpg"}}
	filesJpg := matchFiltersForStrs(filenames, ftsJpg)
	c.Assert(len(filesJpg) > 0, Equals, true)

	ftsRtf := []filterOptionType{{"--include", "*.rtf"}}
	filesRtf := matchFiltersForStrs(filenames, ftsRtf)
	c.Assert(len(filesRtf) > 0, Equals, true)

	existFiles := make([]string, 0)
	existFiles = append(existFiles, filesJpg...)
	existFiles = append(existFiles, filesRtf...)
	c.Assert(len(existFiles), Equals, len(filesJpg)+len(filesRtf))

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	// exist file check
	for _, filename := range existFiles {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// not exist file check
	for _, filename := range filesTxt {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, NotNil)
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test: only --include "*.jpg" --exclude="*.txt"
func (s *OssutilCommandSuite) TestBatchCPObjectWithMultiNormalIncludeMixtureExcludeEqual(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-inc-exc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include "*.jpg" --exclude='*.txt'
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.jpg", "--exclude=*.txt"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-inc-exc1" + randLowStr(5)
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	ftsTxt := []filterOptionType{{"--include", "*.txt"}}
	filesTxt := matchFiltersForStrs(filenames, ftsTxt)

	ftsJpg := []filterOptionType{{"--include", "*.jpg"}}
	filesJpg := matchFiltersForStrs(filenames, ftsJpg)
	c.Assert(len(filesJpg) > 0, Equals, true)

	ftsRtf := []filterOptionType{{"--include", "*.rtf"}}
	filesRtf := matchFiltersForStrs(filenames, ftsRtf)
	c.Assert(len(filesRtf) > 0, Equals, true)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	// exist file check
	for _, filename := range filesJpg {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])
	}

	// not exist file check
	for _, filename := range filesTxt {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, NotNil)
	}

	// not exist file check
	for _, filename := range filesRtf {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, NotNil)
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test for --meta, --acl testing
func (s *OssutilCommandSuite) TestBatchCPObjectWithMetaAcl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-meta"
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-inc-exc/ oss://tempb4 -rf --include '*.txt' --exclude "*2*" --meta Cache-Control:no-cache#X-Oss-Meta-Test:with-pattern --acl public-read
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf", "--include", "*.txt", "--include", "*.jpg", "--exclude", "*2*"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache#X-Oss-Meta-Test:with-pattern", "public-read")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-meta"
	args = []string{bucketStr, downdir}
	cmdline = []string{"ossutil", "cp", bucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get uploaded files (with above conditions: --include "*.txt" --include "*.jpg" --exclude "*2*") and use these for verification
	fts := []filterOptionType{{"--include", "*.txt"}, {"--include", "*.jpg"}, {"--exclude", "*2*"}}
	files := matchFiltersForStrs(filenames, fts)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])

		objectStat := s.getStat(bucketName, filename, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
		c.Assert(objectStat["Cache-Control"], Equals, "no-cache")
		c.Assert(objectStat["X-Oss-Meta-Test"], Equals, "with-pattern")
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

// Test for copy objects, with --include, --exclude, --meta, --acl testing
func (s *OssutilCommandSuite) TestBatchCPObjectBetweenOssWithMetaAcl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	dstBucketName := bucketName + "-dest"
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")
	s.putBucket(dstBucketName, c)
	dstBucketStr := CloudURLToString(dstBucketName, "")

	dir := "testdir-oss-copy"
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	// e.g., ossutil cp testdir-oss-copy/ oss://tempb4 -rf
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Copy between 2 buckets
	args = []string{bucketStr, dstBucketStr}
	cmdline = []string{"ossutil", "cp", bucketStr, dstBucketStr, "-rf", "--include", "*", "--exclude", "*10*"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "Cache-Control:no-cache-ok#X-Oss-Meta-Test-Copy-Oss:copy-between-oss", "public-read")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// download all above files for verification
	// e.g., ossutil cp oss://tempb4/ testdownload/ -rf
	downdir := "testdownload-oss-copy"
	args = []string{dstBucketStr, downdir}
	cmdline = []string{"ossutil", "cp", dstBucketStr, downdir, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// Get files (with above conditions: --include "*" --exclude "*10*") and use these for verification
	fts := []filterOptionType{{"--include", "*"}, {"--exclude", "*10*"}}
	files := matchFiltersForStrs(filenames, fts)

	// Verify
	_, err = os.Stat(downdir)
	c.Assert(err, IsNil)

	for _, filename := range files {
		tname := downdir + "/" + filename
		_, err := os.Stat(tname)
		c.Assert(err, IsNil)

		content := s.readFile(tname, c)
		c.Assert(content, Equals, contents[filename])

		objectStat := s.getStat(dstBucketName, filename, c)
		c.Assert(objectStat["ACL"], Equals, "public-read")
		c.Assert(objectStat["Cache-Control"], Equals, "no-cache-ok")
		c.Assert(objectStat["X-Oss-Meta-Test-Copy-Oss"], Equals, "copy-between-oss")
	}

	// cleanup
	os.RemoveAll(dir)
	os.RemoveAll(downdir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectLimitSpeed(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	maxUpSpeed := int64(2) // 2KB/s
	upSecond := 5
	objectContext := randLowStr(int(maxUpSpeed) * upSecond * 1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"maxupspeed":      &maxUpSpeed,
	}

	// calculate time
	startT := time.Now()
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)
	endT := time.Now()
	costSecond := endT.UnixNano()/1000/1000 - startT.UnixNano()/1000/1000
	c.Assert(costSecond >= int64(upSecond)*1000, Equals, true)

	//down object
	downFileName := fileName + "-down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPDirLimitSpeed(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// single dir
	udir := "ossutil_test_" + randStr(5)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)

	// prepare upload parameter
	maxUpSpeed := int64(2) // 2KB/s
	upSecond := 5
	objectContext := randLowStr(int(maxUpSpeed) * upSecond * 1024)

	// prepare two file
	fileCount := 2
	objectFirst := randStr(5) + "1"
	objectSecond := randStr(5) + "2"
	s.createFile(udir+string(os.PathSeparator)+objectFirst, objectContext, c)
	s.createFile(udir+string(os.PathSeparator)+objectSecond, objectContext, c)

	// begin cp dir
	cpArgs := []string{udir, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"maxupspeed":      &maxUpSpeed,
	}

	// calculate time
	startT := time.Now()
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)
	endT := time.Now()
	costSecond := endT.UnixNano()/1000/1000 - startT.UnixNano()/1000/1000
	c.Assert(costSecond >= int64(upSecond)*1000*int64(fileCount), Equals, true)

	//down object 1
	delete(options, "recursive")
	downFileName := objectFirst
	dwArgs := []string{CloudURLToString(bucketName, objectFirst), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	//down object 2
	downFileName = objectSecond
	dwArgs = []string{CloudURLToString(bucketName, objectSecond), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	// compare content
	fileBody, err = ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	os.Remove(downFileName)
	err = os.Remove(udir + string(os.PathSeparator) + objectFirst)
	err = os.Remove(udir + string(os.PathSeparator) + objectSecond)
	err = os.RemoveAll(udir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPDownloadDirLimitSpeed(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// single dir
	udir := "ossutil_test_" + randStr(5)
	os.RemoveAll(udir)
	err := os.MkdirAll(udir, 0755)
	c.Assert(err, IsNil)

	// prepare upload parameter
	objectLen := 1024 * 1024
	objectContext := randLowStr(objectLen)

	// prepare two file
	fileCount := 2
	objectFirst := "ossutil-test-" + randStr(5) + "1"
	objectSecond := "ossutil-test-" + randStr(5) + "2"
	s.createFile(udir+string(os.PathSeparator)+objectFirst, objectContext, c)
	s.createFile(udir+string(os.PathSeparator)+objectSecond, objectContext, c)

	// begin cp dir
	cpArgs := []string{udir, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	force := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"force":           &force,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//begin download file
	downDir := udir + "_down"
	cpArgs = []string{CloudURLToString(bucketName, ""), downDir}
	maxDownSpeed := int64(500)
	options["maxdownspeed"] = &maxDownSpeed

	// calculate time
	startT := time.Now()
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)
	endT := time.Now()
	costSecond := endT.UnixNano()/1000/1000/1000 - startT.UnixNano()/1000/1000/1000

	// KB/s
	downloadSpeed := (float64)(fileCount*objectLen) / (float64)(costSecond) / 1024
	c.Assert(downloadSpeed <= (float64)(maxDownSpeed)*1.2, Equals, true)
	c.Assert(downloadSpeed >= (float64)(maxDownSpeed)*0.8, Equals, true)

	//compare content
	fileBody, err := ioutil.ReadFile(downDir + string(os.PathSeparator) + objectFirst)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	// compare content
	fileBody, err = ioutil.ReadFile(downDir + string(os.PathSeparator) + objectSecond)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	err = os.RemoveAll(udir)
	err = os.RemoveAll(downDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPPartionDownloadSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)

	objectFirst := ""
	objectSecond := ""
	for {
		tempStr := randStr(12)
		fnvIns := fnv.New64()
		fnvIns.Write([]byte(tempStr))
		if fnvIns.Sum64()%2 == 0 {
			objectFirst = tempStr
		} else {
			objectSecond = tempStr
		}
		if objectFirst != "" && objectSecond != "" {
			break
		} else {
			time.Sleep(10 * time.Millisecond)
		}
	}
	s.putObject(bucketName, objectFirst, fileName, c)
	s.putObject(bucketName, objectSecond, fileName, c)

	downloadPath := "." + string(os.PathSeparator) + randLowStr(10)
	err := os.MkdirAll(downloadPath, 0755)
	c.Assert(err, IsNil)

	// download objectFirst
	command := "cp"
	str := ""
	cpDir := CheckpointDir
	strMultiInstance := ""
	bRecursive := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &ConfigFile,
		"checkpointDir":     &cpDir,
		"recursive":         &bRecursive,
		"routines":          &routines,
		"partitionDownload": &strMultiInstance,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args := []string{srcUrl, downloadPath}

	strMultiInstance = "1:2"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// checkfile objectFirst success
	fileInfo, err := os.Stat(downloadPath + string(os.PathSeparator) + objectFirst)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Size(), Equals, int64(len(content)))

	// checkfile objectSecond Error
	fileInfo, err = os.Stat(downloadPath + string(os.PathSeparator) + objectSecond)
	c.Assert(err, NotNil)

	os.RemoveAll(downloadPath)
	err = os.MkdirAll(downloadPath, 0755)
	c.Assert(err, IsNil)

	// download objectSecond
	strMultiInstance = "2:2"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// checkfile objectFirst error
	fileInfo, err = os.Stat(downloadPath + string(os.PathSeparator) + objectFirst)
	c.Assert(err, NotNil)

	// checkfile objectSecond success
	fileInfo, err = os.Stat(downloadPath + string(os.PathSeparator) + objectSecond)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Size(), Equals, int64(len(content)))

	os.RemoveAll(fileName)
	os.RemoveAll(downloadPath)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPPartitionDownloadParameterError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	downloadPath := "." + string(os.PathSeparator) + randLowStr(10)

	command := "cp"
	str := ""
	cpDir := CheckpointDir
	strMultiInstance := ""
	bRecursive := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &ConfigFile,
		"checkpointDir":     &cpDir,
		"recursive":         &bRecursive,
		"routines":          &routines,
		"partitionDownload": &strMultiInstance,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args := []string{srcUrl, downloadPath}

	// error 1
	strMultiInstance = "-1:2"
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error 2
	strMultiInstance = "2:1"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error 3
	strMultiInstance = "abc:1"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error 4
	strMultiInstance = "1:abc"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error 5
	strMultiInstance = "1:2:3"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error 6
	strMultiInstance = ""
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// error7
	strMultiInstance = "1:2"
	args = []string{downloadPath, srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestCPDownloadSnapshot(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	//download snapshot dir
	testSnapshotDir := "ossutil_test_snapshot" + randStr(5)
	os.RemoveAll(testSnapshotDir)

	// download object dir
	downloadDir := "ossutil_test_" + randStr(5)
	os.RemoveAll(downloadDir)
	err := os.MkdirAll(downloadDir, 0755)
	c.Assert(err, IsNil)

	// put object1
	testUploadFileName := "ossutil_test_uploadfile" + randStr(5)
	data := "test"
	object1 := "ossutil_test_object1" + randStr(5)
	s.createFile(testUploadFileName, data, c)
	s.putObject(bucketName, object1, testUploadFileName, c)

	// download with snapshot
	cpArgs := []string{CloudURLToString(bucketName, ""), downloadDir}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"snapshotPath":    &testSnapshotDir,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//check download file object1
	fileInfo, err := os.Stat(downloadDir + string(os.PathSeparator) + object1)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Size(), Equals, int64(len(data)))

	//remove downloadfile and check
	err = os.Remove(downloadDir + string(os.PathSeparator) + object1)
	c.Assert(err, IsNil)
	fileInfo, err = os.Stat(downloadDir + string(os.PathSeparator) + object1)
	c.Assert(err, NotNil)

	// put object2
	object2 := "ossutil_test_object2" + randStr(5)
	s.putObject(bucketName, object2, testUploadFileName, c)

	// download with cp download snapshot
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//check download file object2
	fileInfo, err = os.Stat(downloadDir + string(os.PathSeparator) + object2)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Size(), Equals, int64(len(data)))

	// check download file object1 error
	fileInfo, err = os.Stat(downloadDir + string(os.PathSeparator) + object1)
	c.Assert(err, NotNil)

	//remove snapshot file
	err = os.RemoveAll(testSnapshotDir)
	c.Assert(err, IsNil)

	// download again
	ok := true
	options["update"] = &ok
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check download file object1 success
	fileInfo, err = os.Stat(downloadDir + string(os.PathSeparator) + object1)
	c.Assert(fileInfo.Size(), Equals, int64(len(data)))

	os.Remove(testUploadFileName)
	err = os.RemoveAll(downloadDir)
	err = os.RemoveAll(testSnapshotDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPVersioingParameterError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	cpTestDir := "ossutil_test_" + randStr(5)

	// error:upload with version-id
	cpArgs := []string{cpTestDir, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	versionId := "123"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	//error:download with -r
	recursive := true
	options["recursive"] = &recursive
	cpArgs = []string{CloudURLToString(bucketName, ""), cpTestDir}
	c.Assert(err, NotNil)
}

// down load with versionId
func (s *OssutilCommandSuite) TestCPVersioingDownloadSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	// put object
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)

	// get versionID
	objectStat := s.getStat(bucketName, objectName, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)

	// overwrite object
	s.createFile(testFileName, randStr(200), c)
	s.putObject(bucketName, objectName, testFileName, c)

	downFileName := "ossutil_test_" + randStr(5)

	//download with version-id
	cpArgs := []string{CloudURLToString(bucketName, objectName), downFileName}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check download file
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(testFileName)
	os.RemoveAll(downFileName)
	s.removeBucket(bucketName, true, c)
}

// down load with versionId
func (s *OssutilCommandSuite) TestCPVersioingCopySuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	// put object
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)

	// get versionID
	objectStat := s.getStat(bucketName, objectName, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)

	// overwrite object
	s.createFile(testFileName, randStr(200), c)
	s.putObject(bucketName, objectName, testFileName, c)

	// copy object
	objectTarget := objectName + "-target"

	//copy object
	cpArgs := []string{CloudURLToString(bucketName, objectName), CloudURLToString(bucketName, objectTarget)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download target object
	// delete versionId
	delete(options, "versionId")
	downFileName := "ossutil_test_" + randStr(5)
	cpArgs = []string{CloudURLToString(bucketName, objectTarget), downFileName}
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check download file
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(testFileName)
	os.RemoveAll(downFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPWithAuthProxy(c *C) {
	if proxyHost == "" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)
	objectName := "ossutil_test_object" + randStr(5)

	// upload object
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	command := "cp"
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &ConfigFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"bigfileThreshold": &thre,
		"proxyHost":        &proxyHost,
		"proxyUser":        &proxyUser,
		"proxyPwd":         &proxyPwd,
	}
	srcUrl := CloudURLToString(bucketName, objectName)
	args := []string{fileName, srcUrl}

	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	os.RemoveAll(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPWithAuthProxyError(c *C) {
	if proxyHost == "" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)
	objectName := "ossutil_test_object" + randStr(5)

	// upload object,proxy-user is empty
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	command := "cp"
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &ConfigFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"bigfileThreshold": &thre,
		"proxyHost":        &proxyHost,
		"proxyUser":        &str,
		"proxyPwd":         &proxyPwd,
	}
	srcUrl := CloudURLToString(bucketName, objectName)
	args := []string{fileName, srcUrl}

	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// upload object,proxy-pwd is empty
	options["proxyUser"] = &proxyUser
	options["proxyPwd"] = &str

	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	os.RemoveAll(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPSetLocalAddrSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	//get local ip
	conn, err := net.Dial("udp", "8.8.8.8:80")
	c.Assert(err, IsNil)
	localAddr := conn.LocalAddr().(*net.UDPAddr)
	localIp := localAddr.IP.String()
	conn.Close()

	// prepare two file
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	cpArgs := []string{testFileName, CloudURLToString(bucketName, objectName)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"localHost":       &localIp,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, objectName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	// remove
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPSetLocalAddrError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	//get local ip
	localIp := "127.0.0.1"

	// prepare file
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	cpArgs := []string{testFileName, CloudURLToString(bucketName, objectName)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"localHost":       &localIp,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	// ResolveIPAddr error
	localIp = "11.11.11.11"
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadSymlinkDir(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	// mk symlink dir
	symlinkDir := "ossutil_test_dir_" + randStr(5)
	err = os.Symlink(dirName, symlinkDir)
	c.Assert(err, IsNil)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testFileName, data, c)

	// begin cp file
	cpArgs := []string{symlinkDir, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download
	delete(options, "recursive")
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(testFileName)
	os.Remove(downFileName)
	os.RemoveAll(dirName)
	os.RemoveAll(symlinkDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadSubSymlinkDir(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// mk symlink dir
	symlinkDir := "ossutil_test_dir_" + randStr(5)
	err = os.Symlink(subDirName, dirName+string(os.PathSeparator)+symlinkDir)
	c.Assert(err, IsNil)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testFileName, data, c)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	enableSymlinkDir := true
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &configFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"recursive":        &recursive,
		"enableSymlinkDir": &enableSymlinkDir,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download symlink dir object
	delete(options, "recursive")
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, symlinkDir+"/"+testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)

	// download dir object
	dwArgs = []string{CloudURLToString(bucketName, subDirName+"/"+testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err = ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(downFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadOnlyCurrentDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// filename
	testDirFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testDirFileName, data, c)

	// subdir filename
	testSubDirFileName := "ossutil_test_file" + randStr(5)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testSubDirFileName, data, c)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	onlyCurrentDir := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"onlyCurrentDir":  &onlyCurrentDir,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download dir object success
	delete(options, "recursive")
	downFileName := "ossutil_test_file" + randStr(5)
	dwArgs := []string{CloudURLToString(bucketName, testDirFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)

	// download sub dir object error
	dwArgs = []string{CloudURLToString(bucketName, subDirName+"/"+testSubDirFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, NotNil)

	os.Remove(downFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDownloadOnlyCurrentDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// filename
	testDirFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testDirFileName, data, c)

	// subdir filename
	testSubDirFileName := "ossutil_test_file" + randStr(5)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testSubDirFileName, data, c)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, dirName)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	onlyCurrentDir := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		//"onlyCurrentDir":  &onlyCurrentDir,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download dir object success
	delete(options, "recursive")
	downFileName := "ossutil_test_file" + randStr(5)
	dwArgs := []string{CloudURLToString(bucketName, dirName+"/"+testDirFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)

	// download sub dir object success
	dwArgs = []string{CloudURLToString(bucketName, dirName+"/"+subDirName+"/"+testSubDirFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	// check content
	fileBody, err = ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)

	// download with onlyCurrentDir
	downDir := dirName + "-down"
	options["recursive"] = &recursive
	options["onlyCurrentDir"] = &onlyCurrentDir

	// download sub dir object success
	dwArgs = []string{CloudURLToString(bucketName, dirName+"/"), downDir}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	// stat dir object success
	_, err = os.Stat(downDir + string(os.PathSeparator) + testDirFileName)
	c.Assert(err, IsNil)

	// stat subdir object error
	_, err = os.Stat(downDir + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + testSubDirFileName)
	c.Assert(err, NotNil)

	os.Remove(downFileName)
	os.RemoveAll(dirName)
	os.RemoveAll(downDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadDisableDirObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	// mk subdir
	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, dirName)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	disableDirObject := true
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &configFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"recursive":        &recursive,
		"disableDirObject": &disableDirObject,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download dir object error
	delete(options, "recursive")
	delete(options, "disableDirObject")

	downFileName := "ossutil_test_file" + randStr(5)
	dwArgs := []string{CloudURLToString(bucketName, dirName+"/"+subDirName+"/"), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, NotNil)

	// upload again without disableDirObject
	options["recursive"] = &recursive
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//down success
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	os.Remove(downFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func MockOssServerHandle(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	go func() {
		ioutil.ReadAll(r.Body)
	}()
	time.Sleep(time.Second * 5)
	w.WriteHeader(500)
	w.Write([]byte(""))
}

func (s *OssutilCommandSuite) TestCPObjectProgressBarNetErrorRetry(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	maxUpSpeed := int64(2) // 2KB/s
	upSecond := 120
	objectContext := randLowStr(int(maxUpSpeed) * upSecond * 1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	mockHost := "127.0.0.1:32915" // mock oss http server
	str := "mock"
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(1)
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	strRetryTimes := "3"
	options := OptionMapType{
		"endpoint":         &mockHost,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &configFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"force":            &bForce,
		"maxupspeed":       &maxUpSpeed,
		"bigfileThreshold": &thre,
		"retryTimes":       &strRetryTimes,
	}

	//start mock http server
	svr := startHttpServer(MockOssServerHandle)

	// calculate time
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	svr.Close()
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadWithDisableAllSymlinkDirFailure(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	dirName := "ossutil_test_dir_" + randStr(5)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	enableSymlinkDir := true
	disableAllSymlink := true

	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &configFile,
		"checkpointDir":     &cpDir,
		"routines":          &routines,
		"recursive":         &recursive,
		"enableSymlinkDir":  &enableSymlinkDir,
		"disableAllSymlink": &disableAllSymlink,
	}

	//--enable-symlink-dir and --disable-all-symlink can't be both exist
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadWithDisableAllSymlinkDirSuccess(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// mk symlink dir
	symlinkDir := "ossutil_test_dir_" + randStr(5)
	err = os.Symlink(subDirName, dirName+string(os.PathSeparator)+symlinkDir)
	c.Assert(err, IsNil)

	// file under subdir
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testFileName, data, c)

	// file under dir
	s.createFile(dirName+string(os.PathSeparator)+testFileName, data, c)

	// symlink file under dir
	testSymlinkFile := testFileName + "-symlink"
	err = os.Symlink(testFileName, dirName+string(os.PathSeparator)+testSymlinkFile)
	c.Assert(err, IsNil)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	disableAllSymlink := true
	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &configFile,
		"checkpointDir":     &cpDir,
		"routines":          &routines,
		"recursive":         &recursive,
		"disableAllSymlink": &disableAllSymlink,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// download symlink dir object failure
	delete(options, "recursive")
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, symlinkDir+"/"+testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, NotNil)

	// download sub dir object success
	dwArgs = []string{CloudURLToString(bucketName, subDirName+"/"+testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//download dir object success
	dwArgs = []string{CloudURLToString(bucketName, testFileName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//download dir symlink failure
	dwArgs = []string{CloudURLToString(bucketName, testSymlinkFile), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, NotNil)

	os.Remove(downFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadSymlinkFileProgressPrecise(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	// file under dir
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testFileName, data, c)

	// symlink file under dir
	testSymlinkFile := testFileName + "-symlink"
	err = os.Symlink(testFileName, dirName+string(os.PathSeparator)+testSymlinkFile)
	c.Assert(err, IsNil)

	// begin cp file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := true
	disableAllSymlink := true
	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &configFile,
		"checkpointDir":     &cpDir,
		"routines":          &routines,
		"recursive":         &recursive,
		"disableAllSymlink": &disableAllSymlink,
	}

	// upload
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check progress is 100%,not over 100%
	snap := copyCommand.monitor.getSnapshot()
	c.Assert((copyCommand.monitor.getPrecent(snap)) == 100, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDownLoadWithoutDisableIgnoreError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// file under dir
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// put object1, standard
	bucketStr := CloudURLToString(bucketName, "object1")
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Standard"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Standard", "")
	c.Assert(err, IsNil)

	// put object2, Archive
	bucketStr = CloudURLToString(bucketName, "object2")
	args = []string{testFileName, bucketStr}
	cmdline = []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Archive"}
	_, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Archive", "")
	c.Assert(err, IsNil)

	// put object3, standard
	bucketStr = CloudURLToString(bucketName, "object3")
	args = []string{testFileName, bucketStr}
	cmdline = []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Standard"}
	_, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Standard", "")
	c.Assert(err, IsNil)

	//down dir name
	dirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	// downloas without disable-ignore-error
	cpArgs := []string{CloudURLToString(bucketName, ""), dirName}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(1)
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
	}

	// download
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check,success download 2 file
	// exist
	filePath := dirName + string(os.PathSeparator) + "object1"
	strObject := s.readFile(filePath, c)
	c.Assert(len(strObject) > 0, Equals, true)
	os.Remove(filePath)

	// not exist
	filePath = dirName + string(os.PathSeparator) + "object2"
	_, err = os.Stat(filePath)
	c.Assert(err, NotNil)

	// exist
	filePath = dirName + string(os.PathSeparator) + "object3"
	strObject = s.readFile(filePath, c)
	c.Assert(len(strObject) > 0, Equals, true)
	os.Remove(filePath)

	os.Remove(testFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDownLoadWithDisableIgnoreError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// file under dir
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// put object1, standard
	bucketStr := CloudURLToString(bucketName, "object1")
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Standard"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Standard", "")
	c.Assert(err, IsNil)

	// put object2, Archive
	bucketStr = CloudURLToString(bucketName, "object2")
	args = []string{testFileName, bucketStr}
	cmdline = []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Archive"}
	_, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Archive", "")
	c.Assert(err, IsNil)

	// put object3, standard
	bucketStr = CloudURLToString(bucketName, "object3")
	args = []string{testFileName, bucketStr}
	cmdline = []string{"ossutil", "cp", testFileName, bucketStr, "-f", "--meta", "X-oss-Storage-Class:Standard"}
	_, err = s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "X-oss-Storage-Class:Standard", "")
	c.Assert(err, IsNil)

	//down dir name
	dirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	// downloas disable-ignore-error
	cpArgs := []string{CloudURLToString(bucketName, ""), dirName}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(1)
	recursive := true
	disableIgnoreError := true
	options := OptionMapType{
		"endpoint":           &str,
		"accessKeyID":        &str,
		"accessKeySecret":    &str,
		"configFile":         &configFile,
		"checkpointDir":      &cpDir,
		"routines":           &routines,
		"recursive":          &recursive,
		"disableIgnoreError": &disableIgnoreError,
	}

	// download
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	// check,success download 1 file
	// exist
	filePath := dirName + string(os.PathSeparator) + "object1"
	strObject := s.readFile(filePath, c)
	c.Assert(len(strObject) > 0, Equals, true)
	os.Remove(filePath)

	// not exist
	filePath = dirName + string(os.PathSeparator) + "object2"
	_, err = os.Stat(filePath)
	c.Assert(err, NotNil)

	// not exist
	filePath = dirName + string(os.PathSeparator) + "object3"
	_, err = os.Stat(filePath)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCpWithTaggingError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// upload with tagging
	cpArgs := []string{testFileName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := false

	// invalid format 1
	tagging := "tagA=A&&tagb=B"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"tagging":         &tagging,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	// invalid format 2
	tagging = "tagA=A&"
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	// invalid format 3
	tagging = "tagA==A"
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	// invalid format 4
	tagging = "tagA=A="
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadFileWithTagging(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// upload with tagging
	cpArgs := []string{testFileName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := false

	tagging := "tagA=A&tagb=B&tagc=C"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"tagging":         &tagging,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	objectStat := s.getStat(bucketName, testFileName, c)
	c.Assert(objectStat["X-Oss-Tagging-Count"], Equals, "3")

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCopyFileWithoutTagging(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// upload with tagging
	cpArgs := []string{testFileName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := false
	tagging := "tagA=A&tagb=B&tagc=C"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
		"tagging":         &tagging,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)
	objectStat := s.getStat(bucketName, testFileName, c)
	c.Assert(objectStat["X-Oss-Tagging-Count"], Equals, "3")

	// then copy file without tagging
	destFileName := testFileName + "dest"
	srcObjectURL := CloudURLToString(bucketName, testFileName)
	destObjectURL := CloudURLToString(bucketName, destFileName)
	cpArgs = []string{srcObjectURL, destObjectURL}
	delete(options, "tagging")

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check dest object, tagging not exist
	objectStat = s.getStat(bucketName, destFileName, c)
	_, ok := objectStat["X-Oss-Tagging-Count"]
	c.Assert(ok, Equals, false)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCopyFileWithTagging(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// upload without tagging
	cpArgs := []string{testFileName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := false

	tagging := "tagA=A&tagb=B&tagc=C"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"recursive":       &recursive,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// then copy file with tagging
	destFileName := testFileName + "dest"
	srcObjectURL := CloudURLToString(bucketName, testFileName)
	destObjectURL := CloudURLToString(bucketName, destFileName)
	cpArgs = []string{srcObjectURL, destObjectURL}

	options["tagging"] = &tagging
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// check dest object, tagging exist
	objectStat := s.getStat(bucketName, destFileName, c)
	c.Assert(objectStat["X-Oss-Tagging-Count"], Equals, "3")
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadMultiFileFileWithTagging(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileLen := int64(2 * 1024 * 1024)
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(int(fileLen))
	s.createFile(testFileName, data, c)

	// multipart upload with tagging
	cpArgs := []string{testFileName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	recursive := false
	threshold := strconv.FormatInt(fileLen/2, 10)
	tagging := "tagA=A&tagb=B&tagc=C"
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &configFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"recursive":        &recursive,
		"tagging":          &tagging,
		"bigfileThreshold": &threshold,
	}

	// upload
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	objectStat := s.getStat(bucketName, testFileName, c)
	c.Assert(objectStat["X-Oss-Tagging-Count"], Equals, "3")
	c.Assert(objectStat["X-Oss-Object-Type"], Equals, "Multipart")

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

// test PutObject with kms sm4
func (s *OssutilCommandSuite) TestPutObjectWithKmsSm4Encryption(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	content := randStr(1024)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "--meta", "x-oss-server-side-encryption:KMS#x-oss-server-side-data-encryption:SM4"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "x-oss-server-side-encryption:KMS#x-oss-server-side-data-encryption:SM4", "")
	c.Assert(err, IsNil)

	// stat
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideEncryption], Equals, "KMS")
	c.Assert(len(objectStat["Etag"]) > 0, Equals, true)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideDataEncryption], Equals, "SM4")
}

// test PutObject with sm4
func (s *OssutilCommandSuite) TestPutObjectWithSm4Encryption(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	content := randStr(1024)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "--meta", "x-oss-server-side-encryption:SM4"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "x-oss-server-side-encryption:SM4", "")
	c.Assert(err, IsNil)

	// stat
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideEncryption], Equals, "SM4")
	c.Assert(len(objectStat["Etag"]) > 0, Equals, true)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideDataEncryption], Equals, "")
}

// test PutObject with KMS
func (s *OssutilCommandSuite) TestPutObjectWithKmsEncryption(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	content := randStr(1024)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "--meta", "x-oss-server-side-encryption:KMS"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "x-oss-server-side-encryption:KMS", "")
	c.Assert(err, IsNil)

	// stat
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideEncryption], Equals, "KMS")
	c.Assert(len(objectStat["Etag"]) > 0, Equals, true)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideDataEncryption], Equals, "")
}

// test PutObject with AES256
func (s *OssutilCommandSuite) TestPutObjectWithAES256(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	content := randStr(1024)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr, "--meta", "x-oss-server-side-encryption:AES256"}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "x-oss-server-side-encryption:AES256", "")
	c.Assert(err, IsNil)

	// stat
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideEncryption], Equals, "AES256")
	c.Assert(len(objectStat["Etag"]) > 0, Equals, true)
	c.Assert(objectStat[oss.HTTPHeaderOssServerSideDataEncryption], Equals, "")
}

func (s *OssutilCommandSuite) TestBatchDownloadSymlinkObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	len := 1024 * 1024
	content := randStr(len)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)

	// create symlink object
	symObject := object + "-link"
	strCmdline := fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), CloudURLToString(bucketName, object))
	err = s.initCreateSymlink(strCmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	// batch download symlink object
	args = []string{CloudURLToString(bucketName, symObject), "." + string(os.PathSeparator)}
	cmdline = []string{"ossutil", "cp", CloudURLToString(bucketName, symObject), "." + string(os.PathSeparator)}
	_, err = s.rawCPWithFilter(args, true, true, false, int64(len/2), CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)

	//check size
	fileInfo, err := os.Stat(symObject)
	c.Assert(err, IsNil)
	c.Assert(fileInfo.Size(), Equals, int64(len))
	c.Assert(copyCommand.monitor.totalSize, Equals, int64(len))

	os.Remove(testFileName)
	os.Remove(symObject)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchDownloadSymlinkObjectWithMultilevelAndEmptyObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	object := randStr(12)
	bucketStr := CloudURLToString(bucketName, object)

	// put object
	testFileName := "ossutil-test-" + randStr(10)
	len := 1024 * 1024
	content := randStr(len)
	s.createFile(testFileName, content, c)

	// upload files
	args := []string{testFileName, bucketStr}
	cmdline := []string{"ossutil", "cp", testFileName, bucketStr}
	_, err := s.rawCPWithFilter(args, false, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)

	// create symlink object
	symObject := object + "-link"
	strCmdline := fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), CloudURLToString(bucketName, object))
	err = s.initCreateSymlink(strCmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	// create symlink to symlink
	symObject2 := object + "-link2"
	strCmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject2), CloudURLToString(bucketName, symObject))
	err = s.initCreateSymlink(strCmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	// batch download symlink object
	args = []string{CloudURLToString(bucketName, symObject2), "." + string(os.PathSeparator)}
	cmdline = []string{"ossutil", "cp", CloudURLToString(bucketName, symObject2), "." + string(os.PathSeparator)}
	_, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)

	_, err = os.Stat(symObject2)
	c.Assert(err, NotNil)

	//delete object
	command := "rm"
	args = []string{CloudURLToString(bucketName, object)}
	testLogger.Print(configFile)
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// download symlink object with deleted object
	args = []string{CloudURLToString(bucketName, symObject), "." + string(os.PathSeparator)}
	cmdline = []string{"ossutil", "cp", CloudURLToString(bucketName, symObject), "." + string(os.PathSeparator)}
	_, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)

	_, err = os.Stat(symObject)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
	os.Remove(testFileName)
}

func (s *OssutilCommandSuite) TestCPObjectWithInputPassword(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// prepare file and object
	objectContext := randLowStr(1024)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	str := ""
	bPassword := true
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"password":        &bPassword,
	}

	fmt.Printf("password\n")

	// calculate time
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderAKmode(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "AK"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
	}

	c.Log(options)
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderEcsmodeWithConfigEcsUrl(c *C) {
	ecsAk := "http://100.100.100.200/latest/meta-data/Ram/security-credentials/" + ecsRoleName
	configStr := "[Credentials]" + "\n" + "language=EN" + "\n" + "endpoint=" + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk

	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	// create bucket
	mode := "EcsRamRole"

	str := ""
	bucketName := bucketNamePrefix + randLowStr(12)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	options := OptionMapType{
		"endpoint":   &str,
		"mode":       &mode,
		"configFile": &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options = OptionMapType{
		"endpoint":      &str,
		"configFile":    &cfile,
		"mode":          &mode,
		"routines":      &routines,
		"checkpointDir": &cpDir,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(cfile)
	os.Remove(downFileName)
	os.Remove(testFileName)

	// rm bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName, "")}
	ok := true
	options = OptionMapType{
		"endpoint":   &str,
		"configFile": &configFile,
		"mode":       &mode,
		"bucket":     &ok,
		"force":      &ok,
		"allType":    &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestCPObjectUnderEcsmodeWithEmptyEcsUrl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "EcsRamRole"
	routines := strconv.Itoa(Routines)

	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderEcsmodeWithRolename(c *C) {
	mode := "EcsRamRole"
	str := ""

	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	options := OptionMapType{
		"endpoint":    &str,
		"mode":        &mode,
		"configFile":  &configFile,
		"ecsRoleName": &ecsRoleName,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options = OptionMapType{
		"endpoint":      &str,
		"configFile":    &configFile,
		"mode":          &mode,
		"routines":      &routines,
		"checkpointDir": &cpDir,
		"ecsRoleName":   &ecsRoleName,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)

	// rm bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName, "")}
	ok := true
	options = OptionMapType{
		"endpoint":    &str,
		"configFile":  &configFile,
		"mode":        &mode,
		"ecsRoleName": &ecsRoleName,
		"bucket":      &ok,
		"force":       &ok,
		"allType":     &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestCPObjectUnderRamRoleArnmodeWithArn(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "RamRoleArn"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"ramRoleArn":      &stsARN,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderRamRoleArnmodeWithConfigArn(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	configStr := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", endpoint, stsAccessID, stsAccessKeySecret)
	configStr = configStr + "ramRoleArn=" + stsARN
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)
	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "RamRoleArn"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &cfile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"ramRoleArn":      &str,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))

	os.Remove(cfile)
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)

}

func (s *OssutilCommandSuite) TestCPObjectUnderRamRoleArnmodeWithEmptyArn(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "RamRoleArn"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"ramRoleArn":      &str,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderRamRoleArnmodeWithTokenTimeout(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "RamRoleArn"
	routines := strconv.Itoa(Routines)
	tokenTimeout := "2000"
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"ramRoleArn":      &stsARN,
		"tokenTimeout":    &tokenTimeout,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderRamRoleArnmodeWithStsRegion(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}
	str := ""
	mode := "RamRoleArn"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &stsAccessID,
		"accessKeySecret": &stsAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"ramRoleArn":      &stsARN,
		"stsRegion":       &stsRegion,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderNomode(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderNomodeWithEmptyAKIdAndEcsUrl(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	configStr := "[Credentials]" + "\n" + "language=EN" + "\n" + "endpoint=oss-cn-shenzhen.aliyuncs.com" + "\n"
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &cfile,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderNomodeUsingEcsRoleAK(c *C) {
	ecsAk := "http://100.100.100.200/latest/meta-data/Ram/security-credentials/" + ecsRoleName
	configStr := "[Credentials]" + "\n" + "language=EN" + "\n" + "endpoint=" + endpoint + "\n"
	configStr = configStr + "[AkService]" + "\n" + "ecsAk=" + ecsAk
	cfile := randStr(10)
	s.createFile(cfile, configStr, c)

	// create bucket
	mode := "EcsRamRole"
	str := ""
	bucketName := bucketNamePrefix + randLowStr(12)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	options := OptionMapType{
		"endpoint":   &str,
		"mode":       &mode,
		"configFile": &cfile,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// prepare file and object
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &cfile,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)

	// rm bucket
	command = "rm"
	args = []string{CloudURLToString(bucketName, "")}
	ok := true
	options = OptionMapType{
		"endpoint":   &str,
		"configFile": &cfile,
		"mode":       &mode,
		"bucket":     &ok,
		"force":      &ok,
		"allType":    &ok,
		"recursive":  &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	s.removeBucket(bucketName, true, c)
	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestCPObjectUnderNomodeWithTimeOut(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	readTimeOut := "1000"
	connectTimeOut := "100"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"readTimeOut":     &readTimeOut,
		"connectTimeOut":  &connectTimeOut,
	}

	c.Log(options)
	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderSTStokenmode(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	client := NewClient(stsAccessID, stsAccessKeySecret, stsARN, "sts_test")

	resp, err := client.AssumeRole(900, "")
	c.Assert(err, IsNil)

	temAccessID := resp.Credentials.AccessKeyId
	temAccessKeySecret := resp.Credentials.AccessKeySecret
	temSTSToken := resp.Credentials.SecurityToken

	c.Assert(temAccessID, Not(Equals), "")
	c.Assert(temAccessKeySecret, Not(Equals), "")
	c.Assert(temSTSToken, Not(Equals), "")

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "StsToken"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &temAccessID,
		"accessKeySecret": &temAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"stsToken":        &temSTSToken,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	// down object
	downFileName := testFileName + ".down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//check content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(data, Equals, string(fileBody))
	os.Remove(downFileName)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectUnderSTSTokenmodeWithEmptySTSToken(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	client := NewClient(stsAccessID, stsAccessKeySecret, stsARN, "sts_test")

	resp, err := client.AssumeRole(900, "")
	c.Assert(err, IsNil)

	temAccessID := resp.Credentials.AccessKeyId
	temAccessKeySecret := resp.Credentials.AccessKeySecret
	temSTSToken := resp.Credentials.SecurityToken

	c.Assert(temAccessID, Not(Equals), "")
	c.Assert(temAccessKeySecret, Not(Equals), "")
	c.Assert(temSTSToken, Not(Equals), "")

	// prepare file and object

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// begin cp file
	object := randLowStr(12)
	cpArgs := []string{testFileName, CloudURLToString(bucketName, object)}

	str := ""
	mode := "StsToken"
	routines := strconv.Itoa(Routines)
	cpDir := CheckpointDir
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &temAccessID,
		"accessKeySecret": &temAccessKeySecret,
		"configFile":      &configFile,
		"mode":            &mode,
		"routines":        &routines,
		"checkpointDir":   &cpDir,
		"stsToken":        &str,
	}

	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCPObjectSkipVerifyCert(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	objectContext := randLowStr(1024 * 10)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"skipVerifyCert":  &str,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//down object
	downFileName := fileName + "-down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCloudBoxCreateAndDeleteBucket(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	objectContext := randLowStr(1024 * 10)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContext, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"skipVerifyCert":  &str,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	//down object
	downFileName := fileName + "-down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	//compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContext, Equals, string(fileBody))

	os.Remove(downFileName)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCpCmdWithTimeSingleObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	objectContent := randLowStr(10)
	fileName := "ossutil_test." + randLowStr(12)
	s.createFile(fileName, objectContent, c)

	object := randLowStr(12)
	cpArgs := []string{fileName, CloudURLToString(bucketName, object)}

	cpDir := CheckpointDir
	maxTime := time.Now().Add(+10 * time.Second).Unix()
	c.Log(maxTime)
	minTime := time.Now().Add(-10 * time.Second).Unix()
	endTime := time.Now().Add(+20 * time.Second).Unix()
	str := ""
	force := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
	}

	_, err := cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)

	downFileName := fileName + "-down"
	dwArgs := []string{CloudURLToString(bucketName, object), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(objectContent, Equals, string(fileBody))

	options[OptionStartTime] = &maxTime
	options[OptionEndTime] = &minTime
	object2 := object + "2"
	cpArgs = []string{fileName, CloudURLToString(bucketName, object2)}
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "start time"), Equals, true)

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	options[OptionStartTime] = &minTime
	options[OptionEndTime] = &maxTime
	object3 := object + "3"
	cpArgs = []string{fileName, CloudURLToString(bucketName, object3)}
	_, err = cm.RunCommand("cp", cpArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(2 * time.Second)

	downFileName3 := fileName + "-down3"
	dwArgs = []string{CloudURLToString(bucketName, object3), downFileName3}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)
	fileBody, err = ioutil.ReadFile(downFileName3)
	c.Assert(err, IsNil)
	c.Assert(objectContent, Equals, string(fileBody))
	c.Log(string(fileBody))
	time.Sleep(2 * time.Second)

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	options[OptionStartTime] = &minTime
	downFileName4 := fileName + "-down4"
	dwArgs = []string{CloudURLToString(bucketName, object3), downFileName4}
	c.Log(dwArgs)
	c.Log(options)
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	time.Sleep(2 * time.Second)

	fileBody, err = ioutil.ReadFile(downFileName4)
	c.Assert(err, IsNil)
	c.Assert(objectContent, Equals, string(fileBody))

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	options[OptionStartTime] = &maxTime
	downFileName5 := fileName + "-down5"
	dwArgs = []string{CloudURLToString(bucketName, object3), downFileName5}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	fileBody, err = ioutil.ReadFile(downFileName5)
	c.Log(string(fileBody))
	c.Assert(err, NotNil)

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	options[OptionEndTime] = &minTime
	downFileName6 := fileName + "-down6"
	dwArgs = []string{CloudURLToString(bucketName, object3), downFileName6}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	fileBody, err = ioutil.ReadFile(downFileName6)
	c.Log(string(fileBody))
	c.Assert(err, NotNil)

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	options[OptionStartTime] = &maxTime
	options[OptionEndTime] = &endTime
	downFileName7 := fileName + "-down7"
	dwArgs = []string{CloudURLToString(bucketName, object3), downFileName7}
	_, err = cm.RunCommand("cp", dwArgs, options)
	c.Assert(err, IsNil)

	fileBody, err = ioutil.ReadFile(downFileName7)
	c.Log(string(fileBody))
	c.Assert(err, NotNil)

	s.clearObjects(bucketName, "", c)

	// cp bucket to other bucket
	optionsCopy := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
	}
	maxTime = time.Now().Add(+10 * time.Second).Unix()
	minTime = time.Now().Add(-10 * time.Second).Unix()
	endTime = time.Now().Add(+20 * time.Second).Unix()

	s.putObject(bucketName, object, fileName, c)
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName := fileName + "-copy"
	s.getObject(bucketNameExist, object, downCopyName, c)
	contentCopy := s.readFile(downCopyName, c)
	c.Assert(objectContent, Equals, contentCopy)

	optionsCopy[OptionStartTime] = &maxTime
	optionsCopy[OptionEndTime] = &minTime
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object2)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "start time"), Equals, true)

	delete(options, OptionStartTime)
	delete(options, OptionEndTime)
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object3)}
	optionsCopy[OptionStartTime] = &minTime
	optionsCopy[OptionEndTime] = &maxTime
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName3 := fileName + "-copy3"
	s.getObject(bucketNameExist, object3, downCopyName3, c)
	contentCopy3 := s.readFile(downCopyName, c)
	c.Assert(objectContent, Equals, contentCopy3)

	delete(optionsCopy, OptionStartTime)
	delete(optionsCopy, OptionEndTime)
	optionsCopy[OptionStartTime] = &minTime
	object4 := object + "4"
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object4)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName4 := fileName + "-copy4"
	s.getObject(bucketNameExist, object4, downCopyName4, c)
	contentCopy4 := s.readFile(downCopyName, c)
	c.Assert(objectContent, Equals, contentCopy4)

	delete(optionsCopy, OptionStartTime)
	delete(optionsCopy, OptionEndTime)
	optionsCopy[OptionEndTime] = &maxTime
	object5 := object + "5"
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object5)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName5 := fileName + "-copy5"
	s.getObject(bucketNameExist, object5, downCopyName5, c)
	contentCopy5 := s.readFile(downCopyName5, c)
	c.Assert(objectContent, Equals, contentCopy5)

	delete(optionsCopy, OptionStartTime)
	delete(optionsCopy, OptionEndTime)
	optionsCopy[OptionEndTime] = &minTime
	object6 := object + "6"
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object6)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName6 := fileName + "-copy6"
	args := []string{CloudURLToString(bucketNameExist, object6), downCopyName6}
	_, err = s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "The specified key does not exist"), Equals, true)

	delete(optionsCopy, OptionStartTime)
	delete(optionsCopy, OptionEndTime)
	optionsCopy[OptionStartTime] = &maxTime
	optionsCopy[OptionEndTime] = &endTime

	object7 := object + "7"
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object7)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName7 := fileName + "-copy7"
	args = []string{CloudURLToString(bucketNameExist, object7), downCopyName7}
	_, err = s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "The specified key does not exist"), Equals, true)

	delete(optionsCopy, OptionStartTime)
	delete(optionsCopy, OptionEndTime)
	optionsCopy[OptionStartTime] = &maxTime
	object8 := object + "8"
	cpArgs = []string{CloudURLToString(bucketName, object), CloudURLToString(bucketNameExist, object8)}
	_, err = cm.RunCommand("cp", cpArgs, optionsCopy)
	c.Assert(err, IsNil)

	downCopyName8 := fileName + "-copy8"
	args = []string{CloudURLToString(bucketNameExist, object8), downCopyName8}
	_, err = s.rawCPWithArgs(args, false, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "The specified key does not exist"), Equals, true)

	s.clearObjects(bucketName, "", c)
	os.Remove(downFileName)
	os.Remove(fileName)
	os.Remove(downFileName3)
	os.Remove(downFileName4)
	os.Remove(downFileName5)
	os.Remove(downFileName6)
	os.Remove(downFileName7)
	os.Remove(downCopyName)
	os.Remove(downCopyName3)
	os.Remove(downCopyName4)
	os.Remove(downCopyName5)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCpCmdWithTimeDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	//objectContext := randLowStr(10)
	dir := "testdir-inc1" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	cpArgs := []string{dir + "/", CloudURLToString(bucketName, dir+"/")}
	cpDir := CheckpointDir
	maxTime := time.Now().Add(+10 * time.Second).Unix()
	c.Log(maxTime)
	minTime := time.Now().Add(-10 * time.Second).Unix()

	endTime := time.Now().Add(+20 * time.Second).Unix()
	force := true
	recursion := true
	routines := strconv.Itoa(Routines)
	str := ""
	optionsDir := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
		OptionRecursion:   &recursion,
	}

	_, err := cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	downDir := "cp-down-dir" + randLowStr(5)
	dwArgs := []string{CloudURLToString(bucketName, dir+"/"), downDir}
	_, err = cm.RunCommand("cp", dwArgs, optionsDir)
	c.Assert(err, IsNil)

	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	objectsExist := s.listObjects(bucketNameExist, dir, "ls -", c)
	s.clearObjects(bucketNameExist, "", c)
	var count, count1, count2 int
	err = filepath.Walk(downDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		fileInfo, _ := os.Stat(dir + "/" + filename)
		if fileInfo.Size() > 0 {
			count1++
		}
	}
	for _, object := range objectsExist {
		lastChar := object[len(object)-1:]
		if lastChar != "/" {
			count2++
		}
	}
	c.Assert(count, Equals, count2)
	c.Assert(count, Equals, count1)

	s.clearObjects(bucketName, "", c)
	os.RemoveAll(downDir)

	optionsDir[OptionStartTime] = &maxTime
	optionsDir[OptionEndTime] = &minTime
	cpArgs = []string{dir + "/", CloudURLToString(bucketName, dir)}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "start time"), Equals, true)

	delete(optionsDir, OptionStartTime)
	delete(optionsDir, OptionEndTime)
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)
	optionsDir[OptionStartTime] = &maxTime
	optionsDir[OptionEndTime] = &minTime
	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "start time"), Equals, true)

	delete(optionsDir, OptionStartTime)
	delete(optionsDir, OptionEndTime)
	cpArgs = []string{dir + "/", CloudURLToString(bucketName, dir)}
	optionsDir[OptionStartTime] = &minTime
	optionsDir[OptionEndTime] = &maxTime
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	objectsExist = s.listLimitedMarker(bucketNameExist, dir+"/", "ls ", -1, "", "", c)
	s.clearObjects(bucketNameExist, "", c)
	testLogger.Println(objectsExist)
	downDir3 := downDir + "3"
	dwArgs = []string{CloudURLToString(bucketName, dir+"/"), downDir3}
	_, err = cm.RunCommand("cp", dwArgs, optionsDir)
	c.Assert(err, IsNil)

	count = 0
	count1 = 0
	count2 = 0
	err = filepath.Walk(downDir3, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		fileInfo, _ := os.Stat(dir + "/" + filename)
		end := time.Unix(maxTime, 0)
		start := time.Unix(minTime, 0)
		if fileInfo.ModTime().Before(end) && fileInfo.ModTime().After(start) {
			count1++
		}
	}
	for _, object := range objectsExist {
		lastChar := object[len(object)-1:]
		if lastChar != "/" {
			count2++
		}
	}
	c.Assert(count, Equals, count2)
	c.Assert(count, Equals, count1)
	s.clearObjects(bucketName, "", c)
	os.RemoveAll(downDir3)

	cpArgs = []string{dir + "/", CloudURLToString(bucketName, dir)}
	delete(optionsDir, OptionStartTime)
	delete(optionsDir, OptionEndTime)
	optionsDir[OptionEndTime] = &minTime
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	objectsExist = s.listLimitedMarker(bucketNameExist, "", "ls ", -1, "", "", c)
	s.clearObjects(bucketNameExist, "", c)

	downDir4 := downDir + "4"
	dwArgs = []string{CloudURLToString(bucketName, dir+"/"), downDir4}
	_, err = cm.RunCommand("cp", dwArgs, optionsDir)
	c.Assert(err, IsNil)

	count = 0
	count1 = 0
	count2 = 0
	err = filepath.Walk(downDir4, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		fileInfo, _ := os.Stat(dir + "/" + filename)
		t := time.Unix(minTime, 0)
		if !fileInfo.ModTime().After(t) {
			count1++
		}
	}
	for _, object := range objectsExist {
		lastChar := object[len(object)-1:]
		if lastChar != "/" {
			count2++
		}
	}
	c.Assert(count, Equals, count2)
	c.Assert(count, Equals, count1)
	c.Assert(count, Equals, len(objectsExist))
	s.clearObjects(bucketName, "", c)
	os.RemoveAll(downDir4)

	cpArgs = []string{dir + "/", CloudURLToString(bucketName, dir)}
	delete(optionsDir, OptionStartTime)
	delete(optionsDir, OptionEndTime)
	optionsDir[OptionStartTime] = &maxTime
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	objectsExist = s.listLimitedMarker(bucketNameExist, "", "ls ", -1, "", "", c)
	s.clearObjects(bucketNameExist, "", c)

	downDir5 := downDir + "5"
	dwArgs = []string{CloudURLToString(bucketName, dir+"/"), downDir5}
	_, err = cm.RunCommand("cp", dwArgs, optionsDir)
	c.Assert(err, IsNil)

	count = 0
	count1 = 0
	count2 = 0
	err = filepath.Walk(downDir5, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		fileInfo, _ := os.Stat(dir + "/" + filename)
		t := time.Unix(maxTime, 0)
		if !fileInfo.ModTime().Before(t) {
			count1++
		}
	}
	for _, object := range objectsExist {
		lastChar := object[len(object)-1:]
		if lastChar != "/" {
			count2++
		}
	}
	c.Assert(count, Equals, count2)
	c.Assert(count, Equals, count1)
	os.RemoveAll(downDir5)
	s.clearObjects(bucketName, "", c)

	delete(optionsDir, OptionStartTime)
	delete(optionsDir, OptionEndTime)
	optionsDir[OptionStartTime] = &maxTime
	optionsDir[OptionEndTime] = &maxTime
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	cpArgs = []string{CloudURLToString(bucketName, dir+"/"), CloudURLToString(bucketNameExist, dir+"/")}
	_, err = cm.RunCommand("cp", cpArgs, optionsDir)
	c.Assert(err, IsNil)

	objectsExist = s.listLimitedMarker(bucketNameExist, "", "ls ", -1, "", "", c)
	s.clearObjects(bucketNameExist, "", c)

	downDir6 := downDir + "6"
	dwArgs = []string{CloudURLToString(bucketName, dir+"/"), downDir6}
	_, err = cm.RunCommand("cp", dwArgs, optionsDir)
	c.Assert(err, IsNil)

	count = 0
	count1 = 0
	count2 = 0
	err = filepath.Walk(downDir6, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			count++
		}
		return nil
	})
	c.Assert(err, IsNil)

	for _, filename := range filenames {
		fileInfo, _ := os.Stat(dir + "/" + filename)
		end := time.Unix(endTime, 0)
		start := time.Unix(maxTime, 0)
		if fileInfo.ModTime().Before(end) && fileInfo.ModTime().After(start) {
			count1++
		}
	}
	for _, object := range objectsExist {
		lastChar := object[len(object)-1:]
		if lastChar != "/" {
			count2++
		}
	}
	c.Assert(count, Equals, count2)
	c.Assert(count, Equals, count1)
	os.RemoveAll(downDir5)
	s.clearObjects(bucketName, "", c)

	s.removeBucket(bucketName, true, c)
}

// Test: --start-time & --end-time
func (s *OssutilCommandSuite) TestBatchCPFileToObjectWithStartTimeAndEndTime(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-time" + randLowStr(5)
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	var fileinfos []os.FileInfo
	num := 5
	for i := 0; i < num; i++ {
		content := "hello world"
		filename := fmt.Sprintf("%s%sfilename-%d", dir, string(os.PathSeparator), i)
		s.createFile(filename, content, c)
		info, err := os.Lstat(filename)
		c.Assert(err, IsNil)
		fileinfos = append(fileinfos, info)
		time.Sleep(time.Second * 2)
	}

	//cp 1,2,3,4
	// e.g., ossutil cp testdir-inc/ oss://tempb4 -rf --start-time fileinfos[1].ModTime().Unix()
	// upload files
	args := []string{dir, bucketStr}
	cpDir := CheckpointDir
	force := true
	recursion := true
	routines := 1
	str := ""
	optionsDir := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
		OptionRecursion:   &recursion,
	}
	startTime := fileinfos[1].ModTime().Unix()
	optionsDir[OptionStartTime] = &startTime
	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)

	objects := s.listObjects(bucketName, "", "ls -s", c)
	c.Assert(len(objects), Equals, num-1)
	for i := 1; i < num; i++ {
		c.Assert(objects[i-1], Equals, fileinfos[i].Name())
	}

	//cp 2,3
	s.clearObjects(bucketName, "", c)
	startTime = fileinfos[2].ModTime().Unix()
	endTime := fileinfos[3].ModTime().Unix()
	optionsDir[OptionStartTime] = &startTime
	optionsDir[OptionEndTime] = &endTime
	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)

	objects = s.listObjects(bucketName, "", "ls -s", c)
	c.Assert(len(objects), Equals, 2)
	for i := 2; i <= 3; i++ {
		c.Assert(objects[i-2], Equals, fileinfos[i].Name())
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

var timeFormats = []string{
	"2006-01-02 15:04:05",
}

// parse the date as time in various date formats
func parseTimeDates(date string) (t time.Time, err error) {
	var instant time.Time
	for _, timeFormat := range timeFormats {
		instant, err = time.ParseInLocation(timeFormat, date[:len(timeFormat)], time.Local)
		if err == nil {
			return instant, nil
		}
	}
	return t, err
}

// Test: --start-time & --end-time
func (s *OssutilCommandSuite) TestBatchCPObjectToFiletWithStartTimeAndEndTime(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "testdir-time" + randLowStr(5)
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	content := "hello world"
	filename := fmt.Sprintf("%s%stmp.txt", dir, string(os.PathSeparator))
	s.createFile(filename, content, c)

	num := 5
	for i := 0; i < num; i++ {
		key := fmt.Sprintf("filename-%d", i)
		s.putObject(bucketName, key, filename, c)
		time.Sleep(time.Second * 2)
	}
	objects := s.listObjects(bucketName, "", "ls -s", c)
	c.Assert(len(objects), Equals, num)
	os.Remove(filename)

	//cp 1,2,3,4
	// e.g., ossutil cp oss://tempb4  testdir-inc/ -rf --start-time objects[1].ModTime().Unix()
	// upload files
	args := []string{bucketStr, dir}
	cpDir := CheckpointDir
	force := true
	recursion := true
	routines := 1
	str := ""
	optionsDir := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
		OptionRecursion:   &recursion,
	}

	stat := s.getStat(bucketName, objects[1], c)
	t, err := parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)
	startTime := t.Unix()
	optionsDir[OptionStartTime] = &startTime

	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)
	files, err := s.getFileList(dir)
	c.Assert(len(files), Equals, num-1)
	for i := 1; i < num; i++ {
		c.Assert(files[i-1], Equals, objects[i])
	}

	//2,3
	os.RemoveAll(dir)
	err = os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	stat = s.getStat(bucketName, objects[2], c)
	t, err = parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)
	startTime = t.Unix()

	stat = s.getStat(bucketName, objects[3], c)
	t, err = parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)
	endTime := t.Unix()

	optionsDir[OptionStartTime] = &startTime
	optionsDir[OptionEndTime] = &endTime
	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)
	files, err = s.getFileList(dir)
	c.Assert(len(files), Equals, 2)
	for i := 2; i <= 3; i++ {
		c.Assert(files[i-2], Equals, objects[i])
	}

	// cleanup
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

// Test: --start-time & --end-time
func (s *OssutilCommandSuite) TestBatchCPObjectToObjectWithStartTimeAndEndTime(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	bucketName2 := bucketName + "-cp"
	s.putBucket(bucketName, c)
	s.putBucket(bucketName2, c)
	bucketStr := CloudURLToString(bucketName, "")
	bucket2Str := CloudURLToString(bucketName2, "")

	dir := "testdir-time" + randLowStr(5)
	err := os.MkdirAll(dir, 0755)
	c.Assert(err, IsNil)

	content := "hello world"
	filename := fmt.Sprintf("%s%stmp.txt", dir, string(os.PathSeparator))
	s.createFile(filename, content, c)

	num := 5
	for i := 0; i < num; i++ {
		key := fmt.Sprintf("filename-%d", i)
		s.putObject(bucketName, key, filename, c)
		time.Sleep(time.Second * 2)
	}
	objects := s.listObjects(bucketName, "", "ls -s", c)
	c.Assert(len(objects), Equals, num)
	os.RemoveAll(dir)

	//cp 1,2,3,4
	// e.g., ossutil cp oss://tempb4  testdir-inc/ -rf --start-time objects[1].ModTime().Unix()
	// upload files
	args := []string{bucketStr, bucket2Str}
	cpDir := CheckpointDir
	force := true
	recursion := true
	routines := 1
	str := ""
	optionsDir := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		OptionForce:       &force,
		"routines":        &routines,
		OptionRecursion:   &recursion,
	}

	stat := s.getStat(bucketName, objects[1], c)
	t, err := parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)

	startTime := t.Unix()
	optionsDir[OptionStartTime] = &startTime
	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)
	objects2 := s.listObjects(bucketName2, "", "ls -s", c)
	c.Assert(len(objects2), Equals, num-1)
	for i := 1; i < num; i++ {
		c.Assert(objects2[i-1], Equals, objects[i])
	}

	//2,3
	s.clearObjects(bucketName2, "", c)
	stat = s.getStat(bucketName, objects[2], c)
	t, err = parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)
	startTime = t.Unix()

	stat = s.getStat(bucketName, objects[3], c)
	t, err = parseTimeDates((stat[StatLastModified]))
	c.Assert(err, IsNil)
	endTime := t.Unix()

	optionsDir[OptionStartTime] = &startTime
	optionsDir[OptionEndTime] = &endTime
	_, err = cm.RunCommand("cp", args, optionsDir)
	c.Assert(err, IsNil)
	objects2 = s.listObjects(bucketName2, "", "ls -s", c)
	c.Assert(len(objects2), Equals, 2)
	for i := 2; i <= 3; i++ {
		c.Assert(objects2[i-2], Equals, objects[i])
	}

	// cleanup
	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketName2, true, c)
}
