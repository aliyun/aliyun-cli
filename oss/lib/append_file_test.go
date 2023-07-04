package lib

import (
	"fmt"
	"io/ioutil"
	"os"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestAppendFileSuccessWithoutMeta(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	// begin append
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	if err != nil {
		fmt.Printf("error:%s\n", err.Error())
	}
	c.Assert(err, IsNil)

	// append again
	_, err = cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, IsNil)

	// downalod object
	cpDir := CheckpointDir
	cpOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"checkpointDir":   &cpDir,
		"configFile":      &configFile,
	}
	downFileName := randLowStr(10) + "-download"
	dwArgs := []string{CloudURLToString(bucketName, objectName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, cpOptions)
	c.Assert(err, IsNil)

	// compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(strText+strText, Equals, string(fileBody))

	os.Remove(fileName)
	os.Remove(downFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileSuccessWithMeta(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	// begin append
	var str string
	strMeta := "x-oss-meta-author:luxun"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"meta":            &strMeta,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	if err != nil {
		fmt.Printf("error:%s\n", err.Error())
	}
	c.Assert(err, IsNil)

	// downalod object
	cpDir := CheckpointDir
	cpOptions := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"checkpointDir":   &cpDir,
		"configFile":      &configFile,
	}
	downFileName := randLowStr(10) + "-download"
	dwArgs := []string{CloudURLToString(bucketName, objectName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, cpOptions)
	c.Assert(err, IsNil)

	// compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(strText, Equals, string(fileBody))

	// check meta
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	//get object meta
	props, err := bucket.GetObjectDetailedMeta(objectName)
	c.Assert(err, IsNil)

	author := props.Get("x-oss-meta-author")
	c.Assert(author, Equals, "luxun")
	c.Assert(err, IsNil)

	os.Remove(fileName)
	os.Remove(downFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	// begin append
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// cloud url is error
	appendArgs := []string{fileName, "oss:///1.jpg"}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	// object parameter is empy,error
	appendArgs = []string{fileName, CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	// input file is not exist
	appendArgs = []string{fileName + "-test", CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	// input file is dir
	dirName := fileName + "-dir"
	os.Mkdir(dirName, 0664)
	appendArgs = []string{dirName, CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)
	os.Remove(dirName)

	// append to a exist normal object
	s.PutObject(bucketName, objectName, strText, c)
	appendArgs = []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileClientError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// begin append
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileObjectExistMetaError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// put object
	objectName := "test-ossutil-object-" + randLowStr(10)
	s.AppendObject(bucketName, objectName, strText, 0, c)

	// begin append
	var str string
	strMeta := "x-oss-meta-author:luxun"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"meta":            &strMeta,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileParserMetaError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// objectName
	objectName := "test-ossutil-object-" + randLowStr(10)

	// begin append
	var str string
	strMeta := "x-oss-meta-author#luxun"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"meta":            &strMeta,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileAppendFromFileError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// objectName
	objectName := "test-ossutil-object-" + randLowStr(10)

	// file not exist
	var testCommand AppendFileCommand
	testCommand.afOption.fileName = randLowStr(12)

	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)
	err = testCommand.AppendFromFile(bucket, 0)
	c.Assert(err, NotNil)

	// append object error
	testCommand.afOption.fileName = fileName
	s.PutObject(bucketName, objectName, strText, c)

	// append error,position can't be 0
	err = testCommand.AppendFromFile(bucket, 0)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAppendFileHelp(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"appendfromfile"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestAppendFileWithPayer(c *C) {
	bucketName := payerBucket

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(100)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	// begin append
	var str string
	requester := "requester"
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}

	appendArgs := []string{fileName, CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("appendfromfile", appendArgs, options)
	if err != nil {
		fmt.Printf("error:%s\n", err.Error())
	}
	c.Assert(err, IsNil)

	// downalod object
	cpDir := CheckpointDir
	cpOptions := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"checkpointDir":   &cpDir,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	downFileName := randLowStr(10) + "-download"
	dwArgs := []string{CloudURLToString(bucketName, objectName), downFileName}
	_, err = cm.RunCommand("cp", dwArgs, cpOptions)
	c.Assert(err, IsNil)

	// compare content
	fileBody, err := ioutil.ReadFile(downFileName)
	c.Assert(err, IsNil)
	c.Assert(strText, Equals, string(fileBody))

	os.Remove(fileName)
	os.Remove(downFileName)
}
