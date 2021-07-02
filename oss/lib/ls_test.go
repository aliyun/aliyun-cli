package lib

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

// test list buckets
func (s *OssutilCommandSuite) TestListLoadConfig(c *C) {
	command := "ls"
	var args []string
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	args = []string{"oss://"}
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestListNotExistConfigFile(c *C) {
	command := "ls"
	var args []string
	str := ""
	cfile := "notexistfile"
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestListErrConfigFile(c *C) {
	cfile := randStr(10)
	s.createFile(cfile, content, c)

	command := "ls"
	var args []string
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestListConfigFile(c *C) {
	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n", endpoint, accessKeyID, accessKeySecret)
	s.createFile(cfile, data, c)

	command := "ls"
	var args []string
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestListWithBucketEndpoint(c *C) {
	bucketName := bucketNameExist

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s", "abc", accessKeyID, accessKeySecret, bucketName, endpoint)
	s.createFile(cfile, data, c)

	command := "ls"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)
}

func (s *OssutilCommandSuite) TestListWithBucketCname(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s\n[Bucket-Cname]\n%s=%s", "abc", accessKeyID, accessKeySecret, bucketName, "abc", bucketName, bucketName+"."+endpoint)
	s.createFile(cfile, data, c)

	command := "ls"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

// list objects with not exist bucket
func (s *OssutilCommandSuite) TestListObjectsBucketNotExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	command := "ls"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

// list objects
func (s *OssutilCommandSuite) TestListObjects(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put objects
	num := 3
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("lstest:#%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	object := "another_object"
	s.putObject(bucketName, object, uploadFileName, c)

	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat[StatACL], Equals, "default")
	c.Assert(len(objectStat["Etag"]), Equals, 32)
	c.Assert(objectStat["Last-Modified"] != "", Equals, true)
	c.Assert(objectStat[StatOwner] != "", Equals, true)

	//put directories
	num1 := 2
	for i := 0; i < num1; i++ {
		object := fmt.Sprintf("lstest:#%d/", i)
		s.putObject(bucketName, object, uploadFileName, c)

		object = fmt.Sprintf("lstest:#%d/%d/", i, i)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	// "ls oss://bucket -s"
	//objects := s.listObjects(bucketName, "", true, false, false, false, c)
	//c.Assert(len(objects), Equals, num + 2*num1 + 1)

	// "ls oss://bucket/prefix -s"
	objects := s.listObjects(bucketName, "lstest:", "ls -s", c)
	c.Assert(len(objects), Equals, num+2*num1)

	// "ls oss://bucket/prefix"
	objects = s.listObjects(bucketName, "lstest:#", "ls - ", c)
	c.Assert(len(objects), Equals, num+2*num1)

	// "ls oss://bucket/prefix -d"
	objects = s.listObjects(bucketName, "lstest:#", "ls -d", c)
	c.Assert(len(objects), Equals, num+num1)

	objects = s.listObjects(bucketName, "lstest:#1/", "ls -d", c)
	c.Assert(len(objects), Equals, 2)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListObjectsWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket
	objectName := randStr(10)

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, objectName)}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	args = []string{CloudURLToString(bucketName, objectName)}
	showElapse, err = s.rawList(args, "ls - ", OptionPair{Key: "payer", Value: "requester"}, OptionPair{Key: "endpoint", Value: payerBucketEndPoint})
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestListMultipartUploadsWithPayer(c *C) {
	bucketName := payerBucket

	// list buckets with -m
	args := []string{CloudURLToString(bucketName, "")}
	showElapse, err := s.rawList(args, "ls -m", OptionPair{Key: "payer", Value: "requester"}, OptionPair{Key: "endpoint", Value: payerBucketEndPoint})
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

func (s *OssutilCommandSuite) TestListObjectWithPayerInvalidPayer(c *C) {
	bucketName := payerBucket
	invalidPayer := randStr(10)

	// list buckets with -m
	args := []string{CloudURLToString(bucketName, "")}
	showElapse, err := s.rawList(args, "ls - ", OptionPair{Key: "payer", Value: invalidPayer})
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestErrList(c *C) {
	showElapse, err := s.rawList([]string{"../"}, "ls -s")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// not exist bucket
	bucketName := bucketNamePrefix + randLowStr(10)
	showElapse, err = s.rawList([]string{CloudURLToString(bucketName, "")}, "ls -d")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// list buckets with -d
	showElapse, err = s.rawList([]string{"oss://"}, "ls -d")
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestListIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "ls"
	str := ""
	args := []string{"oss://"}
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListBucketIDKey(c *C) {
	bucketName := bucketNameExist

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "ls"
	str := ""
	args := []string{CloudURLToString(bucketName, "")}
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)
	cfile = randStr(20)

	// list without config file
	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)
}

// list multipart
func (s *OssutilCommandSuite) TestListMultipartUploads(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	// "rm -arf oss://bucket/"
	command := "rm"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	ok := true
	limitedNum := strconv.FormatInt(-1, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &ok,
		"force":           &ok,
		"allType":         &ok,
		"limitedNum":      &limitedNum,
	}
	cm.RunCommand(command, args, options)

	object := "TestMultipartObjectLs"
	s.putObject(bucketName, object, uploadFileName, c)

	// list object
	objects := s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	bucket, err := copyCommand.command.ossBucket(bucketName)

	lmr_origin, e := bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)

	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}

	lmr, e := bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, 20+len(lmr_origin.Uploads))

	// list multipart: ls oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	// list multipart: ls -m oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls -m", c)
	c.Assert(len(objects), Equals, 20)

	// list all type object: ls -a oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls -a", c)
	c.Assert(len(objects), Equals, 21)

	// list multipart: ls -am oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls -am", c)
	c.Assert(len(objects), Equals, 21)

	// list multipart: ls -ms oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls -ms", c)
	c.Assert(len(objects), Equals, 20)

	// list multipart: ls -as oss://bucket/object
	objects = s.listObjects(bucketName, object, "ls -as", c)
	c.Assert(len(objects), Equals, 21)

	lmr, e = bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, 20+len(lmr_origin.Uploads))

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListMultipartUploadsError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s", endpoint, "abc", accessKeySecret)
	s.createFile(cfile, data, c)

	command := "ls"
	var args []string
	str := ""
	limitedNum := strconv.FormatInt(-1, 10)
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"limitedNum":      &limitedNum,
		"multipart":       &ok,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListLimitedMarker(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	bucketName1 := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName1, c)

	// list bucket
	buckets := s.listLimitedMarker("", "", "ls ", -1, "", "", c)
	c.Assert(FindPos(bucketName, buckets) != -1, Equals, true)

	// list bucket
	buckets = s.listLimitedMarker("", "", "ls ", 0, "", "", c)
	c.Assert(len(buckets), Equals, 0)

	// list bucket
	buckets = s.listLimitedMarker("", "", "ls ", 1, "", "", c)
	c.Assert(len(buckets), Equals, 1)

	buckets = s.listLimitedMarker("", "", "ls ", -1, "t", "", c)
	c.Assert(FindPos(bucketName, buckets), Equals, -1)

	buckets = s.listLimitedMarker("", "", "ls ", -1, "", "t", c)
	c.Assert(FindPos(bucketName, buckets) != -1, Equals, true)

	objectPrefix := "中文"
	for i := 0; i < 5; i++ {
		s.putObject(bucketName, fmt.Sprintf("%s%d", objectPrefix, i), uploadFileName, c)
	}

	bucket, err := listCommand.command.ossBucket(bucketName)

	for i := 0; i < 5; i++ {
		_, err = bucket.InitiateMultipartUpload(fmt.Sprintf("%s%d", objectPrefix, 0))
		c.Assert(err, IsNil)
	}

	lmr, err := bucket.ListMultipartUploads(oss.Prefix(fmt.Sprintf("%s%d", objectPrefix, 0)))
	c.Assert(err, IsNil)
	uploadIDs := []string{}
	for _, uploadID := range lmr.Uploads {
		uploadIDs = append(uploadIDs, uploadID.UploadID)
	}
	c.Assert(len(uploadIDs), Equals, 5)

	sort.Strings(uploadIDs)

	// list object with limitedNum
	objects := s.listLimitedMarker(bucketName, "", "ls ", -1, "", "", c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -a ", -1, "", "", c)
	c.Assert(len(objects), Equals, 10)

	objects = s.listLimitedMarker(bucketName, "", "ls -m ", -1, "", "", c)
	c.Assert(len(objects), Equals, 5)

	// normal list
	objects = s.listLimitedMarker(bucketName, "", "ls ", 6, "", "", c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -d", 6, "", "", c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls ", 2, "", "", c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls -d", 2, "", "", c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls ", 0, "", "", c)
	c.Assert(len(objects), Equals, 0)

	objects = s.listLimitedMarker(bucketName, "", "ls ", 2, fmt.Sprintf("%s%d", objectPrefix, 1), "", c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls ", 2, fmt.Sprintf("%s%d", objectPrefix, 3), "", c)
	c.Assert(len(objects), Equals, 1)

	objects = s.listLimitedMarker(bucketName, "", "ls ", 2, fmt.Sprintf("%s%d", objectPrefix, 3), "t", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, fmt.Sprintf("%s%d", objectPrefix, 4))

	// list -m
	objects = s.listLimitedMarker(bucketName, "", "ls -m", 6, "", "", c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 2, "", "", c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 2, "", uploadIDs[0], c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls -md", 2, "", uploadIDs[0], c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 10, "", uploadIDs[1], c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -m --encoding-type url", 10, "", url.QueryEscape(uploadIDs[1]), c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 10, objectPrefix, uploadIDs[1], c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 10, fmt.Sprintf("%s%d", objectPrefix, 3), uploadIDs[1], c)
	c.Assert(len(objects), Equals, 0)

	objects = s.listLimitedMarker(bucketName, "", "ls -m", 2, objectPrefix, uploadIDs[2], c)
	c.Assert(len(objects), Equals, 2)

	// list -a
	objects = s.listLimitedMarker(bucketName, "", "ls -a", 6, "", "", c)
	c.Assert(len(objects), Equals, 6)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 11, "", "", c)
	c.Assert(len(objects), Equals, 10)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 20, fmt.Sprintf("%s%d", objectPrefix, 0), uploadIDs[2], c)
	c.Assert(len(objects), Equals, 6)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 5, fmt.Sprintf("%s%d", objectPrefix, 0), uploadIDs[2], c)
	c.Assert(len(objects), Equals, 5)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 20, fmt.Sprintf("%s%d", objectPrefix, 3), "", c)
	c.Assert(len(objects), Equals, 1)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 20, url.QueryEscape(fmt.Sprintf("%s%d", objectPrefix, 3)), "", c)
	c.Assert(len(objects), Equals, 10)

	objects = s.listLimitedMarker(bucketName, "", "ls -a --encoding-type url", 20, url.QueryEscape(fmt.Sprintf("%s%d", objectPrefix, 3)), "", c)
	c.Assert(len(objects), Equals, 1)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 4, fmt.Sprintf("%s%d", objectPrefix, 0), uploadIDs[0], c)
	c.Assert(len(objects), Equals, 4)

	objects = s.listLimitedMarker(bucketName, "", "ls -a --encoding-type url", 4, url.QueryEscape(fmt.Sprintf("%s%d", objectPrefix, 0)), url.QueryEscape(uploadIDs[0]), c)
	c.Assert(len(objects), Equals, 4)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", -1, url.QueryEscape(fmt.Sprintf("%s%d", objectPrefix, 0)), url.QueryEscape(uploadIDs[0]), c)
	c.Assert(len(objects), Equals, 10)

	objects = s.listLimitedMarker(bucketName, "", "ls -a --encoding-type url", -1, url.QueryEscape(fmt.Sprintf("%s%d", objectPrefix, 0)), url.QueryEscape(uploadIDs[0]), c)
	c.Assert(len(objects), Equals, 8)

	objects = s.listLimitedMarker(bucketName, "", "ls -a", 20, "中文t", uploadIDs[0], c)
	c.Assert(len(objects), Equals, 0)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketName1, true, c)
}

