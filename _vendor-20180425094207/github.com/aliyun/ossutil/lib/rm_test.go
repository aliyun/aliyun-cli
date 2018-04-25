package lib

import (
	"fmt"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestRemoveObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put object
	object := "TestRemoveObject"
	s.putObject(bucketName, object, uploadFileName, c)

	// list object
	objects := s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	// remove object
	s.removeObjects(bucketName, object, false, true, c)

	// list object
	objects = s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 0)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveObjects(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put object
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("remove%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	command := "rm"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// list object
	objects := s.listObjects(bucketName, "", "ls - ", c)
	c.Assert(len(objects), Equals, num)

	// "rm oss://bucket/ -r"
	// remove object
	s.removeObjects(bucketName, "", true, false, c)

	objects = s.listObjects(bucketName, "", "ls - ", c)
	c.Assert(len(objects), Equals, num)

	// "rm oss://bucket/prefix -r -f"
	// remove object
	s.removeObjects(bucketName, "re", true, true, c)

	// list object
	objects = s.listObjects(bucketName, "", "ls - ", c)
	c.Assert(len(objects), Equals, 0)

	//reput objects and delete bucket
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("remove%d", i)
		s.putObject(bucketName, object, uploadFileName, c)
	}

	// list buckets
	bucketNames := s.listBuckets(false, c)
	c.Assert(FindPos(bucketName, bucketNames) != -1, Equals, true)

	// error remove bucket with config
	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s[Bucket-Cname]\n%s=%s", "abc", "def", "ghi", bucketName, "abc", bucketName, "abc")
	s.createFile(cfile, data, c)

	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"recursive":       &ok,
		"bucket":          &ok,
		"allType":         &ok,
		"force":           &ok,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
		"recursive":       &ok,
		"bucket":          &ok,
		"allType":         &ok,
		"force":           &ok,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)
	time.Sleep(7 * time.Second)

	// list buckets
	bucketNames = s.listBuckets(false, c)
	c.Assert(FindPos(bucketName, bucketNames) == -1, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveObjectBucketOption(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	object := "test_object"
	command := "rm"
	args := []string{CloudURLToString(bucketName, object)}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"bucket":          &ok,
		"force":           &ok,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// list buckets
	bucketNames := s.listBuckets(false, c)
	c.Assert(FindPos(bucketName, bucketNames) != -1, Equals, true)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestErrRemove(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	showElapse, err := s.rawRemove([]string{"oss://"}, false, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawRemove([]string{"./"}, false, true, true)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawRemove([]string{CloudURLToString(bucketName, "")}, false, true, false)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawRemove([]string{"oss:///object"}, false, true, false)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// remove bucket without force
	showElapse, err = s.rawRemove([]string{CloudURLToString(bucketName, "")}, false, false, true)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	bucketStat := s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatName], Equals, bucketName)

	// batch delete not exist objects
	object := "batch_delete_notexst_object"
	showElapse, err = s.rawRemove([]string{CloudURLToString(bucketName, object)}, true, true, false)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	s.removeBucket(bucketName, true, c)

	// clear not exist bucket
	bucketName = bucketNameNotExist
	showElapse, err = s.rawRemove([]string{CloudURLToString(bucketName, "")}, true, true, false)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	// test oss batch delete not exist objects
	objects := []string{}
	ossBucket, err := removeCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	num, err := removeCommand.ossBatchDeleteObjectsRetry(ossBucket, objects)
	c.Assert(err, IsNil)
	c.Assert(num, Equals, 0)
}

func (s *OssutilCommandSuite) TestErrDeleteObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	bucket, err := removeCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	object := "object"
	err = removeCommand.ossDeleteObjectRetry(bucket, object)
	c.Assert(err, NotNil)

	_, err = removeCommand.ossBatchDeleteObjectsRetry(bucket, []string{object})
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestAllTypeObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	err := s.initRemove(bucketName, "", "rm -marf")
	c.Assert(err, IsNil)
	removeCommand.RunCommand()

	normal_object := randStr(10)
	s.putObject(bucketName, normal_object, uploadFileName, c)

	object := randStr(10)
	s.putObject(bucketName, object, uploadFileName, c)

	objects := s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	bucket, err := copyCommand.command.ossBucket(bucketName)
	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}

	lmr, e := bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads) >= 20, Equals, true)

	_, e = s.removeWrapper("rm -arf", bucketName, object, c)
	c.Assert(e, IsNil)

	lmr, e = bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, 0)

	// list normal_object
	objects = s.listObjects(bucketName, normal_object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, normal_object)

	err = s.initRemove(bucketName, "", "rm -marfb")
	c.Assert(err, IsNil)
	removeCommand.RunCommand()
}

