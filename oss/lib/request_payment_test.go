package lib

import (
	"os"
	"time"

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

	time.Sleep(5 * time.Second)

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
	time.Sleep(5 * time.Second)

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

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// request-payment command test
	var str string
	strMethod := "put"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
	}

	requestArgs := []string{CloudURLToString(bucketName, ""), "Requester"}
	_, err := cm.RunCommand("request-payment", requestArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRequestPaymentGetEmptyEndpoint(c *C) {
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
	_, err := cm.RunCommand("request-payment", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
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
