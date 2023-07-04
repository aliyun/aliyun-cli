package lib

import (
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestCatObjectSuccess(c *C) {
	// create client and bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucketName := bucketNamePrefix + randLowStr(5)
	err = client.CreateBucket(bucketName)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	// put object
	//first:upload a object
	textBuffer := randStr(1024)
	objectName := randStr(10)
	err = bucket.PutObject(objectName, strings.NewReader(textBuffer))
	c.Assert(err, IsNil)

	// begin cat
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// output to file
	fileName := "test-file-" + randLowStr(5)
	testResultFile, _ = os.OpenFile(fileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	catArgs := []string{CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout

	// check file content
	catBody := s.readFile(fileName, c)
	c.Assert(strings.Contains(catBody, textBuffer), Equals, true)

	//remove file
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCatObjectError(c *C) {
	// create client and bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucketName := bucketNamePrefix + randLowStr(5)
	err = client.CreateBucket(bucketName)
	c.Assert(err, IsNil)

	// begin cat
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// object is empty
	catArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, NotNil)

	// object is not exist
	catArgs = []string{CloudURLToString(bucketName, randLowStr(5))}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, NotNil)

	// cloud url is error
	catArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, NotNil)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCatObjecEndpointEmptyError(c *C) {
	// create client and bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucketName := bucketNamePrefix + randLowStr(5)
	err = client.CreateBucket(bucketName)
	c.Assert(err, IsNil)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// begin cat
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}

	catArgs := []string{CloudURLToString(bucketName, randLowStr(5))}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCatObjectHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"cat"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestCatObjectWithVersion(c *C) {
	bucketName := bucketNamePrefix + "-cat-" + randLowStr(10)
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

	// begin cat without versionid
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// output to file
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	catArgs := []string{CloudURLToString(bucketName, objectName)}
	_, err := cm.RunCommand("cat", catArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout

	// check file content
	catBody := s.readFile(resultPath, c)
	c.Assert(strings.Contains(catBody, textBufferV2), Equals, true)
	os.Remove(resultPath)

	//begin cat with version id v1
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionIdV1,
	}
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout = os.Stdout
	os.Stdout = testResultFile

	catArgs = []string{CloudURLToString(bucketName, objectName)}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout

	// check file content
	catBody = s.readFile(resultPath, c)
	c.Assert(strings.Contains(catBody, textBufferV1), Equals, true)
	os.Remove(resultPath)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCatObjectWithPayer(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// begin cat with payer
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

	// output to file
	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	catArgs := []string{CloudURLToString(bucketName, uploadFileName)}
	_, err = cm.RunCommand("cat", catArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout

	// check file content
	catBody := s.readFile(resultPath, c)
	c.Assert(strings.Contains(catBody, content), Equals, true)
	os.Remove(resultPath)
}
