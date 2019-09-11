package lib

import (
	"io/ioutil"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestMkdirAll(c *C) {
	// create bucket
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucketName := bucketNamePrefix + randLowStr(12)
	err = client.CreateBucket(bucketName)
	c.Assert(err, IsNil)

	bucket, err := client.Bucket(bucketName)
	c.Assert(err, IsNil)

	// mkdir command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// error,not a cloud url
	mkArgs := []string{"http://www.nn.com/test.jpg"}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, NotNil)

	// error,bucket is empty
	mkArgs = []string{CloudURLToString("", "")}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, NotNil)

	// error,object is empty
	mkArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, NotNil)

	// success
	dirNameA := randLowStr(12)
	mkArgs = []string{CloudURLToString(bucketName, dirNameA)}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, IsNil)

	//mkdir again ,error
	mkArgs = []string{CloudURLToString(bucketName, dirNameA)}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, NotNil)

	// dirname/
	dirNameB := randLowStr(12) + "/"
	mkArgs = []string{CloudURLToString(bucketName, dirNameB)}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, IsNil)

	// dirname/dirname
	dirNameC := randLowStr(12) + "/" + randLowStr(12)
	mkArgs = []string{CloudURLToString(bucketName, dirNameC)}
	_, err = cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, IsNil)

	//check the exist dirname
	dirList := []string{dirNameA, dirNameB, dirNameC}
	for _, v := range dirList {
		if !strings.HasSuffix(v, "/") {
			v += "/"
		}
		body, err := bucket.GetObject(v)
		c.Assert(err, IsNil)

		data, err := ioutil.ReadAll(body)
		body.Close()
		c.Assert(err, IsNil)
		c.Assert(string(data), Equals, "")
	}
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMkdirAllEncodingError(c *C) {
	// mkdir command test
	var str string
	strEncode := "url"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"encodingType":    &strEncode,
	}

	// url encoding error
	mkArgs := []string{"http%3a%3a%2%2faaada%5ct"}
	_, err := cm.RunCommand("mkdir", mkArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestMkdirHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"mkdir"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
