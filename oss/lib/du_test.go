package lib

import (
	"os"
	"strings"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestDuObjectSize(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	dir := "ossutil-test-dir-" + randLowStr(5)
	subDir := "dir1"
	contents := map[string]string{}
	s.createTestFiles(dir, subDir, c, contents)
	filePathList, _ := getFileList(dir)

	allObjectSize := int64(0)
	subDirSize := int64(0)

	for _, filename := range filePathList {
		fileInfo, err := os.Stat(filename)
		c.Assert(err, IsNil)
		if fileInfo.IsDir() {
			continue
		}

		allObjectSize += fileInfo.Size()
		if strings.Contains(filename, subDir) {
			subDirSize += fileInfo.Size()
		}
	}

	// upload files
	bucketStr := CloudURLToString(bucketName, "")
	args := []string{dir, bucketStr}
	cmdline := []string{"ossutil", "cp", dir, bucketStr, "-rf"}
	showElapse, err := s.rawCPWithFilter(args, true, true, false, DefaultBigFileThreshold, CheckpointDir, cmdline, "", "")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// du size,all bucket
	command := "du"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args = []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(len(filePathList)))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, allObjectSize)
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(0))

	//du size:directory
	srcUrl = CloudURLToString(bucketName, subDir)
	args = []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, subDirSize)
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(0))

	os.RemoveAll(dir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuObjectSizeWithBlockSize(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := "test-file-" + randStr(5)
	content := randLowStr(1024 * 1024)
	s.createFile(fileName, content, c)
	object := "test-object-" + randStr(5)
	s.putObject(bucketName, object, fileName, c)

	// du size,all bucket
	command := "du"
	str := ""
	strUnit := "KB"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"blockSize":       &strUnit,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuBlockSizeError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	fileName := "test-file-" + randStr(5)
	content := randLowStr(1024 * 1024)
	s.createFile(fileName, content, c)
	object := "test-object-" + randStr(5)
	s.putObject(bucketName, object, fileName, c)

	// du size,all bucket
	command := "du"
	str := ""
	strUnit := "KBbb" // error value
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"blockSize":       &strUnit,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuPartSize(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	content_len := 100
	content := randLowStr(content_len)
	fileName := "ossutil-testfile-" + randLowStr(5)
	s.createFile(fileName, content, c)

	// object jpg
	object1 := "ossutil-test-object-" + randLowStr(5) + ".jpg"
	imu1, err := bucket.InitiateMultipartUpload(object1)
	c.Assert(err, IsNil)
	_, err = bucket.UploadPartFromFile(imu1, fileName, 0, int64(content_len), 1)
	c.Assert(err, IsNil)

	// object png
	object2 := "ossutil-test-object-" + randLowStr(5) + ".png"
	imu2, err := bucket.InitiateMultipartUpload(object2)
	c.Assert(err, IsNil)
	_, err = bucket.UploadPartFromFile(imu2, fileName, 0, int64(content_len), 1)
	c.Assert(err, IsNil)

	// du size,all bucket
	command := "du"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	srcUrl := CloudURLToString(bucketName, "")
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(2))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(2*content_len))

	// cleanup
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuObjectAndPartSize(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	content_len := 100
	content := randLowStr(content_len)
	fileName := "ossutil-testfile-" + randLowStr(5)
	s.createFile(fileName, content, c)

	dirName := randLowStr(5)
	// object jpg
	object1 := dirName + "/" + "ossutil-test-object-" + randLowStr(5) + ".jpg"
	s.PutObject(bucketName, object1, content, c)

	// object png
	object2 := dirName + "/" + "ossutil-test-object-" + randLowStr(5) + ".png"
	imu2, err := bucket.InitiateMultipartUpload(object2)
	c.Assert(err, IsNil)
	_, err = bucket.UploadPartFromFile(imu2, fileName, 0, int64(content_len), 1)
	c.Assert(err, IsNil)

	// du size,all bucket
	command := "du"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	srcUrl := CloudURLToString(bucketName, dirName)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(1))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, int64(content_len))
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(1))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(content_len))

	// cleanup
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuPayerObject(c *C) {
	bucketName := payerBucket
	objectName := randStr(10)

	fileName := "ossutil-test-file-" + randLowStr(5)
	textBuffer := randStr(100)
	s.createFile(fileName, textBuffer, c)

	//put object, with --payer=requester
	args := []string{fileName, CloudURLToString(bucketName, objectName)}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// du size,all bucket
	command := "du"
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
	srcUrl := CloudURLToString(bucketName, objectName)
	args = []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(1))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, int64(len(textBuffer)))
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(0))
}