func (s *OssutilCommandSuite) TestListURLEncoding(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "^M特殊字符 加上空格 test"
	s.putObject(bucketName, object, uploadFileName, c)

	urlObject := url.QueryEscape(object)
	c.Assert(object != urlObject, Equals, true)

	// list object
	objects := s.listLimitedMarker(bucketName, "", "ls ", -1, "", "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	objects = s.listLimitedMarker(bucketName, urlObject, "ls ", -1, "", "", c)
	c.Assert(len(objects), Equals, 0)

	objects = s.listLimitedMarker(bucketName, urlObject, "ls --encoding-type url -as", -1, "", "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	// remove object
	_, err := s.removeWrapper("rm -f", bucketName, urlObject, c)
	c.Assert(err, IsNil)

	objects = s.listLimitedMarker(bucketName, urlObject, "ls --encoding-type url -s", -1, "", "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	// remove object
	_, err = s.removeWrapper("rm --encoding-type url -f", bucketName, urlObject, c)
	c.Assert(err, IsNil)

	objects = s.listLimitedMarker(bucketName, urlObject, "ls --encoding-type url -s", -1, "", "", c)
	c.Assert(len(objects), Equals, 0)

	s.putObject(bucketName, object, uploadFileName, c)

	// test url encode marker
	object1 := "^M特殊字符 加上空格 test1"
	s.putObject(bucketName, object1, uploadFileName, c)

	urlObject1 := url.QueryEscape(object1)
	c.Assert(object1 != urlObject1, Equals, true)

	objects = s.listLimitedMarker(bucketName, "", "ls --encoding-type url -as", -1, "", "", c)
	c.Assert(len(objects), Equals, 2)

	objects = s.listLimitedMarker(bucketName, "", "ls --encoding-type url -as", -1, urlObject, "", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object1)

	_, err = s.rawListLimitedMarker([]string{"oss%3a%2f%2f"}, "ls --encoding-type url", -1, "", "")
	c.Assert(err, NotNil)

	bucketName1 := url.QueryEscape("中文")
	_, err = s.rawListLimitedMarker([]string{"oss://" + bucketName1}, "ls --encoding-type url", -1, "", "")
	c.Assert(err, NotNil)

	_, err = s.rawListLimitedMarker([]string{"oss://" + bucketName}, "ls --encoding-type url", -1, "", "")
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListObjectfilterInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "ossutil-test-dir-" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// ls files
	// e.g., ossutil ls oss://tempb4/ --include "*.jpg" --include "*.txt"
	cmdline = []string{"ossutil", "ls", bucketStr, "--include", "*.jpg", "--include", "*.txt"}
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	files := filterStrsWithInclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	files = filterStrsWithInclude(filenames, "*.txt")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	files = filterStrsWithInclude(filenames, "*.rtf")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListObjectfilterExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	dir := "ossutil-test-dir-" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// ls files
	cmdline = []string{"ossutil", "ls", bucketStr, "--exclude", "*.jpg"}
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	files := filterStrsWithInclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	files = filterStrsWithInclude(filenames, "*.txt")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	files = filterStrsWithInclude(filenames, "*.rtf")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListDirectoryfilterInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	// directory1
	dir1 := "ossutil-test-dir-" + randLowStr(5)
	subdir1 := "dir1"
	contents1 := map[string]string{}
	filenames1 := s.createTestFiles(dir1, subdir1, c, contents1)

	// directory2
	dir2 := "ossutil-test-dir-" + randLowStr(5)
	subdir2 := "dir2"
	contents2 := map[string]string{}
	filenames2 := s.createTestFiles(dir2, subdir2, c, contents2)

	// upload directory1
	args := []string{dir1, bucketStr + "/" + dir1}
	cmdline := []string{"ossutil", "cp", dir1, bucketStr + "/" + dir1, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// upload directory2
	args = []string{dir2, bucketStr + "/" + dir2}
	cmdline = []string{"ossutil", "cp", dir2, bucketStr + "/" + dir2, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// ls files
	// e.g., ossutil ls oss://tempb4/ --include "*.jpg" --include dir1
	cmdline = []string{"ossutil", "ls", bucketStr, "--include"}
	strFilter := "*" + dir1
	cmdline = append(cmdline, strFilter)

	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	bDirectory := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"directory":       &bDirectory,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), dir1), Equals, true)
	for _, filename := range filenames1 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	c.Assert(strings.Contains(string(fileBody), dir2), Equals, false)
	for _, filename := range filenames2 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir1)
	os.RemoveAll(dir2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListDirectoryfilterExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	// directory1
	dir1 := "ossutil-test-dir-" + randLowStr(5)
	subdir1 := "dir1"
	contents1 := map[string]string{}
	filenames1 := s.createTestFiles(dir1, subdir1, c, contents1)

	// directory2
	dir2 := "ossutil-test-dir-" + randLowStr(5)
	subdir2 := "dir2"
	contents2 := map[string]string{}
	filenames2 := s.createTestFiles(dir2, subdir2, c, contents2)

	// upload directory1
	args := []string{dir1, bucketStr + "/" + dir1}
	cmdline := []string{"ossutil", "cp", dir1, bucketStr + "/" + dir1, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// upload directory2
	args = []string{dir2, bucketStr + "/" + dir2}
	cmdline = []string{"ossutil", "cp", dir2, bucketStr + "/" + dir2, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// ls files
	// e.g., ossutil ls oss://tempb4/ --include "*.jpg" --include dir1
	cmdline = []string{"ossutil", "ls", bucketStr, "--exclude"}
	strFilter := "*" + dir1
	cmdline = append(cmdline, strFilter)

	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	bDirectory := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"directory":       &bDirectory,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), dir1), Equals, false)
	for _, filename := range filenames1 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	c.Assert(strings.Contains(string(fileBody), dir2), Equals, true)
	for _, filename := range filenames2 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir1)
	os.RemoveAll(dir2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListPartfilterInclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	// object jpg
	object1 := "ossutil-test-object-" + randLowStr(5) + ".jpg"
	_, err = bucket.InitiateMultipartUpload(object1)
	c.Assert(err, IsNil)

	// object png
	object2 := "ossutil-test-object-" + randLowStr(5) + ".png"
	_, err = bucket.InitiateMultipartUpload(object2)
	c.Assert(err, IsNil)

	// ls files
	cmdline := []string{"ossutil", "ls", "-m", bucketStr, "--include", "*.jpg"}

	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	bPart := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"multipart":       &bPart,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), object1), Equals, true)
	c.Assert(strings.Contains(string(fileBody), object2), Equals, false)

	// cleanup
	os.Remove(testOutFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListPartfilterExclude(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucketStr := CloudURLToString(bucketName, "")

	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	// object jpg
	object1 := "ossutil-test-object-" + randLowStr(5) + ".jpg"
	_, err = bucket.InitiateMultipartUpload(object1)
	c.Assert(err, IsNil)

	// object png
	object2 := "ossutil-test-object-" + randLowStr(5) + ".png"
	_, err = bucket.InitiateMultipartUpload(object2)
	c.Assert(err, IsNil)

	// ls files
	cmdline := []string{"ossutil", "ls", "-m", bucketStr, "--exclude", "*.jpg"}

	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	bPart := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"multipart":       &bPart,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), object1), Equals, false)
	c.Assert(strings.Contains(string(fileBody), object2), Equals, true)

	// cleanup
	os.Remove(testOutFileName)
	s.removeBucket(bucketName, true, c)
}

