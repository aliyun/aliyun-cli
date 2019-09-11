package lib

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestRequestPaymentPutSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// request command test
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

	requestArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, IsNil)
	c.Assert(requestPaymentCommand.paymentResult.Payer, Equals, "BucketOwner")

	// set bucket request enabled
	strMethod = "put"
	requestArgs = []string{CloudURLToString(bucketName, ""), "Requester"}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, IsNil)

	// check
	strMethod = "get"
	requestArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, IsNil)
	c.Assert(requestPaymentCommand.paymentResult.Payer, Equals, "Requester")

	// set bucket request suspend
	strMethod = "put"
	requestArgs = []string{CloudURLToString(bucketName, ""), "BucketOwner"}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, IsNil)

	// check
	strMethod = "get"
	requestArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, IsNil)
	c.Assert(requestPaymentCommand.paymentResult.Payer, Equals, "BucketOwner")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRequestPaymentError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// request-payment command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// method is empty
	requestArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod := "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	// args is empty
	strMethod = "put"
	requestArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	//value is error
	requestArgs = []string{CloudURLToString(bucketName, ""), "Bucket-Owner"}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRequestPaymentPutEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// request-payment command test
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

	// oss client error
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	requestArgs := []string{CloudURLToString(bucketName, ""), "Requester"}
	_, err = cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRequestPaymentGetEmptyEndpoint(c *C) {
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
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("request-payment", versioingArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRequestPaymentHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"request-payment"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
