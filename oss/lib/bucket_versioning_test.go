package lib

import (
	"os"
	"time"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestBucketVersioningPutSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// versioning command test
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

	versioningArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketVersioningCommand.versioningResult.Status, Equals, "null")

	// set bucket versioning enabled
	strMethod = "put"
	versioningArgs = []string{CloudURLToString(bucketName, ""), "enabled"}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// check
	strMethod = "get"
	versioningArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketVersioningCommand.versioningResult.Status, Equals, "Enabled")
	time.Sleep(time.Second * 3)

	// set bucket versioning suspend
	strMethod = "put"
	versioningArgs = []string{CloudURLToString(bucketName, ""), "suspended"}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	// check
	strMethod = "get"
	versioningArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, IsNil)
	c.Assert(bucketVersioningCommand.versioningResult.Status, Equals, "Suspended")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketVersioningError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// bucket-versioning command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// method is empty
	versioningArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, NotNil)

	// method is error
	strMethod := "puttt"
	options["method"] = &strMethod
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, NotNil)

	// args is empty
	strMethod = "put"
	versioningArgs = []string{CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, NotNil)

	//value is error
	versioningArgs = []string{CloudURLToString(bucketName, ""), "disabled"}
	_, err = cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketVersioningPutEmptyEndpoint(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	cfile := randStr(10)
	data := "[Credentials]" + "\n" + "language=CH" + "\n" + "accessKeyID=123" + "\n" + "accessKeySecret=456" + "\n" + "endpoint="
	s.createFile(cfile, data, c)

	// bucket-versioing command test
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

	versioningArgs := []string{CloudURLToString(bucketName, ""), "enabled"}
	_, err := cm.RunCommand("bucket-versioning", versioningArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketVersioningGetEmptyEndpoint(c *C) {
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
		"configFile":      &configFile,
		"method":          &strMethod,
	}

	versioingArgs := []string{CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("bucket-versioing", versioingArgs, options)
	c.Assert(err, NotNil)

	os.Remove(cfile)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBucketVersioningHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"bucket-versioning"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
