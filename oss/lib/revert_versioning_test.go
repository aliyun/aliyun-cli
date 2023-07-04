package lib

import (
	"os"
	"strconv"
	"time"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestRevertObjectSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// put object1
	object1 := "ossutil_test_object" + randStr(5)
	s.putObject(bucketName, object1, testFileName, c)

	// put object1
	object2 := object1 + "-" + randStr(5)
	s.putObject(bucketName, object2, testFileName, c)

	// rm objects
	s.removeObjects(bucketName, object1, true, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// revert object success
	command := "revert-versioning"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	srcUrl := CloudURLToString(bucketName, object1)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// stat object1 success
	objectStat := s.getStat(bucketName, object1, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertBatchObjectSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)

	// put object1
	object1 := "ossutil_test_object" + randStr(5)
	s.putObject(bucketName, object1, testFileName, c)

	// put object1
	object2 := object1 + "-" + randStr(5)
	s.putObject(bucketName, object2, testFileName, c)

	// rm objects
	s.removeObjects(bucketName, object1, true, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// revert object success
	command := "revert-versioning"
	str := ""
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
	}
	srcUrl := CloudURLToString(bucketName, object1)
	args := []string{srcUrl}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	// stat object1 success
	objectStat := s.getStat(bucketName, object1, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object2 success
	objectStat = s.getStat(bucketName, object2, c)
	versionId = objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertIncludeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	// put object1
	object2 := prefix + "2.txt"
	s.putObject(bucketName, object2, testFileName, c)

	// put object3
	object3 := prefix + "3.jpg"
	s.putObject(bucketName, object3, testFileName, c)

	// rm objects
	s.removeObjects(bucketName, prefix, true, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// stat object3 failure
	_, err = s.rawGetStat(bucketName, object3)
	c.Assert(err, NotNil)

	// revert object success
	srcUrl := CloudURLToString(bucketName, prefix)
	cmdline := []string{"ossutil", "revert-versioning", srcUrl, "--include", "*.txt"}
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
	}
	os.Args = cmdline
	_, err = cm.RunCommand(command, revertArgs, options)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// stat object1 success  1.txt
	objectStat := s.getStat(bucketName, object1, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object2 success  2.txt
	objectStat = s.getStat(bucketName, object2, c)
	versionId = objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object3 failure 3.jpg
	_, err = s.rawGetStat(bucketName, object3)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertExcludeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	// put object1
	object2 := prefix + "2.txt"
	s.putObject(bucketName, object2, testFileName, c)

	// put object3
	object3 := prefix + "3.jpg"
	s.putObject(bucketName, object3, testFileName, c)

	// rm objects
	s.removeObjects(bucketName, prefix, true, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// stat object3 failure
	_, err = s.rawGetStat(bucketName, object3)
	c.Assert(err, NotNil)

	// revert object success
	srcUrl := CloudURLToString(bucketName, prefix)
	cmdline := []string{"ossutil", "revert-versioning", srcUrl, "--exclude", "*.txt"}
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
	}
	os.Args = cmdline
	_, err = cm.RunCommand(command, revertArgs, options)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// stat object1 failuer  1.txt
	_, err = s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure  2.txt
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// stat object3 failure 3.jpg
	objectStat := s.getStat(bucketName, object3, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertStartTimeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	// put object2
	object2 := prefix + "2.txt"
	s.putObject(bucketName, object2, testFileName, c)

	// rm object1
	s.removeObjects(bucketName, object1, false, true, c)

	time.Sleep(1 * time.Second)
	startTime := time.Now().Unix()
	time.Sleep(1 * time.Second)

	// rm object2
	s.removeObjects(bucketName, object2, false, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// revert object
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	strStartTime := strconv.FormatInt(startTime, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
		"startTime":       &strStartTime,
	}
	_, err = cm.RunCommand(command, revertArgs, options)
	c.Assert(err, IsNil)

	// stat object1 failuer 1.txt
	_, err = s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 success 2.txt
	objectStat := s.getStat(bucketName, object2, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertEndTimeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)
	time.Sleep(3 * time.Second)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	// put object2
	object2 := prefix + "2.txt"
	s.putObject(bucketName, object2, testFileName, c)

	// rm object1
	s.removeObjects(bucketName, object1, false, true, c)

	time.Sleep(1 * time.Second)
	endTime := time.Now().Unix()
	time.Sleep(1 * time.Second)

	// rm object2
	s.removeObjects(bucketName, object2, false, true, c)

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// revert object
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	strEndTime := strconv.FormatInt(endTime, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
		"endTime":         &strEndTime,
	}
	_, err = cm.RunCommand(command, revertArgs, options)
	c.Assert(err, IsNil)

	// stat object1 success
	objectStat := s.getStat(bucketName, object1, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertStartAndEndTimeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	// put object2
	object2 := prefix + "2.txt"
	s.putObject(bucketName, object2, testFileName, c)

	startTime := time.Now().Unix()
	time.Sleep(1 * time.Second)

	// rm object1
	s.removeObjects(bucketName, object1, false, true, c)
	// rm object2
	s.removeObjects(bucketName, object2, false, true, c)
	time.Sleep(1 * time.Second)
	endTime := time.Now().Unix()

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// stat object2 failure
	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	// revert object
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	strStartTime := strconv.FormatInt(startTime, 10)
	strEndTime := strconv.FormatInt(endTime, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
		"startTime":       &strStartTime,
		"endTime":         &strEndTime,
	}
	_, err = cm.RunCommand(command, revertArgs, options)
	c.Assert(err, IsNil)

	// stat object1 success
	objectStat := s.getStat(bucketName, object1, c)
	versionId := objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	// stat object2 success
	objectStat = s.getStat(bucketName, object2, c)
	versionId = objectStat["X-Oss-Version-Id"]
	c.Assert(len(versionId) > 0, Equals, true)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertStartAndEndTimeFilterError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object1 := prefix + "1.txt"
	s.putObject(bucketName, object1, testFileName, c)

	startTime := time.Now().Unix()
	time.Sleep(1 * time.Second)

	// rm object1
	s.removeObjects(bucketName, object1, false, true, c)

	time.Sleep(1 * time.Second)
	endTime := time.Now().Unix()

	// stat object1 failure
	_, err := s.rawGetStat(bucketName, object1)
	c.Assert(err, NotNil)

	// revert object, startTime less than endTime
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command := "revert-versioning"
	str := ""
	recursive := true
	strStartTime := strconv.FormatInt(startTime, 10)
	strEndTime := strconv.FormatInt(endTime, 10)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
		"startTime":       &strEndTime,
		"endTime":         &strStartTime,
	}
	_, err = cm.RunCommand(command, revertArgs, options)
	c.Assert(err, NotNil)
	os.Remove(testFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRevertPayer(c *C) {
	bucketName := payerBucket + randLowStr(10)
	s.putBucket(bucketName, c)
	s.putBucketVersioning(bucketName, "enabled", c)
	policy := `
	{
		"Version":"1",
		"Statement":[
			{
				"Action":[
					"oss:*"
				],
				"Effect":"Allow",
				"Principal":["` + payerAccountID + `"],
				"Resource":["acs:oss:*:*:` + bucketName + `", "acs:oss:*:*:` + bucketName + `/*"]
			}
		]
	}`
	s.putBucketPolicy(bucketName, policy, c)

	// create file
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(testFileName, data, c)
	prefix := "ossutil_test_object" + randStr(5)

	// put object1
	object := prefix + "1.txt"
	args := []string{testFileName, CloudURLToString(bucketName, object)}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// rm with payer
	rmArgs := []string{CloudURLToString(bucketName, prefix)}
	bForce := true
	str := ""
	recursive := true
	requester := "requester"
	rmOptions := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"force":           &bForce,
		"recursive":       &recursive,
		"payer":           &requester,
	}
	_, err = cm.RunCommand("rm", rmArgs, rmOptions)
	os.Args = []string{}
	c.Assert(err, IsNil)

	// stat with payer failure
	command := "stat"
	args = []string{CloudURLToString(bucketName, object)}
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// revert object
	revertArgs := []string{CloudURLToString(bucketName, prefix)}
	command = "revert-versioning"
	recursive = true
	options = OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"recursive":       &recursive,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, revertArgs, options)
	c.Assert(err, IsNil)

	// stat with payer success
	command = "stat"
	args = []string{CloudURLToString(bucketName, object)}
	options = OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	os.Remove(testFileName)
}

func (s *OssutilCommandSuite) TestRevertPayerError(c *C) {
	bucketName := payerBucket

	requester := "Invalie-requester"

	// revert object
	revertArgs := []string{CloudURLToString(bucketName, "")}
	command := "revert-versioning"
	recursive := true
	str := ""
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"recursive":       &recursive,
		"payer":           &requester,
	}
	_, err := cm.RunCommand(command, revertArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestRevertNotInputKeyError(c *C) {
	bucketName := "ossutil_test_object" + randStr(5)
	// revert object, startTime less than endTime
	revertArgs := []string{CloudURLToString(bucketName, "")}
	command := "revert-versioning"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	_, err := cm.RunCommand(command, revertArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestRevertHelpInfo(c *C) {
	options := OptionMapType{}
	mkArgs := []string{"revert-versioning"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestRevertCloudUrlError(c *C) {
	bucketName := "ossutil_test_object" + randStr(5)
	// revert object, startTime less than endTime
	revertArgs := []string{"osss://" + bucketName}
	command := "revert-versioning"
	str := ""
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
	}
	_, err := cm.RunCommand(command, revertArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestRevertFilterError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	srcUrl := CloudURLToString(bucketName, "")
	cmdline := []string{"ossutil", "revert-versioning", srcUrl, "--include", "a/b/c.txt"}
	revertArgs := []string{CloudURLToString(bucketName, "")}
	command := "revert-versioning"
	str := ""
	recursive := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &ConfigFile,
		"recursive":       &recursive,
	}
	os.Args = cmdline
	_, err := cm.RunCommand(command, revertArgs, options)
	os.Args = []string{}
	c.Assert(err, NotNil)
}
