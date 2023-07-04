package lib

import (
	"io/ioutil"
	"os"
	"strings"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketCnameTokenSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cname := "oct-10.site"

	// put success
	var str string
	strMethod := "put"
	strItem := "token"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &strItem,
	}

	cnameArgs := []string{CloudURLToString(bucketName, ""), cname}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)

	// get success
	// output to file
	outputFile := "test-file-" + randLowStr(5)
	testResultFile, err = os.OpenFile(outputFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	c.Assert(err, IsNil)

	oldStdout := os.Stdout
	os.Stdout = testResultFile

	strMethod = "get"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout
	outBody := s.readFile(outputFile, c)
	c.Assert(strings.Contains(outBody, cname), Equals, true)

	os.Remove(outputFile)

	cnameArgs = []string{CloudURLToString(bucketName, ""), cname, outputFile}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)

	outBody = s.readFile(outputFile, c)
	c.Assert(strings.Contains(outBody, cname), Equals, true)
	os.Remove(outputFile)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameTokenError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cname := "oct-10.site"

	var str string
	// error strMethod
	strMethod := "putt"
	strItem := "token"
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
		"item":            &strItem,
	}

	cnameArgs := []string{CloudURLToString(bucketName, ""), cname}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// error item
	strItem = "tokenn"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// not supported delete
	strMethod = "delete"
	strItem = "token"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnamePutGetDelete(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cname := "oct-10.site"
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

	cnameArgs := []string{CloudURLToString(bucketName, ""), cname}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "ErrorCode=NeedVerifyDomainOwnership"), Equals, true)

	localFile := "test-ossutil-cname-" + randLowStr(5) + ".xml"
	xmlContent := "<BucketCnameConfiguration>\n  <Cname>\n    <Domain>" + cname + "</Domain>\n    <CertificateConfiguration>\n      <CertId>493****-cn-hangzhou</CertId>\n      <Certificate>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</Certificate>\n      <PrivateKey>-----BEGIN CERTIFICATE----- MIIDhDCCAmwCCQCFs8ixARsyrDANBgkqhkiG9w0BAQsFADCBgzELMAkGA1UEBhMC **** -----END CERTIFICATE-----</PrivateKey>\n      <PreviousCertId>493****-cn-hangzhou</PreviousCertId>\n      <Force>true</Force>\n    </CertificateConfiguration>\n  </Cname>\n</BucketCnameConfiguration>"
	s.createFile(localFile, xmlContent, c)
	cnameArgs = []string{CloudURLToString(bucketName, ""), cname}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "ErrorCode=NeedVerifyDomainOwnership"), Equals, true)

	os.Remove(localFile)
	localFile = "test-ossutil-cname-" + randLowStr(5) + ".xml"
	xmlContent = "<BucketCnameConfiguration>\n  <Cname>\n    <Domain>" + cname + "</Domain>\n    <CertificateConfiguration>\n      <DeleteCertificate>True</DeleteCertificate>\n    </CertificateConfiguration></Cname>\n</BucketCnameConfiguration>"
	s.createFile(localFile, xmlContent, c)
	cnameArgs = []string{CloudURLToString(bucketName, ""), cname}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "ErrorCode=NeedVerifyDomainOwnership"), Equals, true)

	strMethod = "delete"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)

	strMethod = "get"
	cnameArgs = []string{CloudURLToString(bucketName, "")}
	outputFile := "test-file-" + randLowStr(5)
	testResultFile, err = os.OpenFile(outputFile, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	c.Assert(err, IsNil)

	oldStdout := os.Stdout
	os.Stdout = testResultFile
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)
	testResultFile.Close()
	os.Stdout = oldStdout
	outBody := s.readFile(outputFile, c)
	c.Assert(strings.Contains(outBody, bucketName), Equals, true)
	c.Assert(strings.Contains(outBody, cname), Equals, false)

	// error method
	strMethod = "gett"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	os.Remove(outputFile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cname := "oct-10.site"
	var str string

	// method is empty
	strMethod := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	cnameArgs := []string{CloudURLToString(bucketName, ""), cname}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// cloud url is error
	strMethod = "put"
	cnameArgs = []string{"http://test-bucket", cname}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// bucket name is empty
	cnameArgs = []string{"oss://", cname}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	//put, no cname
	strMethod = "put"
	cnameArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	//delete,no cname
	strMethod = "delete"
	cnameArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// get token, no cname
	strToken := "token"
	options["item"] = &strToken

	strMethod = "get"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	// put token,no cname
	strMethod = "put"
	_, err = cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameNotConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)
	// get cname
	cnameDownName := "test-ossutil-cname-" + randLowStr(5)
	strMethod := "get"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	cnameArgs := []string{CloudURLToString(bucketName, ""), cnameDownName}
	_, err := cm.RunCommand("bucket-cname", cnameArgs, options)
	c.Assert(err, IsNil)
	xmlBody, err := ioutil.ReadFile(cnameDownName)
	testLogger.Println(string(xmlBody))
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(xmlBody), "ListCnameResult"), Equals, true)
	c.Assert(strings.Contains(string(xmlBody), bucketName), Equals, true)

	os.Remove(cnameDownName)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketCnameHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"bucket-cname"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
