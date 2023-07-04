package lib

import (
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestSyncUploadSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	dirName := "testdir-" + randLowStr(3)
	subDirName := "subdir-" + randLowStr(4)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName
	object1 := fileName
	testFileName2 := dirName + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + fileName
	object2 := subDirName + "/" + fileName

	err := os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// begin sync file
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
	}

	// upload
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(bucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	objectStat = s.getStat(bucketName, object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadWithOssPrefix1Success(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	dirName := "testdir-" + randLowStr(3)
	subDirName := "subdir-" + randLowStr(4)
	fileName := "testfile-" + randLowStr(5)

	ossPrefix := "prefix"
	testFileName1 := dirName + string(os.PathSeparator) + fileName
	object1 := ossPrefix + "/" + fileName
	testFileName2 := dirName + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + fileName
	object2 := ossPrefix + "/" + subDirName + "/" + fileName

	err := os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// begin sync file
	syncArgs := []string{dirName, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
	}

	// upload
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(bucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	objectStat = s.getStat(bucketName, object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadWithOssPrefix2Success(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	dirName := "testdir-" + randLowStr(3)
	subDirName := "subdir-" + randLowStr(4)
	fileName := "testfile-" + randLowStr(5)

	ossPrefix := "prefix/"
	testFileName1 := dirName + string(os.PathSeparator) + fileName
	object1 := ossPrefix + fileName
	testFileName2 := dirName + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + fileName
	object2 := ossPrefix + subDirName + "/" + fileName

	err := os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// begin sync file
	syncArgs := []string{dirName, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
	}

	// upload
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(bucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	objectStat = s.getStat(bucketName, object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadWithDeleteOption1Success(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	// dir2
	dirName2 := "testdir2-" + randLowStr(3)
	subDirName2 := "subdir2-" + randLowStr(4)
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName2, subDirName2, filePrefix2, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	syncArgs := []string{dirName1, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// upload dir2 without prefix
	syncArgs = []string{dirName2, CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// dir1 objects are deleted
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(bucketName, object)
		c.Assert(err, NotNil)
	}

	os.RemoveAll(dirName1)
	os.RemoveAll(dirName2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadWithDeleteOption2Success(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	// dir2
	dirName2 := "testdir2-" + randLowStr(3)
	subDirName2 := "subdir2-" + randLowStr(4)
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName2, subDirName2, filePrefix2, "", 3, c)

	// dir3
	dirName3 := "testdir3-" + randLowStr(3)
	subDirName3 := "subdir3-" + randLowStr(4)
	filePrefix3 := "prefix3"
	fileNameList3 := s.prepareTestFiles(dirName3, subDirName3, filePrefix3, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	syncArgs := []string{dirName1, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// upload dir2 without prefix
	syncArgs = []string{dirName2, CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// dir1 objects are not deleted
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// upload dir3 with prefix and delete
	options["delete"] = &bDelete
	syncArgs = []string{dirName3, CloudURLToString(bucketName, ossPrefix)}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList3 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// dir1 objects are deleted
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(bucketName, object)
		c.Assert(err, NotNil)
	}

	// dir2 objects are not deleted
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	time.Sleep(time.Second * 240)

	os.RemoveAll(dirName1)
	os.RemoveAll(dirName2)
	os.RemoveAll(dirName3)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopySuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	destBucketName := bucketName + "-dest"
	s.putBucket(destBucketName, c)

	text := randLowStr(100)
	dirName := "testdir-" + randLowStr(3)
	subDirName := "subdir-" + randLowStr(4)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName
	object1 := fileName
	testFileName2 := dirName + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + fileName
	object2 := subDirName + "/" + fileName

	err := os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	//put objects
	_, err = s.rawCP(dirName, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// sync to dest bucket
	syncArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(destBucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(destBucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	objectStat = s.getStat(destBucketName, object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopyWithDestPrefixSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	destBucketName := bucketName + "-dest"
	destPrefix := "prefix"
	s.putBucket(destBucketName, c)

	text := randLowStr(100)
	dirName := "testdir-" + randLowStr(3)
	subDirName := "subdir-" + randLowStr(4)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName
	object1 := fileName
	testFileName2 := dirName + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + fileName
	object2 := subDirName + "/" + fileName

	err := os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	//put objects
	_, err = s.rawCP(dirName, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// sync to dest bucket
	syncArgs := []string{CloudURLToString(bucketName, ""), CloudURLToString(destBucketName, destPrefix)}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(destBucketName, destPrefix+"/"+object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	objectStat = s.getStat(destBucketName, destPrefix+"/"+object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopyWithSrcPrefixNotExistSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	destBucketName := bucketName + "-dest"
	s.putBucket(destBucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	//put objects
	_, err := s.rawCP(dirName1, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// sync to dest bucket
	syncArgs := []string{CloudURLToString(bucketName, filePrefix1), CloudURLToString(destBucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// dest object not exist
	for _, v := range fileNameList1 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(destBucketName, object)
		c.Assert(err, NotNil)
	}
	os.RemoveAll(dirName1)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopyWithSrcPrefixExistSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	destBucketName := bucketName + "-dest"
	s.putBucket(destBucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	//put objects
	_, err := s.rawCP(dirName1, CloudURLToString(bucketName, filePrefix1), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// sync to dest bucket
	syncArgs := []string{CloudURLToString(bucketName, filePrefix1), CloudURLToString(destBucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// dest object not exist
	for _, v := range fileNameList1 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(destBucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}
	os.RemoveAll(dirName1)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopyWithDeleteOptionSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	destBucketName := bucketName + "-dest"
	s.putBucket(destBucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	// dir2
	dirName2 := "testdir2-" + randLowStr(3)
	subDirName2 := "subdir2-" + randLowStr(4)
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName2, subDirName2, filePrefix2, "", 3, c)

	// dir3
	dirName3 := "testdir3-" + randLowStr(3)
	subDirName3 := "subdir3-" + randLowStr(4)
	filePrefix3 := "prefix3"
	fileNameList3 := s.prepareTestFiles(dirName3, subDirName3, filePrefix3, "", 3, c)

	// upload dir1 with prefix on destBucket
	ossPrefix := "prefix"
	_, err := s.rawCP(dirName1, CloudURLToString(destBucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// upload dir2 without prefix on destBucket
	_, err = s.rawCP(dirName2, CloudURLToString(destBucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// upload dir3 with prefix on Bucket
	_, err = s.rawCP(dirName3, CloudURLToString(bucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// sync to dest bucket
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), CloudURLToString(destBucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check dest bucketname, fileNameList1,not exist, are deleted
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(destBucketName, object)
		c.Assert(err, NotNil)
	}

	// check dest bucketname, exist, old objects
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(destBucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// check dest bucketname, exist, new sync object from bucket
	for _, v := range fileNameList3 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(destBucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	os.RemoveAll(dirName1)
	os.RemoveAll(dirName2)
	os.RemoveAll(dirName3)
	s.removeBucket(bucketName, true, c)
	s.removeBucket(destBucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDownloadSubDirExist(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	_, err := s.rawCP(dirName, CloudURLToString(bucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// make dir2
	dirName2 := "testdir2-" + randLowStr(2)
	err = os.MkdirAll(dirName2, 0755)
	c.Assert(err, IsNil)

	// sync to dir2
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName2}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check file exist
	for _, v := range fileNameList1 {
		fileName := dirName2 + string(os.PathSeparator) + v
		f, err := os.Stat(fileName)
		c.Assert(err, IsNil)
		c.Assert(!f.IsDir(), Equals, true)
	}

	// create prefix2 file in dir1
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName, subDirName, filePrefix2, "", 3, c)

	// sync to dir1 with delete operation
	backupDir := "test-backup-dir"
	bDelete := true
	options["delete"] = &bDelete
	options["backupDir"] = &backupDir

	syncArgs = []string{CloudURLToString(bucketName, ossPrefix), dirName}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check file exist
	for _, v := range fileNameList1 {
		fileName := dirName + string(os.PathSeparator) + v
		f, err := os.Stat(fileName)
		c.Assert(err, IsNil)
		c.Assert(!f.IsDir(), Equals, true)
	}

	// check file not exist
	for _, v := range fileNameList2 {
		fileName := dirName + string(os.PathSeparator) + v
		_, err := os.Stat(fileName)
		c.Assert(err, NotNil)
	}

	// check bakup dir,file exist
	for _, v := range fileNameList2 {
		fileName := backupDir + string(os.PathSeparator) + v
		_, err := os.Stat(fileName)
		c.Assert(err, IsNil)
	}

	// subdir is exist
	f, err := os.Stat(dirName + string(os.PathSeparator) + subDirName)
	c.Assert(err, IsNil)
	c.Assert(f.IsDir(), Equals, true)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDownloadSubDirRemoved(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	_, err := s.rawCP(dirName, CloudURLToString(bucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// create prefix2 file in subdir2
	subDirName2 := subDirName + "-dest"
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName, subDirName2, filePrefix2, "", 3, c)

	// sync to dir
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName}
	str := ""
	cpDir := CheckpointDir
	backupDir := "test-backup-dir"
	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
		"backupDir":       &backupDir,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check file exist
	for _, v := range fileNameList1 {
		fileName := dirName + string(os.PathSeparator) + v
		f, err := os.Stat(fileName)
		c.Assert(err, IsNil)
		c.Assert(!f.IsDir(), Equals, true)
	}

	// check file not exist
	for _, v := range fileNameList2 {
		fileName := dirName + string(os.PathSeparator) + v
		_, err := os.Stat(fileName)
		c.Assert(err, NotNil)
	}

	// check bakup dir,file exist
	for _, v := range fileNameList2 {
		fileName := backupDir + string(os.PathSeparator) + v
		_, err := os.Stat(fileName)
		c.Assert(err, IsNil)
	}

	// subdir is not exist
	_, err = os.Stat(dirName + string(os.PathSeparator) + subDirName2)
	c.Assert(err, NotNil)
	os.RemoveAll(dirName)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadIncludeFilterSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	// dir
	dirName := "testdir1-" + randLowStr(3)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName + ".txt"
	object1 := fileName + ".txt"
	testFileName2 := dirName + string(os.PathSeparator) + fileName + ".jpg"
	object2 := fileName + ".jpg"

	err := os.MkdirAll(dirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// sync dir to oss
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	cmdline := []string{"ossutil", "sync", dirName, CloudURLToString(bucketName, ""), "-f", "--include", "*.txt"}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	os.Args = cmdline
	_, err = cm.RunCommand("sync", syncArgs, options)
	os.Args = []string{}
	c.Assert(err, IsNil)

	//check, get stat
	objectStat := s.getStat(bucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	_, err = s.rawGetStat(bucketName, object2)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDownloadIncludeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	// dir
	dirName := "testdir1-" + randLowStr(3)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName + ".txt"
	object1 := fileName + ".txt"
	testFileName2 := dirName + string(os.PathSeparator) + fileName + ".jpg"
	object2 := fileName + ".jpg"

	err := os.MkdirAll(dirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// raw cp to oss
	_, err = s.rawCP(dirName, CloudURLToString(bucketName, ""), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	testFileName3 := dirName + string(os.PathSeparator) + "dest-" + fileName + ".txt"
	s.createFile(testFileName3, text, c)

	syncArgs := []string{CloudURLToString(bucketName, ""), dirName}
	cmdline := []string{"ossutil", "sync", CloudURLToString(bucketName, ""), dirName, "-f", "--include", "*.txt"}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	bDelete := true
	backupDir := "test-backup-dir"
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
		"backupDir":       &backupDir,
	}

	os.Args = cmdline
	_, err = cm.RunCommand("sync", syncArgs, options)
	os.Args = []string{}
	c.Assert(err, IsNil)

	//check, txt object exist
	objectStat := s.getStat(bucketName, object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	//check, jpg object exist
	objectStat = s.getStat(bucketName, object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	//check, jpg file exist
	_, err = os.Stat(testFileName1)
	c.Assert(err, IsNil)

	//check, jpg file exist
	_, err = os.Stat(testFileName2)
	c.Assert(err, IsNil)

	//check, file not exist
	_, err = os.Stat(testFileName3)
	c.Assert(err, NotNil)

	//check, backup file exist
	backupFile := backupDir + string(os.PathSeparator) + "dest-" + fileName + ".txt"
	_, err = os.Stat(backupFile)
	c.Assert(err, IsNil)

	os.RemoveAll(dirName)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCopyIncludeFilter(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	// dir
	dirName := "testdir1-" + randLowStr(3)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName + ".txt"
	object1 := fileName + ".txt"
	testFileName2 := dirName + string(os.PathSeparator) + fileName + ".jpg"
	object2 := fileName + ".jpg"

	err := os.MkdirAll(dirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	prefix1 := "prefix1"
	prefix2 := "prefix2"

	// raw cp to oss
	_, err = s.rawCP(dirName, CloudURLToString(bucketName, prefix1), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	syncArgs := []string{CloudURLToString(bucketName, prefix1), CloudURLToString(bucketName, prefix2)}
	cmdline := []string{"ossutil", "sync", CloudURLToString(bucketName, prefix1), CloudURLToString(bucketName, prefix2), "-f", "--include", "*.txt"}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	os.Args = cmdline
	_, err = cm.RunCommand("sync", syncArgs, options)
	os.Args = []string{}
	c.Assert(err, IsNil)

	//check, txt object exist
	objectStat := s.getStat(bucketName, prefix1+"/"+object1, c)
	etag := objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	//check, jpg object exist
	objectStat = s.getStat(bucketName, prefix1+"/"+object2, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	//check, txt object exist
	objectStat = s.getStat(bucketName, prefix2+"/"+object1, c)
	etag = objectStat["Etag"]
	c.Assert(len(etag) > 0, Equals, true)

	//check, jpg object exist
	_, err = s.rawGetStat(bucketName, prefix2+"/"+object2)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncWithPayerSucess(c *C) {
	bucketName := payerBucket

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	ossPrefix := "ossutil-prefix"

	// upload dir1 with prefix
	args := []string{dirName, CloudURLToString(bucketName, ossPrefix)}
	_, err := s.rawCPWithPayer(args, true, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)

	// make dir2
	dirName2 := "testdir2-" + randLowStr(2)
	err = os.MkdirAll(dirName2, 0755)
	c.Assert(err, IsNil)

	// sync to dir2
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName2}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	requester := "requester"
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &payerConfigFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"payer":           &requester,
	}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check file exist
	for _, v := range fileNameList1 {
		fileName := dirName2 + string(os.PathSeparator) + v
		f, err := os.Stat(fileName)
		c.Assert(err, IsNil)
		c.Assert(!f.IsDir(), Equals, true)
	}
	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
}

func (s *OssutilCommandSuite) TestSyncPayerError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	// dir
	dirName := "testdir1-" + randLowStr(3)
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bForce := true

	// cp command doesn't support --all-versions
	requester := "requester" + "---"
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"payer":           &requester,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncInitCpCommandError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	// dir
	dirName := "testdir1-" + randLowStr(3)
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bForce := true

	// cp command doesn't support --all-versions
	bAllVersions := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"allVersions":     &bAllVersions,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncCloudUrlError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	// dir
	dirName := "testdir1-" + randLowStr(3)
	syncArgs := []string{dirName, "s3://" + bucketName}
	str := ""
	cpDir := CheckpointDir
	bForce := true

	// cp command doesn't support --all-versions
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncBetweenLocalDirError(c *C) {
	// dir1
	dirName := "testdir1-" + randLowStr(3)
	str := ""
	cpDir := CheckpointDir

	//dir2
	dirName1 := "dest-" + dirName
	syncArgs := []string{dirName, dirName1}
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncUploadWithDeleteWithoutConfirm(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	// dir2
	dirName2 := "testdir2-" + randLowStr(3)
	subDirName2 := "subdir2-" + randLowStr(4)
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName2, subDirName2, filePrefix2, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	syncArgs := []string{dirName1, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// upload dir2 without prefix
	syncArgs = []string{dirName2, CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	// dir1 objects are not deleted,because not confirm
	for _, v := range fileNameList1 {
		object := ossPrefix + "/" + strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(bucketName, object)
		c.Assert(err, IsNil)
	}

	os.RemoveAll(dirName1)
	os.RemoveAll(dirName2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadExceedMaxCount(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	maxCount := MaxSyncNumbers
	MaxSyncNumbers = 2

	// upload dir1 with prefix
	ossPrefix := "prefix"
	syncArgs := []string{dirName1, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
	MaxSyncNumbers = maxCount

	os.RemoveAll(dirName1)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDownloadExceedMaxCount(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 3, c)

	maxCount := MaxSyncNumbers

	// upload dir1 with prefix
	ossPrefix := "prefix"
	syncArgs := []string{dirName1, CloudURLToString(bucketName, ossPrefix)}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	os.RemoveAll(dirName1)
	MaxSyncNumbers = 2
	syncArgs = []string{CloudURLToString(bucketName, ossPrefix), dirName1}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	MaxSyncNumbers = maxCount
	os.RemoveAll(dirName1)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDeleteObjectsExceedMaxBatchCount(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName1 := "testdir1-" + randLowStr(3)
	subDirName1 := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName1, subDirName1, filePrefix1, "", 150, c)

	// upload dir1 with prefix
	syncArgs := []string{dirName1, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// dir2
	dirName2 := "testdir2-" + randLowStr(3)
	subDirName2 := "subdir2-" + randLowStr(4)
	filePrefix2 := "prefix2"
	fileNameList2 := s.prepareTestFiles(dirName2, subDirName2, filePrefix2, "", 10, c)

	syncArgs = []string{dirName2, CloudURLToString(bucketName, "")}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// check objects are deleted
	for _, v := range fileNameList1 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		_, err = s.rawGetStat(bucketName, object)
		c.Assert(err, NotNil)
	}

	// check objects are not deleted
	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}
	os.RemoveAll(dirName1)
	os.RemoveAll(dirName2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncFilterError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	text := randLowStr(100)
	// dir
	dirName := "testdir1-" + randLowStr(3)
	fileName := "testfile-" + randLowStr(5)

	testFileName1 := dirName + string(os.PathSeparator) + fileName + ".txt"
	testFileName2 := dirName + string(os.PathSeparator) + fileName + ".jpg"

	err := os.MkdirAll(dirName, 0755)
	s.createFile(testFileName1, text, c)
	s.createFile(testFileName2, text, c)

	// sync dir to oss
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}

	//Error: --include or --exclude does not support format containing dir info
	strFilter := "a" + string(os.PathSeparator) + "b" + string(os.PathSeparator) + "*.txt"

	cmdline := []string{"ossutil", "sync", dirName, CloudURLToString(bucketName, ""), "-f", "--include", strFilter}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
	}
	os.Args = cmdline
	_, err = cm.RunCommand("sync", syncArgs, options)
	os.Args = []string{}
	c.Assert(err, NotNil)
	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncInvalidCloudUrl(c *C) {
	syncArgs := []string{"http://123", "http://123"}
	str := ""
	cpDir := CheckpointDir
	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
	}
	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncSrcIsFileError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	s.createFile(dirName, "123", c)
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		//"delete":          &bDelete,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	options["delete"] = &bDelete
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCreateDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileList := s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	_, err := s.rawCP(dirName, CloudURLToString(bucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// create prefix2 file in subdir2
	dirName2 := dirName + "-dest"

	// sync to dir
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName2}
	str := ""
	cpDir := CheckpointDir
	backupDir := "test-backup-dir" + randLowStr(3)
	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
		//"backupDir":       &backupDir,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// file is dowloaded
	for _, v := range fileList {
		_, err = os.Stat(dirName2 + string(os.PathSeparator) + v)
		c.Assert(err, IsNil)
	}

	// backup dir is not created
	_, err = os.Stat(backupDir)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDestIsFileError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	s.createFile(dirName, "123", c)

	// upload dir1 with prefix
	ossPrefix := "prefix"

	// sync to dir
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName}
	str := ""
	cpDir := CheckpointDir
	backupDir := "test-backup-dir" + randLowStr(3)
	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
		"backupDir":       &backupDir,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncBackupDirIsSubDirError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	// upload dir1 with prefix
	ossPrefix := "prefix"
	_, err := s.rawCP(dirName, CloudURLToString(bucketName, ossPrefix), true, true, false, DefaultBigFileThreshold, CheckpointDir)
	c.Assert(err, IsNil)

	// create prefix2 file in subdir2
	dirName2 := dirName + "-dest"

	// sync to dir
	syncArgs := []string{CloudURLToString(bucketName, ossPrefix), dirName2}
	str := ""
	cpDir := CheckpointDir

	// backup dir is file
	backupDir := dirName2 + string(os.PathSeparator) + "test-backup-dir" + randLowStr(3)
	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
		"backupDir":       &backupDir,
	}

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncMovePathError(c *C) {
	fileName := "testdir1-" + randLowStr(3)
	s.createFile(fileName, "123", c)
	newFileName := "d:" + string(os.PathSeparator) + "root" + string(os.PathSeparator) + "new.txt"
	err := syncCommand.movePath(fileName, newFileName)
	c.Assert(err, NotNil)
	os.RemoveAll(fileName)
}

func (s *OssutilCommandSuite) TestSyncMoveFileToPath(c *C) {
	fileName := "./testdir1-" + randLowStr(3)
	s.createFile(fileName, "123", c)
	newFileName := "." + string(os.PathSeparator) + "new.txt"
	err := syncCommand.moveFileToPath(fileName, newFileName)
	c.Assert(err, IsNil)

	newFileName2 := "." + string(os.PathSeparator) + "root" + string(os.PathSeparator) + "new.txt"
	err = syncCommand.movePath(fileName, newFileName2)
	c.Assert(err, NotNil)
	os.RemoveAll(fileName)
	os.RemoveAll(newFileName)
	os.RemoveAll(newFileName2)
}

func (s *OssutilCommandSuite) TestSyncReadDirLimitError(c *C) {
	dirName := "d:" + string(os.PathSeparator) + "root" + string(os.PathSeparator) + "new.txt"
	_, err := syncCommand.readDirLimit(dirName, 10)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestSyncSrcDirNotExistError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)

	// sync to dir
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir

	bForce := true
	bDelete := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"force":           &bForce,
		"delete":          &bDelete,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncHelpInfo(c *C) {
	// mkdir command test
	options := OptionMapType{}

	mkArgs := []string{"sync"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestSyncSubDirIsNotRemoved(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	filePrefix2 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, ".txt", 3, c)
	fileNameList2 := s.prepareTestFiles(dirName, subDirName, filePrefix2, ".jpg", 3, c)

	// upload dir
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList1 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	for _, v := range fileNameList2 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	filePrefix3 := "prefix3"
	fileNameList3 := s.prepareTestFiles(dirName, subDirName, filePrefix3, ".txt", 3, c)

	// download with filter
	syncArgs = []string{CloudURLToString(bucketName, ""), dirName}
	cmdline := []string{"ossutil", "sync", CloudURLToString(bucketName, ""), dirName, "-f", "--include", "*.txt"}

	backupDir := "test-backup-dir" + randLowStr(3)
	options["backupDir"] = &backupDir

	os.Args = cmdline
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	os.Args = []string{}

	for _, v := range fileNameList1 {
		_, err := os.Stat(dirName + string(os.PathSeparator) + v)
		c.Assert(err, IsNil)
	}

	for _, v := range fileNameList2 {
		_, err := os.Stat(dirName + string(os.PathSeparator) + v)
		c.Assert(err, IsNil)
	}

	// not exist
	for _, v := range fileNameList3 {
		_, err := os.Stat(dirName + string(os.PathSeparator) + v)
		c.Assert(err, NotNil)
	}

	for _, v := range fileNameList3 {
		_, err := os.Stat(backupDir + string(os.PathSeparator) + v)
		c.Assert(err, IsNil)
	}

	os.RemoveAll(dirName)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncBackupDirIsEmpty(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, ".txt", 3, c)

	// upload dir
	syncArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)
	for _, v := range fileNameList1 {
		object := strings.Replace(v, string(os.PathSeparator), "/", -1)
		objectStat := s.getStat(bucketName, object, c)
		etag := objectStat["Etag"]
		c.Assert(len(etag) > 0, Equals, true)
	}

	dirName2 := dirName + "-dest"
	os.MkdirAll(dirName2, 0755)

	// download with filter
	syncArgs = []string{CloudURLToString(bucketName, ""), dirName2}
	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncSubDirIsDeleted(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	filePrefix1 := "prefix1"
	fileNameList1 := s.prepareTestFiles(dirName, subDirName, filePrefix1, "", 3, c)

	dirName2 := "dest-" + dirName
	fileName2 := "test-ossutil-file-" + randStr(5)
	os.MkdirAll(dirName2, 0755)
	s.createFile(dirName2+string(os.PathSeparator)+fileName2, "123", c)

	// upload dir
	syncArgs := []string{dirName2, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// download with filter
	syncArgs = []string{CloudURLToString(bucketName, ""), dirName}
	backupDir := "test-backup-dir" + randLowStr(3)
	options["backupDir"] = &backupDir

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	for _, v := range fileNameList1 {
		_, err := os.Stat(dirName2 + string(os.PathSeparator) + v)
		c.Assert(err, NotNil)
	}

	for _, v := range fileNameList1 {
		_, err := os.Stat(backupDir + string(os.PathSeparator) + v)
		c.Assert(err, IsNil)
	}

	// subdir is delete
	_, err = os.Stat(dirName + string(os.PathSeparator) + subDirName)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncBackupSubDirIsFile(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	fileName1 := "test-ossutil-file-" + randStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	os.MkdirAll(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator), 0755)
	s.createFile(dirName+string(os.PathSeparator)+fileName1, "123", c)

	dirName2 := "dest-" + dirName
	fileName2 := "test-ossutil-file-" + randStr(5)
	os.MkdirAll(dirName2, 0755)
	s.createFile(dirName2+string(os.PathSeparator)+fileName2, "123", c)

	// upload dir
	syncArgs := []string{dirName2, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// download
	syncArgs = []string{CloudURLToString(bucketName, ""), dirName}
	backupDir := "test-backup-dir" + randLowStr(3)
	options["backupDir"] = &backupDir
	os.MkdirAll(backupDir, 0755)
	s.createFile(backupDir+string(os.PathSeparator)+subDirName, "123", c)

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncCreateBackupSubDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucket(bucketName, c)

	// dir1
	dirName := "testdir1-" + randLowStr(3)
	fileName1 := "test-ossutil-file-" + randStr(3)
	subDirName := "subdir1-" + randLowStr(4)
	os.MkdirAll(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator), 0755)
	s.createFile(dirName+string(os.PathSeparator)+fileName1, "123", c)

	dirName2 := "dest-" + dirName
	fileName2 := "test-ossutil-file-" + randStr(5)
	os.MkdirAll(dirName2, 0755)
	s.createFile(dirName2+string(os.PathSeparator)+fileName2, "123", c)

	// upload dir
	syncArgs := []string{dirName2, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	bDelete := true
	bForce := true
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"delete":          &bDelete,
		"force":           &bForce,
	}

	_, err := cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	// download
	syncArgs = []string{CloudURLToString(bucketName, ""), dirName}
	backupDir := "test-backup-dir" + randLowStr(3)
	options["backupDir"] = &backupDir

	_, err = cm.RunCommand("sync", syncArgs, options)
	c.Assert(err, IsNil)

	os.RemoveAll(dirName)
	os.RemoveAll(dirName2)
	os.RemoveAll(backupDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadWithDisableAllSymlinkDirSuccess(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// mk symlink dir
	symlinkDir := "ossutil_test_dir_" + randStr(5)
	err = os.Symlink(subDirName, dirName+string(os.PathSeparator)+symlinkDir)
	c.Assert(err, IsNil)

	// file under subdir
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testFileName, data, c)

	// file under dir
	s.createFile(dirName+string(os.PathSeparator)+testFileName, data, c)

	// symlink file under dir
	testSymlinkFile := testFileName + "-symlink"
	err = os.Symlink(testFileName, dirName+string(os.PathSeparator)+testSymlinkFile)
	c.Assert(err, IsNil)

	// begin sync file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	disableAllSymlink := true
	options := OptionMapType{
		"endpoint":          &str,
		"accessKeyID":       &str,
		"accessKeySecret":   &str,
		"configFile":        &configFile,
		"checkpointDir":     &cpDir,
		"routines":          &routines,
		"disableAllSymlink": &disableAllSymlink,
	}

	// upload
	_, err = cm.RunCommand("sync", cpArgs, options)
	c.Assert(err, IsNil)

	// symlink dir object not exist
	_, err = s.rawGetStat(bucketName, symlinkDir+"/"+testFileName)
	c.Assert(err, NotNil)

	// stat sub dir object success
	_, err = s.rawGetStat(bucketName, subDirName+"/"+testFileName)
	c.Assert(err, IsNil)

	// stat dir object success
	_, err = s.rawGetStat(bucketName, testFileName)
	c.Assert(err, IsNil)

	//stat dir symlink failure
	_, err = s.rawGetStat(bucketName, testSymlinkFile)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadOnlyCurrentDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// filename
	testDirFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testDirFileName, data, c)

	// subdir filename
	testSubDirFileName := "ossutil_test_file" + randStr(5)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testSubDirFileName, data, c)

	// begin sync file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	onlyCurrentDir := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		"onlyCurrentDir":  &onlyCurrentDir,
	}

	// sync
	_, err = cm.RunCommand("sync", cpArgs, options)
	c.Assert(err, IsNil)

	// stat dir object success
	_, err = s.rawGetStat(bucketName, testDirFileName)
	c.Assert(err, IsNil)

	// stat sub dir object error
	_, err = s.rawGetStat(bucketName, subDirName+"/"+testSubDirFileName)
	c.Assert(err, NotNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncDownloadOnlyCurrentDir(c *C) {
	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// filename
	testDirFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+testDirFileName, data, c)

	// subdir filename
	testSubDirFileName := "ossutil_test_file" + randStr(5)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testSubDirFileName, data, c)

	// begin sync file
	cpArgs := []string{dirName, CloudURLToString(bucketName, dirName)}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	onlyCurrentDir := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"configFile":      &configFile,
		"checkpointDir":   &cpDir,
		"routines":        &routines,
		//"onlyCurrentDir":  &onlyCurrentDir,
	}

	// upload
	_, err = cm.RunCommand("sync", cpArgs, options)
	c.Assert(err, IsNil)

	// download with onlyCurrentDir
	downDir := dirName + "-down"
	options["onlyCurrentDir"] = &onlyCurrentDir

	// download sub dir object success
	dwArgs := []string{CloudURLToString(bucketName, dirName+"/"), downDir}
	_, err = cm.RunCommand("sync", dwArgs, options)
	c.Assert(err, IsNil)

	// stat dir object success
	_, err = os.Stat(downDir + string(os.PathSeparator) + testDirFileName)
	c.Assert(err, IsNil)

	// stat subdir object error
	_, err = os.Stat(downDir + string(os.PathSeparator) + subDirName + string(os.PathSeparator) + testSubDirFileName)
	c.Assert(err, NotNil)
	os.RemoveAll(dirName)
	os.RemoveAll(downDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestSyncUploadSubSymlinkDir(c *C) {
	if runtime.GOOS == "windows" {
		return
	}

	bucketName := bucketNamePrefix + randLowStr(12)
	s.putBucket(bucketName, c)

	// mkdir
	dirName := "ossutil_test_dir_" + randStr(5)
	err := os.MkdirAll(dirName, 0755)
	c.Assert(err, IsNil)

	subDirName := "ossutil_test_dir_" + randStr(5)
	err = os.MkdirAll(dirName+string(os.PathSeparator)+subDirName, 0755)
	c.Assert(err, IsNil)

	// mk symlink dir
	symlinkDir := "ossutil_test_dir_" + randStr(5)
	err = os.Symlink(subDirName, dirName+string(os.PathSeparator)+symlinkDir)
	c.Assert(err, IsNil)

	// filename
	testFileName := "ossutil_test_file" + randStr(5)
	data := randStr(100)
	s.createFile(dirName+string(os.PathSeparator)+subDirName+string(os.PathSeparator)+testFileName, data, c)

	// begin sync file
	cpArgs := []string{dirName, CloudURLToString(bucketName, "")}
	str := ""
	cpDir := CheckpointDir
	routines := strconv.Itoa(Routines)
	enableSymlinkDir := true
	options := OptionMapType{
		"endpoint":         &str,
		"accessKeyID":      &str,
		"accessKeySecret":  &str,
		"configFile":       &configFile,
		"checkpointDir":    &cpDir,
		"routines":         &routines,
		"enableSymlinkDir": &enableSymlinkDir,
	}

	// sync
	_, err = cm.RunCommand("sync", cpArgs, options)
	c.Assert(err, IsNil)

	// stat symlink dir object
	_, err = s.rawGetStat(bucketName, symlinkDir+"/"+testFileName)
	c.Assert(err, IsNil)

	// download dir object
	_, err = s.rawGetStat(bucketName, subDirName+"/"+testFileName)
	c.Assert(err, IsNil)

	os.RemoveAll(dirName)
	s.removeBucket(bucketName, true, c)
}
