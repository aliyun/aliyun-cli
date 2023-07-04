package lib

import (
	"os"
	"time"

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
	time.Sleep(time.Second * 3)

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
	time.Sleep(time.Second * 3)

	// get again
	strMethod = "get"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "AES256")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSMasterKeyID, Equals, "")

	// set sm4
	strMethod = "put"
	strSSEAlgorithm = "SM4"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// get again
	strMethod = "get"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "SM4")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSMasterKeyID, Equals, "")

	// set kms sm4
	strMethod = "put"
	strSSEAlgorithm = "KMS"
	strDataEncryption := "SM4"
	options["KMSDataEncryption"] = &strDataEncryption
	time.Sleep(time.Second * 3)

	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)

	// get again
	strMethod = "get"
	_, err = cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.SSEAlgorithm, Equals, "KMS")
	c.Assert(bucketEncryptionCommand.encryptionResult.SSEDefault.KMSDataEncryption, Equals, "SM4")

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
	time.Sleep(time.Second * 3)

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

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// referer command test
	var str string
	strMethod := "put"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketEncryptionGetEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	var str string
	strMethod := "get"
	strSSEAlgorithm := "KMS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
		"method":          &strMethod,
		"SSEAlgorithm":    &strSSEAlgorithm,
	}

	encryptionArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-Encryption", encryptionArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)

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
