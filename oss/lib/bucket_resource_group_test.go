package lib

import (
	"encoding/xml"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
	"os"
)

func (s *OssutilCommandSuite) TestResourceGroupHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"resource-group"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestPutBucketResourceGroupError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	resourceFileName := "resource-group" + randLowStr(12)

	// resource group command test
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
	resourceArgs := []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err := cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	options["method"] = &strMethod
	resourceArgs = []string{"http://mybucket", resourceFileName}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(resourceFileName)
	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	// local file is dir
	err = os.MkdirAll(resourceFileName, 0755)
	c.Assert(err, IsNil)
	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)
	os.Remove(resourceFileName)

	//local file is empty
	s.createFile(resourceFileName, "", c)
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)
	os.Remove(resourceFileName)

	//local file is not xml file
	s.createFile(resourceFileName, "aaa", c)
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)
	os.Remove(resourceFileName)

	// StorageURLFromString error
	resourceArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	resourceArgs = []string{"oss:///"}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	resourceArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	resourceArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("resource-group", resourceArgs, options)
	c.Assert(err, NotNil)

	os.Remove(resourceFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPutBucketResourceGroup(c *C) {
	resourceXml := `<?xml version="1.0" encoding="UTF-8"?>
<BucketResourceGroupConfiguration>
  <ResourceGroupId>rg-acfmy7mo47b3adq</ResourceGroupId>
</BucketResourceGroupConfiguration>`

	accessConfigSrc := oss.PutBucketResourceGroup{}
	err := xml.Unmarshal([]byte(resourceXml), &accessConfigSrc)
	c.Assert(err, IsNil)

	resourceFileName := randLowStr(12)
	s.createFile(resourceFileName, resourceXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// resource group command test
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

	command := "resource-group"
	resourceArgs := []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	// check,get resource group
	resourceDownName := resourceFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceDownName}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	// check resource group DownName
	_, err = os.Stat(resourceDownName)
	c.Assert(err, IsNil)

	accessBody := s.readFile(resourceDownName, c)

	var out oss.GetBucketResourceGroupResult
	err = xml.Unmarshal([]byte(accessBody), &out)
	c.Assert(err, IsNil)

	c.Assert(accessConfigSrc.ResourceGroupId, Equals, out.ResourceGroupId)

	os.Remove(resourceFileName)
	os.Remove(resourceDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetBucketResourceGroupConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	resourceFileName := inputFileName + randLowStr(5)
	// get resource group
	resourceDownName := resourceFileName + "-down"
	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	command := "resource-group"
	resourceArgs := []string{CloudURLToString(bucketName, ""), resourceDownName}
	_, err := cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)
	accessBody := s.readFile(resourceDownName, c)

	var out oss.GetBucketResourceGroupResult
	err = xml.Unmarshal([]byte(accessBody), &out)
	c.Assert(err, IsNil)
	id := out.ResourceGroupId

	os.Remove(resourceDownName)

	resourceXml := `<?xml version="1.0" encoding="UTF-8"?>
<BucketResourceGroupConfiguration>
  <ResourceGroupId>` + id + `</ResourceGroupId>
</BucketResourceGroupConfiguration>`

	s.createFile(resourceFileName, resourceXml, c)

	// resource group command test
	options[OptionMethod] = &strMethod

	command = "resource-group"
	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceFileName}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	// get resource group
	strMethod = "get"
	options[OptionMethod] = &strMethod

	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceDownName}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	resourceArgs = []string{CloudURLToString(bucketName, ""), resourceDownName}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	resourceArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand(command, resourceArgs, options)
	c.Assert(err, IsNil)

	os.Remove(resourceFileName)
	os.Remove(resourceDownName)
	s.removeBucket(bucketName, true, c)
}
