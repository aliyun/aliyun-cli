package lib

import (
	"fmt"
	"os"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestAllPartSize(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024 * 10)
	s.createFile(fileName, strText, c)

	// prepare chunks
	chunks, err := oss.SplitFileByPartNum(fileName, 10)
	c.Assert(err, IsNil)

	fd, err := os.Open(fileName)
	c.Assert(err, IsNil)
	defer fd.Close()

	// begin upload part
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	imur, err := bucket.InitiateMultipartUpload(objectName)
	c.Assert(err, IsNil)

	var parts []oss.UploadPart
	for _, chunk := range chunks {
		fd.Seek(chunk.Offset, os.SEEK_SET)
		part, err := bucket.UploadPart(imur, fd, chunk.Size, chunk.Number)
		c.Assert(err, IsNil)
		parts = append(parts, part)
	}

	// not CompleteMultipartUpload

	// begin getallpartsize
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// output to file
	outputFile := "test-file-" + randLowStr(5)
	testResultFile, err = os.OpenFile(outputFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	c.Assert(err, IsNil)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	alArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("getallpartsize", alArgs, options)
	if err != nil {
		fmt.Printf("error:%s\n", err.Error())
	}
	c.Assert(err, IsNil)
	testResultFile.Close()

	os.Stdout = oldStdout

	// check file content
	outBody := s.readFile(outputFile, c)
	c.Assert(strings.Contains(outBody, "total part count:10"), Equals, true)

	os.Remove(outputFile)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAllPartSizeBucketError(c *C) {

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	alArgs := []string{"oss:////"}
	_, err := cm.RunCommand("getallpartsize", alArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestAllPartSizeEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}

	alArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("getallpartsize", alArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAllPartSizeHelp(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"getallpartsize"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
