package lib

import (
	"fmt"
	"os"
	"strings"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestListPartSuccess(c *C) {
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

	// begin listpart
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

	listArgs := []string{CloudURLToString(bucketName, objectName), imur.UploadID}
	_, err = cm.RunCommand("listpart", listArgs, options)
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

func (s *OssutilCommandSuite) TestListPartError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-appendfile" + randLowStr(5)
	strText := randLowStr(1024 * 10)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)
	s.PutObject(bucketName, objectName, strText, c)

	// begin listpart
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// cloud url error
	listArgs := []string{"oss:///1.jpg", "aaaaaaaaaa"}
	_, err := cm.RunCommand("listpart", listArgs, options)
	c.Assert(err, NotNil)

	// object is empty
	listArgs = []string{CloudURLToString(bucketName, ""), "aaaaaaaaaa"}
	_, err = cm.RunCommand("listpart", listArgs, options)
	c.Assert(err, NotNil)

	// uploadid is not exist
	listArgs = []string{CloudURLToString(bucketName, objectName), "aaaaaaaaaa"}
	_, err = cm.RunCommand("listpart", listArgs, options)
	c.Assert(err, NotNil)

	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListPartClientError(c *C) {
	// create bucket
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// create file
	fileName := "test-ossutil-listpart" + randLowStr(5)
	strText := randLowStr(1024)
	s.createFile(fileName, strText, c)

	// object name
	objectName := "test-ossutil-object-" + randLowStr(10)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// begin listpart
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}

	listArgs := []string{CloudURLToString(bucketName, objectName), "aaaaaaaaaa"}
	_, err := cm.RunCommand("listpart", listArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	os.Remove(fileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestListPartHelp(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"listpart"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
