package lib

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"strconv"
	"strings"

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

func (s *OssutilCommandSuite) TestRmObjectfilter(c *C) {
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
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	for _, filename := range filenames {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	// then rm objects
	cmdline = []string{"ossutil", "rm", bucketStr, "-rf", "--include", "*.jpg"}
	rmArgs := []string{CloudURLToString(bucketName, "")}
	bRecusive := true
	bForce := true
	rmOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"recursive":       &bRecusive,
		"force":           &bForce,
	}
	os.Args = cmdline
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// check again after rm
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout

	fileBody, err = ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify include
	files := filterStrsWithInclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// Verify exclude
	files = filterStrsWithExclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRmPartfilter(c *C) {
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

func (s *OssutilCommandSuite) TestRmPartfilterExclude(c *C) {
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
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout
	os.Args = []string{}

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), object1), Equals, true)
	c.Assert(strings.Contains(string(fileBody), object2), Equals, true)

	// then rm objects
	cmdline := []string{"ossutil", "rm", bucketStr, "-rf", "--include", "*.jpg"}
	rmArgs := []string{CloudURLToString(bucketName, "")}
	bRecusive := true
	bForce := true
	bMultiPart := true
	rmOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"recursive":       &bRecusive,
		"force":           &bForce,
		"multipart":       &bMultiPart,
	}

	os.Args = cmdline
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// check again after rm
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout

	fileBody, err = ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	c.Assert(strings.Contains(string(fileBody), object1), Equals, false)
	c.Assert(strings.Contains(string(fileBody), object2), Equals, true)

	// cleanup
	os.Remove(testOutFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRmSpecialCharacterKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := "ossutil-test-file-" + randLowStr(5)
	text := randLowStr(100)
	s.createFile(fileName, text, c)

	// put object with special characters key
	object := "%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2F%C0%AE%C0%AE%2Fetc%2Fprofile"
	command := "cp"
	args := []string{fileName, CloudURLToString(bucketName, object)}
	str := ""
	ok := true
	cpDir := CheckpointDir
	thre := strconv.FormatInt(DefaultBigFileThreshold, 10)
	routines := strconv.Itoa(Routines)
	partSize := strconv.FormatInt(DefaultPartSize, 10)
	encodingType := "url"
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
		"encodingType":     &encodingType,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// get object
	downloadFileName := fileName + "-down"
	decodedKey, _ := url.QueryUnescape(object)
	s.getObject(bucketName, decodedKey, downloadFileName, c)
	fileData := s.readFile(downloadFileName, c)
	c.Assert(fileData, Equals, text)

	os.Remove(fileName)
	os.Remove(downloadFileName)
	s.removeBucket(bucketName, true, c)
}