// list objects versions
func (s *OssutilCommandSuite) TestListObjectVersionsNormal(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	fileName := "test-ossutil-file-" + randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)

	// put 3 objects
	numObject := 3
	for i := 0; i < numObject; i++ {
		object := fmt.Sprintf("lstest:#%d", i)
		s.putObject(bucketName, object, fileName, c)
	}

	// put 1 object
	numObject++
	object := "another_object" + randLowStr(5)
	s.putObject(bucketName, object, fileName, c)

	//put 2 directories
	numDir := 2
	for i := 0; i < numDir; i++ {
		object := fmt.Sprintf("lstest:#%d/", i)
		s.putObject(bucketName, object, fileName, c)
	}

	allVersions := true
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"allVersions":     &allVersions,
		"limitedNum":      &limitedNum,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	_, err := cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(fileBody), "VERSIONID"), Equals, true)
	c.Assert(strings.Contains(string(fileBody), "COMMON-PREFIX"), Equals, false)

	strTotal := fmt.Sprintf("Object Number is: %d", numObject+numDir)
	c.Assert(strings.Contains(string(fileBody), strTotal), Equals, true)

	// ls oss://bucket -d
	directory := true
	options["directory"] = &directory
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	fileBody, err = ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(fileBody), "VERSIONID"), Equals, true)
	c.Assert(strings.Contains(string(fileBody), "COMMON-PREFIX"), Equals, true)

	// rm all object
	recursive := true
	force := true
	deleteOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &recursive,
		"force":           &force,
	}
	rmArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("rm", rmArgs, deleteOptions)
	c.Assert(err, IsNil)

	// list object again
	delete(options, "directory")
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	os.Stdout = oldStdout

	fileBody, err = ioutil.ReadFile(testOutFileName)
	strTotal = fmt.Sprintf("Object Number is: %d", (numObject+numDir)*2)
	c.Assert(strings.Contains(string(fileBody), strTotal), Equals, true)

	os.Remove(fileName)
	os.Remove(testOutFileName)
	s.removeBucket(bucketName, true, c)
}

