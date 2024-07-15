package lib

import (
	"encoding/xml"
	"os"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestCorsOptionsSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	corsXml := `<?xml version="1.0" encoding="UTF-8"?>
    <CORSConfiguration>
      <CORSRule>
          <AllowedOrigin>www.aliyun.com</AllowedOrigin>
          <AllowedMethod>PUT</AllowedMethod>
          <AllowedHeader>x-oss-meta-author</AllowedHeader>
          <ExposeHeader>x-oss-meta-name</ExposeHeader>
          <MaxAgeSeconds>10001</MaxAgeSeconds>
      </CORSRule>
    </CORSConfiguration>`

	rulesConfigSrc := oss.CORSXML{}
	err := xml.Unmarshal([]byte(corsXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	corsFileName := "ossutil_test." + randLowStr(12)
	s.createFile(corsFileName, corsXml, c)

	// cors command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	corsArgs := []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// cors-options success
	strOrigin := "www.aliyun.com"
	strAcrHeaders := "x-oss-meta-author"
	options["origin"] = &strOrigin
	options["acrHeaders"] = &strAcrHeaders
	options["acrMethod"] = &strMethod
	delete(options, "method")

	corsArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, IsNil)

	// cors-options error
	strOrigin = "www.test.com"
	_, err = cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, NotNil)

	os.Remove(corsFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsOptionsError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	corsFileName := "ossutil_test_corsfile_" + randLowStr(12)

	// cors command test
	var str string
	strMethod := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"acrMethod":       &strMethod,
	}

	// method is empty
	corsArgs := []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err := cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	corsArgs = []string{"http://mybucket", corsFileName}
	_, err = cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, NotNil)

	// StorageURLFromString error
	corsArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("cors-options", corsArgs, options)
	c.Assert(err, NotNil)

	os.Remove(corsFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsOptionsEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"acrMethod":       &strMethod,
	}

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("cors-options", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsOptionsHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"cors-options"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
