package lib

import (
	"encoding/xml"
	"os"
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketQosPutSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	qosXml := `<?xml version="1.0" encoding="UTF-8"?>
    <QoSConfiguration>
        <TotalUploadBandwidth>9</TotalUploadBandwidth>
        <IntranetUploadBandwidth>-1</IntranetUploadBandwidth>
        <ExtranetUploadBandwidth>-1</ExtranetUploadBandwidth>
        <TotalDownloadBandwidth>5</TotalDownloadBandwidth>
        <IntranetDownloadBandwidth>-1</IntranetDownloadBandwidth>
        <ExtranetDownloadBandwidth>-1</ExtranetDownloadBandwidth>
        <TotalQps>1000</TotalQps>
        <IntranetQps>-1</IntranetQps>
        <ExtranetQps>-1</ExtranetQps>
    </QoSConfiguration>`

	rulesConfigSrc := oss.BucketQoSConfiguration{}
	err := xml.Unmarshal([]byte(qosXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	qosFileName := randLowStr(12)
	s.createFile(qosFileName, qosXml, c)

	// qos command test
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

	qosArgs := []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	// wait a moment for configuration effect
	time.Sleep(time.Second)

	// check,get qos
	qosDownName := qosFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	// check qosDownName
	_, err = os.Stat(qosDownName)
	c.Assert(err, IsNil)

	qosBody := s.readFile(qosDownName, c)

	rulesConfigDest := oss.BucketQoSConfiguration{}
	err = xml.Unmarshal([]byte(qosBody), &rulesConfigDest)
	c.Assert(err, IsNil)
	c.Assert(*rulesConfigSrc.TotalUploadBandwidth, Equals, *rulesConfigDest.TotalUploadBandwidth)
	c.Assert(*rulesConfigSrc.IntranetUploadBandwidth, Equals, *rulesConfigDest.IntranetUploadBandwidth)
	c.Assert(*rulesConfigSrc.ExtranetUploadBandwidth, Equals, *rulesConfigDest.ExtranetUploadBandwidth)
	c.Assert(*rulesConfigSrc.TotalDownloadBandwidth, Equals, *rulesConfigDest.TotalDownloadBandwidth)
	c.Assert(*rulesConfigSrc.IntranetDownloadBandwidth, Equals, *rulesConfigDest.IntranetDownloadBandwidth)
	c.Assert(*rulesConfigSrc.ExtranetDownloadBandwidth, Equals, *rulesConfigDest.ExtranetDownloadBandwidth)
	c.Assert(*rulesConfigSrc.TotalQPS, Equals, *rulesConfigDest.TotalQPS)
	c.Assert(*rulesConfigSrc.IntranetQPS, Equals, *rulesConfigDest.IntranetQPS)
	c.Assert(*rulesConfigSrc.ExtranetQPS, Equals, *rulesConfigDest.ExtranetQPS)

	os.Remove(qosFileName)
	os.Remove(qosDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketQosPutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	qosFileName := "qos-file" + randLowStr(12)

	// qos command test
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
	qosArgs := []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err := cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	qosArgs = []string{"http://mybucket", qosFileName}
	_, err = cm.RunCommand("qbucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	qosArgs = []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(qosFileName)
	qosArgs = []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(qosFileName, 0755)
	c.Assert(err, IsNil)
	qosArgs = []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)
	os.Remove(qosFileName)

	//local file is emtpy
	s.createFile(qosFileName, "", c)
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)
	os.Remove(qosFileName)

	//local file is not xml file
	s.createFile(qosFileName, "aaa", c)
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)
	os.Remove(qosFileName)

	// StorageURLFromString error
	qosArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	qosArgs = []string{"oss:///"}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	qosArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	qosArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	os.Remove(qosFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketQosOptionsEmptyEndpoint(c *C) {
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
		"method":          &strMethod,
	}

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-qos", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketQosGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	qosXml := `<?xml version="1.0" encoding="UTF-8"?>
    <QoSConfiguration>
        <TotalUploadBandwidth>9</TotalUploadBandwidth>
        <IntranetUploadBandwidth>-1</IntranetUploadBandwidth>
        <ExtranetUploadBandwidth>-1</ExtranetUploadBandwidth>
        <TotalDownloadBandwidth>5</TotalDownloadBandwidth>
        <IntranetDownloadBandwidth>-1</IntranetDownloadBandwidth>
        <ExtranetDownloadBandwidth>-1</ExtranetDownloadBandwidth>
        <TotalQps>1000</TotalQps>
        <IntranetQps>-1</IntranetQps>
        <ExtranetQps>-1</ExtranetQps>
    </QoSConfiguration>`

	qosFileName := randLowStr(12)
	s.createFile(qosFileName, qosXml, c)

	// qos command test
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

	qosArgs := []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err := cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// get qos
	qosDownName := qosFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	qosArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	os.Remove(qosFileName)
	os.Remove(qosDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketQosDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)
	qosXml := `<?xml version="1.0" encoding="UTF-8"?>
    <QoSConfiguration>
        <TotalUploadBandwidth>9</TotalUploadBandwidth>
        <IntranetUploadBandwidth>-1</IntranetUploadBandwidth>
        <ExtranetUploadBandwidth>-1</ExtranetUploadBandwidth>
        <TotalDownloadBandwidth>9</TotalDownloadBandwidth>
        <IntranetDownloadBandwidth>-1</IntranetDownloadBandwidth>
        <ExtranetDownloadBandwidth>-1</ExtranetDownloadBandwidth>
        <TotalQps>1000</TotalQps>
        <IntranetQps>-1</IntranetQps>
        <ExtranetQps>-1</ExtranetQps>
    </QoSConfiguration>`

	qosFileName := randLowStr(12)
	s.createFile(qosFileName, qosXml, c)

	// qos command test
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

	qosArgs := []string{CloudURLToString(bucketName, ""), qosFileName}
	_, err := cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// get qos
	qosDownName := qosFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)

	// check qosDownName
	_, err = os.Stat(qosDownName)
	c.Assert(err, IsNil)
	os.Remove(qosDownName)

	// delete qosDownName
	strMethod = "delete"
	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// get again
	strMethod = "get"
	qosArgs = []string{CloudURLToString(bucketName, ""), qosDownName}
	_, err = cm.RunCommand("bucket-qos", qosArgs, options)
	c.Assert(err, NotNil)

	os.Remove(qosFileName)
	os.Remove(qosDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketQosHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"bucket-qos"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