func (s *OssutilCommandSuite) TestDuPayerErrorObject(c *C) {
	bucketName := payerBucket
	objectName := randStr(10)

	// requester is error
	command := "du"
	str := ""
	requester := "requester_test"
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	srcUrl := CloudURLToString(bucketName, objectName)
	args := []string{srcUrl}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// srcUrl is error
	args = []string{"http://bucketname"}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestDuPayerPart(c *C) {
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

	// du size,all bucket
	command := "du"
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
	srcUrl := CloudURLToString(bucketName, object)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(1))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(content_len))

	os.Remove(fileName)
}

func (s *OssutilCommandSuite) TestDuHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"du"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestDuVersionsObjectAndStorageClass(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	// put bucket version:enabled
	var str string
	strMethod := "put"
	versionsOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}
	versioningArgs := []string{CloudURLToString(bucketName, ""), "enabled"}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, versionsOptions)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// put object
	content_len := 100
	content := randLowStr(content_len)
	fileName := "ossutil-testfile-" + randLowStr(5)
	s.createFile(fileName, content, c)
	dirName := randLowStr(5)

	// object jpg
	object := dirName + "/" + "ossutil-test-object-" + randLowStr(5) + ".jpg"
	// standard
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageStandard))
	c.Assert(err, IsNil)
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageStandard))
	c.Assert(err, IsNil)

	// archive
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageIA))
	c.Assert(err, IsNil)
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageIA))
	c.Assert(err, IsNil)

	// IA
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageArchive))
	c.Assert(err, IsNil)
	err = bucket.PutObject(object, strings.NewReader(content), oss.ObjectStorageClass(oss.StorageArchive))
	c.Assert(err, IsNil)

	// du size,all bucket
	allVersions := true
	command := "du"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"allVersions":     &allVersions,
	}
	srcUrl := CloudURLToString(bucketName, dirName)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	c.Assert(duSizeCommand.duOption.totalObjectCount, Equals, int64(6))
	c.Assert(duSizeCommand.duOption.sumObjectSize, Equals, 6*int64(content_len))
	c.Assert(duSizeCommand.duOption.totalPartCount, Equals, int64(0))
	c.Assert(duSizeCommand.duOption.sumPartSize, Equals, int64(0))

	c.Assert(duSizeCommand.duOption.countTypeMap["Standard"], Equals, int64(2))
	c.Assert(duSizeCommand.duOption.countTypeMap["IA"], Equals, int64(2))
	c.Assert(duSizeCommand.duOption.countTypeMap["Archive"], Equals, int64(2))
	c.Assert(duSizeCommand.duOption.sizeTypeMap["Standard"], Equals, int64(content_len*2))
	c.Assert(duSizeCommand.duOption.sizeTypeMap["IA"], Equals, int64(content_len*2))
	c.Assert(duSizeCommand.duOption.sizeTypeMap["Archive"], Equals, int64(content_len*2))

	// cleanup
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestDuUploadIdNotExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	bucket, err := makeBucketCommand.command.ossBucket(bucketName)

	object := MultiPartObject{
		objectName: "123",
		uploadId:   "123",
	}
	err = duSizeCommand.statPartSize(bucket, object)
	c.Assert(err, IsNil)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUploadIdProducer(c *C) {
	chObjects := make(chan MultiPartObject, ChannelBuf)
	chListError := make(chan error, 1)
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)
	bucket, err := client.Bucket(bucketNameNotExist)
	c.Assert(err, IsNil)
	duSizeCommand.uploadIdProducer(bucket, chObjects, chListError)
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

	chObjects2 := make(chan MultiPartObject, ChannelBuf)
	chListError2 := make(chan error, 1)
	bucket2, err := client.Bucket(bucketNameExist)
	c.Assert(err, IsNil)
	duSizeCommand.uploadIdProducer(bucket2, chObjects2, chListError2)
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
