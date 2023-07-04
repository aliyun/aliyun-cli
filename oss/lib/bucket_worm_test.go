package lib

import (
	"time"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestWormInitiateBucketWormSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	wormArgs := []string{"init", CloudURLToString(bucketName, ""), "10"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(3 * time.Second)

	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(wormCommand.wmOption.wormConfig.WormId) > 0, Equals, true)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "InProgress")
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormInitiateBucketWormError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// days not exist
	wormArgs := []string{"init", CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	// days is error
	wormArgs = []string{"init", CloudURLToString(bucketName, ""), "abc"}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	// bucket is not exist
	wormArgs = []string{"init", CloudURLToString(bucketName+"-test", ""), "abc"}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormAbortBucketWorm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// init worm
	wormArgs := []string{"init", CloudURLToString(bucketName, ""), "10"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	// get worm
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(wormCommand.wmOption.wormConfig.WormId) > 0, Equals, true)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "InProgress")

	//abort worm
	wormArgs = []string{"abort", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	var wormConfig oss.WormConfiguration
	wormCommand.wmOption.wormConfig = wormConfig

	//get again
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormCompleteBucketWormSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// init worm
	wormArgs := []string{"init", CloudURLToString(bucketName, ""), "10"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	// get worm
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(wormCommand.wmOption.wormConfig.WormId) > 0, Equals, true)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "InProgress")

	//complete worm
	wormArgs = []string{"complete", CloudURLToString(bucketName, ""), wormCommand.wmOption.wormConfig.WormId}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	time.Sleep(time.Second * 3)

	var wormConfig oss.WormConfiguration
	wormCommand.wmOption.wormConfig = wormConfig

	//get again
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "Locked")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormCompleteBucketWormError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	//complete worm
	wormArgs := []string{"complete", CloudURLToString(bucketName, "")}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormExtentBucketWormSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// init worm
	wormArgs := []string{"init", CloudURLToString(bucketName, ""), "10"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	// get worm
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(len(wormCommand.wmOption.wormConfig.WormId) > 0, Equals, true)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "InProgress")

	//complete worm
	wormArgs = []string{"complete", CloudURLToString(bucketName, ""), wormCommand.wmOption.wormConfig.WormId}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	var wormConfig oss.WormConfiguration
	wormCommand.wmOption.wormConfig = wormConfig

	//get again
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "Locked")
	c.Assert(wormCommand.wmOption.wormConfig.RetentionPeriodInDays, Equals, 10)

	// extent to 20
	wormArgs = []string{"extend", CloudURLToString(bucketName, ""), "20", wormCommand.wmOption.wormConfig.WormId}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)

	// get again
	wormArgs = []string{"get", CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, IsNil)
	c.Assert(wormCommand.wmOption.wormConfig.State, Equals, "Locked")
	c.Assert(wormCommand.wmOption.wormConfig.RetentionPeriodInDays, Equals, 20)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormExtentBucketWormError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	wormArgs := []string{"extend", CloudURLToString(bucketName, ""), "20"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	// days is invalid
	wormArgs = []string{"extend", CloudURLToString(bucketName, ""), "abc", "abc"}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormCommandInvalid(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	// command is init-test
	wormArgs := []string{"init-test", CloudURLToString(bucketName, ""), "10"}
	_, err := cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	// cloud url is invalid
	wormArgs = []string{"init", "http://bucket", "10"}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	// cloud url is invalid
	wormArgs = []string{"init", "oss://", "10"}
	_, err = cm.RunCommand("worm", wormArgs, options)
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestWormHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"worm"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}
