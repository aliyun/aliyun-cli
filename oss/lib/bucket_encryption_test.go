package lib

import (
	"io/ioutil"
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketEncryptionPutSuccess(c *C) {
	// put Encryption
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// Encryption command test
	var str string
	strMethod := "put"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)

	// check,get encryption
	strMethod = "get"
	encryptionArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "KMS")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSMasterKeyID, Equals, "")

	// get bucket stat
	bucketStat := s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatSSEAlgorithm], Equals, "KMS")

	// set aes256
	strMethod = "put"
	strSSEAlgorithm = "AES256"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "AES256")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSMasterKeyID, Equals, "")

	// put error
	strMethod = "put"
	strSSEAlgorithm = "BES356"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	// put error
	strMethod = "put"
	strSSEAlgorithm = "AES256"
	strKMSMasterKeyID := "123"
	options["KMSMasterKeyID"] = &strKMSMasterKeyID
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketEncryptionDeleteSuccess(c *C) {
	// put Encryption
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// Encryption command test
	var str string
	strMethod := "put"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)

	// check,get encryption
	strMethod = "get"
	encryptionArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "KMS")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSMasterKeyID, Equals, "")

	// delete encryption
	strMethod = "delete"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)

	//get encryption:error
	strMethod = "get"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketEncryptionError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// bucket-Encryption command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// method is empty
	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod := "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)

}

func (s *OssutilCommandSuite) TestBucketEncryptionPutEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// referer command test
	var str string
	strMethod := "put"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	// oss client error
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketEncryptionGetEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	strMethod := "get"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	// oss client error
	//set endpoint emtpy
	oldConfigStr, err := ioutil.ReadFile(configFile)
	c.Assert(err, IsNil)
	fd, _ := os.OpenFile(configFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	configStr := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	fd.WriteString(configStr)
	fd.Close()

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-Encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	err = ioutil.WriteFile(configFile, []byte(oldConfigStr), 0664)
	c.Assert(err, IsNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketEncryptionHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"bucket-encryption"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
