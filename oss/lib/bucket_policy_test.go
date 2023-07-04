package lib

import (
	"os"
	"time"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestPolicyPutSuccess(c *C) {
	policyJson := ` 
    {
        "Version": "1",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ram:ListObjects"
                ],
                "Resource": [
                    "*"
                ],
                "Condition": {}
            }
        ]
    }`

	policyFileName := randLowStr(12)
	s.createFile(policyFileName, policyJson, c)

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// policy command test
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

	policyArgs := []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err := cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	// check,get policy
	policyDownName := policyFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	// check policyDownName
	_, err = os.Stat(policyDownName)
	c.Assert(err, IsNil)

	policyBody := s.readFile(policyDownName, c)
	c.Assert(len(policyBody) > 0, Equals, true)

	os.Remove(policyFileName)
	os.Remove(policyDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPolicyPutError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	policyFileName := "policy-file" + randLowStr(12)

	// policy command test
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
	policyArgs := []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err := cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	//method is error
	strMethod = "puttt"
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	// cloudurl is error
	strMethod = "put"
	policyArgs = []string{"http://mybucket", policyFileName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	// local file is emtpy
	policyArgs = []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	//local file is not exist
	os.Remove(policyFileName)
	policyArgs = []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	// localfile is dir
	err = os.MkdirAll(policyFileName, 0755)
	c.Assert(err, IsNil)
	policyArgs = []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)
	os.Remove(policyFileName)

	//local file is emtpy
	s.createFile(policyFileName, "", c)
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)
	os.Remove(policyFileName)

	//local file is not xml file
	s.createFile(policyFileName, "aaa", c)
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)
	os.Remove(policyFileName)

	// StorageURLFromString error
	policyArgs = []string{"oss:///1.jpg"}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	// bucketname is error
	policyArgs = []string{"oss:///"}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	//missing parameter
	policyArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	// bucketname not exist
	policyArgs = []string{CloudURLToString("my-bucket", "")}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	os.Remove(policyFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPolicyOptionsEmptyEndpoint(c *C) {
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
	_, err := cm.RunCommand("bucket-policy", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPolicyGetConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	policyJson := ` 
    {
        "Version": "1",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ram:ListObjects"
                ],
                "Resource": [
                    "*"
                ],
                "Condition": {}
            }
        ]
    }`

	policyFileName := randLowStr(12)
	s.createFile(policyFileName, policyJson, c)

	// policy command test
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

	policyArgs := []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err := cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	// get policy
	policyDownName := policyFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	policyArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	os.Remove(policyFileName)
	os.Remove(policyDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPolicyDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	policyJson := ` 
    {
        "Version": "1",
        "Statement": [
            {
                "Effect": "Allow",
                "Action": [
                    "ram:ListObjects"
                ],
                "Resource": [
                    "*"
                ],
                "Condition": {}
            }
        ]
    }`

	policyFileName := randLowStr(12)
	s.createFile(policyFileName, policyJson, c)

	// policy command test
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

	policyArgs := []string{CloudURLToString(bucketName, ""), policyFileName}
	_, err := cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	// get policy
	policyDownName := policyFileName + "-down"
	strMethod = "get"
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	// check policyDownName
	_, err = os.Stat(policyDownName)
	c.Assert(err, IsNil)
	os.Remove(policyDownName)

	// delete policyDownName
	strMethod = "delete"
	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	policyArgs = []string{CloudURLToString(bucketName, ""), policyDownName}
	_, err = cm.RunCommand("bucket-policy", policyArgs, options)
	c.Assert(err, NotNil)

	os.Remove(policyFileName)
	os.Remove(policyDownName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestPolicyHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"bucket-policy"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
