package lib

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestUserQosGetError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// command test
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
	qosArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod = "gett"
	_, err = cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUserQosOptionsEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	strMethod := "get"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	// oss client error
	// set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	qosArgs := []string{}
	_, err = cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUserQosGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// user-qos command test
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

	qosArgs := []string{}
	_, err := cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, NotNil)

	// get qos
	qosDownName := "ossutil-test-file-" + randLowStr(12) + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	qosArgs = []string{}
	_, err = cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, IsNil)

	qosArgs = []string{qosDownName}
	_, err = cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, IsNil)

	qosArgs = []string{qosDownName}
	_, err = cm.RunCommand("user-qos", qosArgs, options)
	c.Assert(err, IsNil)

	os.Remove(qosDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestUserQosHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"user-qos"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
