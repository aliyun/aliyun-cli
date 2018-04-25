package lib

import (
	"fmt"
	"net/url"
	"os"
	"sort"
	"strconv"

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
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\nretryTimes=%s", endpoint, accessKeyID, accessKeySecret, "errretry")
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
