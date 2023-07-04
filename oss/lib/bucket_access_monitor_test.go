package lib

import (
	"encoding/xml"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
	"os"
)

func (s *OssutilCommandSuite) TestAccessMonitorHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"access-monitor"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestPutBucketAccessMonitorError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	accessFileName := "access-monitor" + randLowStr(12)

	// access monitor command test
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
	accessArgs := []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err := cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	accessArgs = []string{"http://mybucket", accessFileName}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	accessArgs = []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(accessFileName)
	accessArgs = []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	// local file is dir
	err = os.MkdirAll(accessFileName, 0755)
	c.Assert(err, IsNil)
	accessArgs = []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)
	os.Remove(accessFileName)

	//local file is empty
	s.createFile(accessFileName, "", c)
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)
	os.Remove(accessFileName)

	//local file is not xml file
	s.createFile(accessFileName, "aaa", c)
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)
	os.Remove(accessFileName)

	// StorageURLFromString error
	accessArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	accessArgs = []string{"oss:///"}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	accessArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	accessArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("access-monitor", accessArgs, options)
	c.Assert(err, NotNil)

	os.Remove(accessFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPutBucketAccessMonitor(c *C) {
	accessXml := `<?xml version="1.0" encoding="UTF-8"?>
<AccessMonitorConfiguration>
  <Status>Enabled</Status>
</AccessMonitorConfiguration>`

	accessConfigSrc := oss.PutBucketAccessMonitor{}
	err := xml.Unmarshal([]byte(accessXml), &accessConfigSrc)
	c.Assert(err, IsNil)

	accessFileName := randLowStr(12)
	s.createFile(accessFileName, accessXml, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// access monitor command test
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

	command := "access-monitor"
	accessArgs := []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	// check,get access monitor
	accessDownName := accessFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	accessArgs = []string{CloudURLToString(bucketName, ""), accessDownName}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	// check access monitor DownName
	_, err = os.Stat(accessDownName)
	c.Assert(err, IsNil)

	accessBody := s.readFile(accessDownName, c)

	var out oss.GetBucketAccessMonitorResult
	err = xml.Unmarshal([]byte(accessBody), &out)
	c.Assert(err, IsNil)

	c.Assert(accessConfigSrc.Status, Equals, out.Status)

	os.Remove(accessFileName)
	os.Remove(accessDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestGetBucketAccessMonitorConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	accessXml := `<?xml version="1.0" encoding="UTF-8"?>
<AccessMonitorConfiguration>
  <Status>Enabled</Status>
</AccessMonitorConfiguration>`

	accessFileName := inputFileName + randLowStr(5)
	s.createFile(accessFileName, accessXml, c)

	// lifecycle command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	command := "access-monitor"

	accessArgs := []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err := cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	// get access monitor
	accessDownName := accessFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	accessArgs = []string{CloudURLToString(bucketName, ""), accessDownName}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	accessArgs = []string{CloudURLToString(bucketName, ""), accessDownName}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	accessArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	os.Remove(accessFileName)
	os.Remove(accessDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestAccessMonitorWithPutLifecycle(c *C) {
	accessXml := `<?xml version="1.0" encoding="UTF-8"?>
<AccessMonitorConfiguration>
  <Status>Enabled</Status>
</AccessMonitorConfiguration>`

	accessFileName := inputFileName + randLowStr(5)
	s.createFile(accessFileName, accessXml, c)

	bucketName := bucketNamePrefix + randLowStr(5)
	s.putBucket(bucketName, c)

	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	command := "access-monitor"
	accessArgs := []string{CloudURLToString(bucketName, ""), accessFileName}
	_, err := cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	lifecycleXml := `<?xml version="1.0" encoding="UTF-8"?>
<LifecycleConfiguration>
	<Rule>
    <ID>mtime transition1</ID>
    <Prefix>logs1/</Prefix>
    <Status>Enabled</Status>
    <Transition>
      <Days>30</Days>
      <StorageClass>IA</StorageClass>
    </Transition>
  </Rule>
  <Rule>
    <ID>mtime transition2</ID>
    <Prefix>logs2/</Prefix>
    <Status>Enabled</Status>
    <Transition>
      <Days>30</Days>
      <StorageClass>IA</StorageClass>
      <IsAccessTime>false</IsAccessTime>
    </Transition>
  </Rule>
  <Rule>
    <ID>atime transition1</ID>
    <Prefix>logs3/</Prefix>
    <Status>Enabled</Status>
    <Transition>
      <Days>30</Days>
      <StorageClass>IA</StorageClass>
      <IsAccessTime>true</IsAccessTime>
      <ReturnToStdWhenVisit>false</ReturnToStdWhenVisit>
    </Transition>
  </Rule>
  <Rule>
    <ID>atime transition2</ID>
    <Prefix>logs4/</Prefix>
    <Status>Enabled</Status>
    <Transition>
      <Days>30</Days>
      <StorageClass>IA</StorageClass>
      <IsAccessTime>true</IsAccessTime>
      <ReturnToStdWhenVisit>true</ReturnToStdWhenVisit>
    </Transition>
  </Rule>
  <Rule>
    <ID>atime transition3</ID>
    <Prefix>logs5/</Prefix>
    <Status>Enabled</Status>
    <NoncurrentVersionTransition>
      <NoncurrentDays>10</NoncurrentDays>
      <StorageClass>IA</StorageClass>
      <IsAccessTime>true</IsAccessTime>
      <ReturnToStdWhenVisit>false</ReturnToStdWhenVisit>
    </NoncurrentVersionTransition>
  </Rule>
</LifecycleConfiguration>`

	rulesConfigSrc := oss.LifecycleConfiguration{}
	err = xml.Unmarshal([]byte(lifecycleXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	lifecycleFileName := randLowStr(12)
	s.createFile(lifecycleFileName, lifecycleXml, c)

	lifecycleArgs := []string{CloudURLToString(bucketName, ""), lifecycleFileName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// get lifecycle
	lifecycleDownName := lifecycleFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	// check access monitor DownName
	_, err = os.Stat(lifecycleDownName)
	c.Assert(err, IsNil)

	accessBody := s.readFile(lifecycleDownName, c)

	var rulesConfigDest oss.GetBucketLifecycleResult
	err = xml.Unmarshal([]byte(accessBody), &rulesConfigDest)
	c.Assert(err, IsNil)

	c.Assert(len(rulesConfigSrc.Rules), Equals, len(rulesConfigDest.Rules))
	c.Assert(*rulesConfigSrc.Rules[2].Transitions[0].IsAccessTime, Equals, *rulesConfigDest.Rules[2].Transitions[0].IsAccessTime)
	c.Assert(*rulesConfigSrc.Rules[3].Transitions[0].ReturnToStdWhenVisit, Equals, *rulesConfigDest.Rules[3].Transitions[0].ReturnToStdWhenVisit)

	strMethod = "put"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	accessXmlOne := `<?xml version="1.0" encoding="UTF-8"?>
<AccessMonitorConfiguration>
  <Status>Disabled</Status>
</AccessMonitorConfiguration>`

	accessFileNameOne := inputFileName + randLowStr(5)
	s.createFile(accessFileNameOne, accessXmlOne, c)

	command = "access-monitor"
	accessArgs = []string{CloudURLToString(bucketName, ""), accessFileNameOne}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, NotNil)

	// delete lifecycleDownName
	strMethod = "delete"
	lifecycleArgs = []string{CloudURLToString(bucketName, ""), lifecycleDownName}
	_, err = cm.RunCommand("lifecycle", lifecycleArgs, options)
	c.Assert(err, IsNil)

	accessXmlOne = `<?xml version="1.0" encoding="UTF-8"?>
<AccessMonitorConfiguration>
  <Status>Disabled</Status>
</AccessMonitorConfiguration>`

	accessFileNameOne = inputFileName + randLowStr(5)
	s.createFile(accessFileNameOne, accessXmlOne, c)

	command = "access-monitor"
	strMethod = "put"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}
	accessArgs = []string{CloudURLToString(bucketName, ""), accessFileNameOne}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	accessDownName := accessFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	accessArgs = []string{CloudURLToString(bucketName, ""), accessDownName}
	_, err = cm.RunCommand(command, accessArgs, options)
	c.Assert(err, IsNil)

	// check access monitor DownName
	_, err = os.Stat(accessDownName)
	c.Assert(err, IsNil)

	accessBody = s.readFile(accessDownName, c)

	var out oss.GetBucketAccessMonitorResult
	err = xml.Unmarshal([]byte(accessBody), &out)
	c.Assert(err, IsNil)

	c.Assert(out.Status, Equals, "Disabled")

	os.Remove(accessFileName)
	os.Remove(lifecycleDownName)
	os.Remove(lifecycleFileName)
	os.Remove(accessFileNameOne)
	os.Remove(accessDownName)
	s.removeBucket(bucketName, true, c)

}