func (s *OssutilCommandSuite) TestMultipartUpload(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put object
	object := "TestMultipartObject"
	s.putObject(bucketName, object, uploadFileName, c)

	// list object
	objects := s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, object)

	bucket, err := copyCommand.command.ossBucket(bucketName)
	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}

	lmr, e := bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads) >= 20, Equals, true)

	_, e = s.removeWrapper("rm -mrf", bucketName, object, c)
	c.Assert(e, IsNil)

	lmr, e = bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, 0)

	obj := "TestMultipartObjectUploads"
	s.putObject(bucketName, obj, uploadFileName, c)

	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(obj)
		c.Assert(err, IsNil)
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMultipartUpload_Prefix(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	bucket, err := copyCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)

	s.clearObjects(bucketName, "", c)

	object := "TestMultipartObject"
	s.putObject(bucketName, object, uploadFileName, c)

	object1 := "TestMultipartObject" + "prefix"
	s.putObject(bucketName, object1, uploadFileName, c)

	object2 := "TestMultipartObject" + "/dir/test"
	s.putObject(bucketName, object2, uploadFileName, c)

	// list object
	objects := s.listObjects(bucketName, object, "ls - ", c)
	c.Assert(len(objects), Equals, 3)

	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object)
		c.Assert(err, IsNil)
	}

	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object1)
		c.Assert(err, IsNil)
	}

	for i := 0; i < 20; i++ {
		_, err = bucket.InitiateMultipartUpload(object2)
		c.Assert(err, IsNil)
	}

	lmr, e := bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads) >= 20*3, Equals, true)

	_, e = s.removeWrapper("rm -mrf", bucketName, "", c)
	c.Assert(e, IsNil)

	lmr, e = bucket.ListMultipartUploads(oss.Prefix(object))
	c.Assert(e, IsNil)
	c.Assert(len(lmr.Uploads), Equals, 0)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAbortNotExistUploadId(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put object
	object := "TestMultipartObject"
	s.putObject(bucketName, object, uploadFileName, c)

	bucket, err := copyCommand.command.ossBucket(bucketName)
	imur, err := bucket.InitiateMultipartUpload(object)
	c.Assert(err, IsNil)
	c.Assert(imur.Bucket, Equals, bucketName)
	c.Assert(imur.Key, Equals, object)
	c.Assert(imur.UploadID != "", Equals, true)

	err = removeCommand.ossAbortMultipartUploadRetry(bucket, object, imur.UploadID)
	c.Assert(err, IsNil)

	err = removeCommand.ossAbortMultipartUploadRetry(bucket, object, imur.UploadID)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMultipartError(c *C) {
	bucketName := bucketNameExist
	object := "TestMultipartError"

	_, e := s.removeWrapper("rm -mb", bucketName, object, c)
	c.Assert(e, NotNil)

	_, e = s.removeWrapper("rm -mf", bucketName, "", c)
	c.Assert(e, NotNil)
}

func (s *OssutilCommandSuite) TestAllTypeError(c *C) {
	bucketName := bucketNameExist
	object := "random"

	_, e := s.removeWrapper("rm -ab", bucketName, object, c)
	c.Assert(e, NotNil)
}
