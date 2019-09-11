package lib

import (
	"encoding/xml"
	"os"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestCorsPutSuccess(c *C) {
	corsXml := `<?xml version="1.0" encoding="UTF-8"?>
    <CORSConfiguration>
      <CORSRule>
          <AllowedOrigin>www.aliyun.com</AllowedOrigin>
          <AllowedMethod>PUT</AllowedMethod>
          <MaxAgeSeconds>10001</MaxAgeSeconds>
      </CORSRule>
    </CORSConfiguration>`

	rulesConfigSrc := oss.CORSXML{}
	err := xml.Unmarshal([]byte(corsXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	corsFileName := randLowStr(12)
	s.createFile(corsFileName, corsXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

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

	// check,get cors
	corsDownName := corsFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// check corsDownName
	_, err = os.Stat(corsDownName)
	c.Assert(err, IsNil)

	corsBody := s.readFile(corsDownName, c)

	rulesConfigDest := oss.CORSXML{}
	err = xml.Unmarshal([]byte(corsBody), &rulesConfigDest)
	c.Assert(err, IsNil)
	c.Assert(rulesConfigSrc.CORSRules[0].MaxAgeSeconds, Equals, rulesConfigDest.CORSRules[0].MaxAgeSeconds)

	os.Remove(corsFileName)
	os.Remove(corsDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsPutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	corsFileName := "corsfile-" + randLowStr(12)

	// cors command test
	var str string
	strMethod := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// method is empty
	corsArgs := []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err := cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	corsArgs = []string{"http://mybucket", corsFileName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	corsArgs = []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(corsFileName)
	corsArgs = []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(corsFileName, 0755)
	c.Assert(err, IsNil)
	corsArgs = []string{CloudURLToString(bucketName, ""), corsFileName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)
	os.Remove(corsFileName)

	//local file is emtpy
	s.createFile(corsFileName, "", c)
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)
	os.Remove(corsFileName)

	//local file is not xml file
	s.createFile(corsFileName, "aaa", c)
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)
	os.Remove(corsFileName)

	// StorageURLFromString error
	corsArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	corsArgs = []string{"oss:///"}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	corsArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	corsArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	os.Remove(corsFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsGetConfirm(c *C) {
	corsXml := `<?xml version="1.0" encoding="UTF-8"?>
    <CORSConfiguration>
      <CORSRule>
          <AllowedOrigin>www.aliyun.com</AllowedOrigin>
          <AllowedMethod>PUT</AllowedMethod>
          <MaxAgeSeconds>10001</MaxAgeSeconds>
      </CORSRule>
    </CORSConfiguration>`

	corsFileName := randLowStr(12)
	s.createFile(corsFileName, corsXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

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
	_, err := cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// get cors
	corsDownName := corsFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	corsArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	os.Remove(corsFileName)
	os.Remove(corsDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsDelete(c *C) {
	corsXml := `<?xml version="1.0" encoding="UTF-8"?>
    <CORSConfiguration>
      <CORSRule>
          <AllowedOrigin>www.aliyun.com</AllowedOrigin>
          <AllowedMethod>PUT</AllowedMethod>
          <MaxAgeSeconds>10001</MaxAgeSeconds>
      </CORSRule>
    </CORSConfiguration>`

	corsFileName := randLowStr(12)
	s.createFile(corsFileName, corsXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

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
	_, err := cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// get cors
	corsDownName := corsFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// check corsDownName
	_, err = os.Stat(corsDownName)
	c.Assert(err, IsNil)
	os.Remove(corsDownName)

	// delete corsDownName
	strMethod = "delete"
	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	corsArgs = []string{CloudURLToString(bucketName, ""), corsDownName}
	_, err = cm.RunCommand("cors", corsArgs, options)
	c.Assert(err, NotNil)

	os.Remove(corsFileName)
	os.Remove(corsDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestCorsHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"cors"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