// versions
func (s *OssutilCommandSuite) TestRemoveObjectInVersioningBucket(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

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

func (s *OssutilCommandSuite) TestRemoveObjectsInVersioningBucket(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

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
	_, err = cm.RunCommand(command, args, options)
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
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRemoveObjectWithVersionId(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	textBuffer := randStr(100)
	s.createFile(uploadFileName, textBuffer, c)

	// put object
	objectName := "TestRemoveObject-versionid"
	s.putObject(bucketName, objectName, uploadFileName, c)
	objectStat := s.getStat(bucketName, objectName, c)
	versionId := objectStat["X-Oss-Version-Id"]

	// list object
	objects := s.listObjects(bucketName, objectName, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, objectName)

	// remove object
	// rm --version-id oss:\bucketName\objectName
	command := "rm"
	args := []string{CloudURLToString(bucketName, objectName)}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// list object
	objects = s.listObjects(bucketName, objectName, "ls - ", c)
	c.Assert(len(objects), Equals, 0)

	// rm -b
	args = []string{CloudURLToString(bucketName, "")}
	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &str,
		"bucket":          &ok,
		"force":           &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestRemoveObjectWithAllVersion(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	textBuffer := randStr(100)
	s.createFile(uploadFileName, textBuffer, c)

	// put object 20 times, and has 20 object with different version id
	objectName := "TestRemoveObject-allversion"
	num := 20
	for i := 0; i < num; i++ {
		s.putObject(bucketName, objectName, uploadFileName, c)
	}
	objectStat := s.getStat(bucketName, objectName, c)
	versionId := objectStat["X-Oss-Version-Id"]

	// list object
	objects := s.listObjects(bucketName, objectName, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, objectName)

	// remove the latest version object
	// rm --version-id oss:\bucketName\objectName
	command := "rm"
	args := []string{CloudURLToString(bucketName, objectName)}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// list object, and remains 19
	objects = s.listObjects(bucketName, objectName, "ls - ", c)
	c.Assert(len(objects), Equals, 1)
	c.Assert(objects[0], Equals, objectName)

	// rm --all-versions oss:\bucketName\objectName
	args = []string{CloudURLToString(bucketName, objectName)}
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"allVersions":     &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	objects = s.listObjects(bucketName, objectName, "ls - ", c)
	c.Assert(len(objects), Equals, 0)

	// rm -b
	args = []string{CloudURLToString(bucketName, "")}
	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &str,
		"bucket":          &ok,
		"force":           &ok,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestRmObjectfilterVersioning(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)
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
	// remove object,create delete marker
	s.removeObjects(bucketName, "re", true, true, c)

	// ls files
	limitedNum := strconv.FormatInt(-1, 10)
	lsArgs := []string{CloudURLToString(bucketName, "")}
	str := ""
	allVersions := true
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
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout

	fileBody, err := ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify
	for _, filename := range filenames {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	// then rm objects
	cmdline = []string{"ossutil", "rm", bucketStr, "-rf", "--include", "*.jpg"}
	rmArgs := []string{CloudURLToString(bucketName, "")}
	bRecusive := true
	bForce := true
	rmOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"recursive":       &bRecusive,
		"force":           &bForce,
		"allVersions":     &allVersions,
	}
	os.Args = cmdline
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// check again after rm
	testOutFile, _ = os.OpenFile(testOutFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	oldStdout = os.Stdout
	os.Stdout = testOutFile
	_, err = cm.RunCommand("ls", lsArgs, options)
	c.Assert(err, IsNil)
	testOutFile.Close()
	os.Stdout = oldStdout

	fileBody, err = ioutil.ReadFile(testOutFileName)
	c.Assert(err, IsNil)

	// Verify include
	files := filterStrsWithInclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, false)
	}

	// Verify exclude
	files = filterStrsWithExclude(filenames, "*.jpg")
	for _, filename := range files {
		c.Assert(strings.Contains(string(fileBody), filename), Equals, true)
	}

	// cleanup
	os.Remove(testOutFileName)
	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRmObjectWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// rm with payer
	rmArgs := []string{CloudURLToString(bucketName, uploadFileName)}
	bForce := true
	str := ""
	requester := "requester"
	rmOptions := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"force":           &bForce,
		"payer":           &requester,
	}
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestRmBatchObjectWithPayer(c *C) {
	bucketName := payerBucket

	dir := "ossutil-test-dir-" + randLowStr(5)
	subDir := "dir1"
	contents := map[string]string{}
	s.createTestFiles(dir, subDir, c, contents)

	// put object, with --payer=requester
	args := []string{dir, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, true, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// rm with payer
	rmArgs := []string{CloudURLToString(bucketName, dir)}
	bForce := true
	recursive := true
	str := ""
	requester := "requester"
	rmOptions := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"force":           &bForce,
		"recursive":       &recursive,
		"payer":           &requester,
	}
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)

	os.RemoveAll(dir)
}

func (s *OssutilCommandSuite) TestRmBatchPartsWithPayer(c *C) {
	bucketName := payerBucket
	client, err := oss.New(payerBucketEndPoint, accessKeyID, accessKeySecret)
	bucket, err := client.Bucket(bucketName)

	content_len := 100
	content := randLowStr(content_len)
	fileName := "ossutil-testfile-" + randLowStr(5)
	s.createFile(fileName, content, c)

	// object jpg
	object := "ossutil-test-object-" + randLowStr(5) + ".jpg"
	imu, err := bucket.InitiateMultipartUpload(object, oss.RequestPayer(oss.PayerType("requester")))
	c.Assert(err, IsNil)
	_, err = bucket.UploadPartFromFile(imu, fileName, 0, int64(content_len), 1, oss.RequestPayer(oss.PayerType("requester")))
	c.Assert(err, IsNil)

	// rm with payer
	rmArgs := []string{CloudURLToString(bucketName, "")}
	bForce := true
	recursive := true
	allType := true
	str := ""
	requester := "requester"
	rmOptions := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"force":           &bForce,
		"recursive":       &recursive,
		"allType":         &allType,
		"payer":           &requester,
	}
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestMultipartUploadsProducer(c *C) {
	chObjects := make(chan uploadIdInfoType, ChannelBuf)
	chListError := make(chan error, 1)
	cloudURL, err := CloudURLFromString(CloudURLToString(bucketNameNotExist, "demo.txt"), "")
	c.Assert(err, IsNil)
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)
	bucket, err := client.Bucket(bucketNameNotExist)
	c.Assert(err, IsNil)
	removeCommand.multipartUploadsProducer(bucket, cloudURL, chObjects, chListError)
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

	chObjects2 := make(chan uploadIdInfoType, ChannelBuf)
	chListError2 := make(chan error, 1)
	cloudURL2, err := CloudURLFromString(CloudURLToString(bucketNameExist, ""), "")
	c.Assert(err, IsNil)
	bucket2, err := client.Bucket(bucketNameExist)
	c.Assert(err, IsNil)
	removeCommand.multipartUploadsProducer(bucket2, cloudURL2, chObjects2, chListError2)
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