// list objects versions
// list objects versions
func (s *OssutilCommandSuite) TestListObjectVersionsMarker(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	fileName := "test-ossutil-file-" + randLowStr(10)
	content := randLowStr(10)
	s.createFile(fileName, content, c)

	object := "test-ossutil-object" + randLowStr(10)

	// default max-key 100
	numObject := 200
	for i := 0; i < numObject; i++ {
		s.putObject(bucketName, object, fileName, c)
	}

	allVersions := true
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"allVersions":     &allVersions,
		"limitedNum":      &limitedNum,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	_, err := cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(fileBody), "VERSIONID"), Equals, true)
	c.Assert(strings.Contains(string(fileBody), "COMMON-PREFIX"), Equals, false)

	strTotal := fmt.Sprintf("Object Number is: %d", numObject)
	c.Assert(strings.Contains(string(fileBody), strTotal), Equals, true)

	// test for limit num
	limitedNum = strconv.FormatInt(int64(numObject/4), 10)
	saveLimit := numObject / 4
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	fileBody, err = ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)
	strTotal = fmt.Sprintf("Object Number is: %d", saveLimit)
	c.Assert(strings.Contains(string(fileBody), strTotal), Equals, true)

	//get all versions info
	bucket, err := listCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	maxKeys := oss.MaxKeys(1000)
	listResult, err := listCommand.command.ossListObjectVersionsRetry(bucket, maxKeys)
	c.Assert(len(listResult.ObjectVersions), Equals, numObject)

	selectVersionId := listResult.ObjectVersions[int(saveLimit)].VersionId

	// test for key-marker version-id-mark
	limitedNum = strconv.FormatInt(-1, 10)
	options["marker"] = &object
	options["versionIdMarker"] = &selectVersionId

	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()

	os.Stdout = oldStdout

	fileBody, err = ioutil.ReadFile(testOutFileName)
	strTotal = fmt.Sprintf("Object Number is: %d", numObject-saveLimit-1)
	c.Assert(strings.Contains(string(fileBody), strTotal), Equals, true)

	os.Remove(fileName)
	os.Remove(testOutFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListObjectfilterIncludeVersions(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	bucketStr := CloudURLToString(bucketName, "")

	dir := "ossutil-test-dir-" + randLowStr(5)
	subdir := "dir1"
	contents := map[string]string{}
	filenames := s.createTestFiles(dir, subdir, c, contents)

	// upload files
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// "rm oss://bucket/prefix -r -f"
	// remove object
	s.removeObjects(bucketName, "re", true, true, c)

	// ls files
	// e.g., ossutil ls oss://tempb4/ --include "*.jpg" --include "*.txt"
	cmdline = []string{"ossutil", "ls", bucketStr, "--include", "*.jpg", "--include", "*.txt"}
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	allVersions := true
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"allVersions":     &allVersions,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	files := filterStrsWithInclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	files = filterStrsWithInclude(filenames, "*.txt")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	files = filterStrsWithInclude(filenames, "*.rtf")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListDirectoryfilterIncludeVersions(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, string(oss.VersionEnabled), c)

	bucketStr := CloudURLToString(bucketName, "")

	// directory1
	dir1 := "ossutil-test-dir-" + randLowStr(5)
	subdir1 := "dir1"
	contents1 := map[string]string{}
	filenames1 := s.createTestFiles(dir1, subdir1, c, contents1)

	// directory2
	dir2 := "ossutil-test-dir-" + randLowStr(5)
	subdir2 := "dir2"
	contents2 := map[string]string{}
	filenames2 := s.createTestFiles(dir2, subdir2, c, contents2)

	// upload directory1
	args := []string{dir1, bucketStr + "/" + dir1}
	cmdline := []string{"ossutil", "cp", dir1, bucketStr + "/" + dir1, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// upload directory2
	args = []string{dir2, bucketStr + "/" + dir2}
	cmdline = []string{"ossutil", "cp", dir2, bucketStr + "/" + dir2, "-rf"}
	showElapse, err = s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// "rm oss://bucket/prefix -r -f"
	// remove object
	s.removeObjects(bucketName, "re", true, true, c)

	// ls files
	// e.g., ossutil ls oss://tempb4/ --include "*.jpg" --include dir1
	cmdline = []string{"ossutil", "ls", bucketStr, "--include"}
	strFilter := "*" + dir1
	cmdline = append(cmdline, strFilter)

	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	bDirectory := true
	allVersions := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"limitedNum":      &limitedNum,
		"directory":       &bDirectory,
		"allVersions":     &allVersions,
	}

	testOutFileName := "ossutil-test-outfile-" + randLowStr(5)
	testOutFile, _ := os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout := os.Stdout
	os.Stdout = testOutFile
	os.Args = cmdline
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), dir1), Equals, true)
	for _, filename := range filenames1 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	c.Assert(strings.Contains(string(fileBody), dir2), Equals, false)
	for _, filename := range filenames2 {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir1)
	os.RemoveAll(dir2)
	s.removeBucket(bucketName, true, c)
}
