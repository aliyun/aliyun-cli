package lib

import (
	"fmt"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"os"
	"strings"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestMakeBucket(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// put bucket already exists
	s.putBucket(bucketName, c)

	// get bucket stat
	bucketStat := s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatName], Equals, bucketName)
	c.Assert(bucketStat[StatACL], Equals, "private")

	s.removeBucket(bucketName, true, c)

	bucketName = bucketNamePrefix + randLowStr(10)
	showElapse, err := s.putBucketWithACL(bucketName, "public-read-write")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	bucketStat = s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatName], Equals, bucketName)
	c.Assert(bucketStat[StatACL], Equals, "public-read-write")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMakeBucketErrorName(c *C) {
	for _, bucketName := range []string{"中文测试", "a"} {
		command := "mb"
		args := []string{CloudURLToString(bucketName, "")}
		str := ""
		options := OptionMapType{
			"endpoint":        &str,
			"accessKeyID":     &str,
			"accessKeySecret": &str,
			"stsToken":        &str,
			"configFile":      &configFile,
		}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(err, NotNil)
		c.Assert(showElapse, Equals, false)

		showElapse, err = s.rawGetStat(bucketName, "")
		c.Assert(err, NotNil)
		c.Assert(showElapse, Equals, false)
	}
}

func (s *OssutilCommandSuite) TestMakeBucketErrorACL(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	for _, language := range []string{DefaultLanguage, EnglishLanguage, LEnglishLanguage, "unknown"} {
		for _, acl := range []string{"default", "def", "erracl"} {
			showElapse, err := s.rawPutBucketWithACLLanguage([]string{CloudURLToString(bucketName, "")}, acl, language)
			c.Assert(err, NotNil)
			c.Assert(showElapse, Equals, false)

			showElapse, err = s.rawGetStat(bucketName, "")
			c.Assert(err, NotNil)
			c.Assert(showElapse, Equals, false)
		}
	}
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMakeBucketErrorOption(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	ok := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"shortFormat":     &ok,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestErrMakeBucket(c *C) {
	acl := "private"
	showElapse, err := s.rawPutBucketWithACL([]string{"os://"}, acl)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawPutBucketWithACL([]string{CloudURLToString("", "")}, acl)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestMakeBucketIDKey(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	cfile := randStr(10)
	data := fmt.Sprintf("[Credentials]\nendpoint=%s\naccessKeyID=%s\naccessKeySecret=%s\n[Bucket-Endpoint]\n%s=%s", "abc", "def", "ghi", bucketName, "abc")
	s.createFile(cfile, data, c)

	command := "mb"
	str := ""
	args := []string{CloudURLToString(bucketName, "")}
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	options = OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &str,
		"configFile":      &cfile,
	}
	showElapse, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	os.Remove(cfile)

	s.removeBucket(bucketName, false, c)
}

func (s *OssutilCommandSuite) TestMakeBucketStorageClass(c *C) {
	result := []string{StorageStandard, StorageIA, StorageArchive}
	for i, class := range []string{StorageStandard, StorageIA, StorageArchive, strings.ToUpper(StorageStandard), strings.ToLower(StorageIA), "arCHIvE"} {
		bucketName := bucketNamePrefix + randLowStr(10)
		err := s.initPutBucketWithStorageClass([]string{CloudURLToString(bucketName, "")}, class)
		c.Assert(err, IsNil)
		err = makeBucketCommand.RunCommand()
		c.Assert(err, IsNil)

		bucketStat := s.getStat(bucketName, "", c)
		c.Assert(bucketStat[StatName], Equals, bucketName)
		c.Assert(bucketStat[StatStorageClass], Equals, result[i%3])

		// delete bucket
		s.removeBucket(bucketName, true, c)
	}

	// test error make bucket
	err := s.initPutBucketWithStorageClass([]string{CloudURLToString("", "")}, StorageStandard)
	c.Assert(err, IsNil)
	err = makeBucketCommand.RunCommand()
	c.Assert(err, NotNil)

	// test error make bucket
	err = s.initPutBucketWithStorageClass([]string{CloudURLToString("", "abc")}, StorageStandard)
	c.Assert(err, IsNil)
	err = makeBucketCommand.RunCommand()
	c.Assert(err, NotNil)

	// test error make bucket
	bucketName := bucketNamePrefix + randLowStr(10)
	err = s.initPutBucketWithStorageClass([]string{CloudURLToString(bucketName, "abc")}, StorageStandard)
	c.Assert(err, IsNil)
	err = makeBucketCommand.RunCommand()
	c.Assert(err, NotNil)

	_, err = s.rawGetStat(bucketName, "")
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestMbCreateBucketWithRedundancy(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	command := "mb"
	args := []string{CloudURLToString(bucketName, "")}
	str := ""
	strRedundancy := "ZRS"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"redundancyType":  &strRedundancy,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	bucketStat := s.getStat(bucketName, "", c)
	fmt.Println(bucketStat)
	c.Assert(bucketStat[StatRedundancyType], Equals, strRedundancy)
	s.removeBucket(bucketName, true, c)

	// redundancyType is error
	bucketName = bucketNamePrefix + randLowStr(10)
	args = []string{CloudURLToString(bucketName, "")}
	strRedundancy = "LLL"
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// create without redundancyType
	bucketName = bucketNamePrefix + randLowStr(10)
	args = []string{CloudURLToString(bucketName, "")}
	delete(options, "redundancyType")
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	bucketStat = s.getStat(bucketName, "", c)
	c.Assert(bucketStat[StatRedundancyType], Equals, "LRS")
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestMbCreateBucketWithConfigFile(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	inputFile := "test-ossutil-file-" + randLowStr(5)
	command := "mb"
	args := []string{CloudURLToString(bucketName, ""), inputFile}
	str := ""
	retryCount := "1"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"retryTimes":      &retryCount,
	}

	xmlBody := `
	<?xml version="1.0" encoding="UTF-8"?>
	<CreateBucketConfiguration>
		<StorageClass>IA</StorageClass>
	</CreateBucketConfiguration>
	`
	s.createFile(inputFile, xmlBody, c)
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	bucketStat := s.getStat(bucketName, "", c)
	c.Assert(bucketStat["StorageClass"], Equals, "IA")
	s.removeBucket(bucketName, true, c)
	os.Remove(inputFile)
}

func (s *OssutilCommandSuite) TestMbCreateBucketWithServerEncryption(c *C) {
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	c.Assert(err, IsNil)

	bucketName := bucketNamePrefix + randLowStr(10)
	inputFile := "test-ossutil-file-" + randLowStr(5)
	command := "mb"
	xmlBody := `
	<?xml version="1.0" encoding="UTF-8"?>
	<CreateBucketConfiguration>
		<StorageClass>IA</StorageClass>
	</CreateBucketConfiguration>
	`
	s.createFile(inputFile, xmlBody, c)
	args := []string{CloudURLToString(bucketName, ""), inputFile}
	encryption := "X-Oss-Server-Side-Encryption:KMS#X-Oss-Server-Side-Encryption-Key-Id:kms-id#X-Oss-Server-Side-Data-Encryption:SM4"
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"meta":            &encryption,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	rs, err := client.GetBucketEncryption(bucketName)
	c.Assert(err, IsNil)
	c.Assert(rs.SSEDefault.SSEAlgorithm, Equals, "KMS")
	c.Assert(rs.SSEDefault.KMSMasterKeyID, Equals, "kms-id")
	c.Assert(rs.SSEDefault.KMSDataEncryption, Equals, "SM4")
	s.removeBucket(bucketName, true, c)
	os.Remove(inputFile)

	encryption1 := "X-Oss-Server-Side-Encryption:AES256"
	options[OptionMeta] = &encryption1
	args = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	rs, err = client.GetBucketEncryption(bucketName)
	c.Assert(err, IsNil)
	c.Assert(rs.SSEDefault.SSEAlgorithm, Equals, "AES256")
	s.removeBucket(bucketName, true, c)
	os.Remove(inputFile)
}
