package lib

import (
	"fmt"
	"net/url"
	"strings"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestRestoreObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	object := "恢复文件" + randStr(5)
	data := randStr(20)
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, object, uploadFileName, c)

	// get object status
	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// restore encoding object
	err := s.initRestoreObject([]string{CloudURLToString(bucketName, url.QueryEscape(object))}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// get object status
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// restore encoding object
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, url.QueryEscape(object))}, "--encoding-type url", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	// restore object
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, object)}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectErrArgs(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	object := randStr(20)

	err := s.initRestoreObject([]string{CloudURLToString("", object)}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchRestoreObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	num := 3
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("恢复object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	// get object status
	objectStat := s.getStat(bucketName, objectNames[0], c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// batch restore object without -r
	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// batch restore object without -f
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "-r", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, objectNames[0], c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// batch restore with encoding
	prefix := url.QueryEscape("恢复")
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, prefix)}, "-rf --encoding-type url", DefaultOutputDir)
	c.Assert(err, IsNil)
	restoreCommand.monitor.init("Restored")

	str := restoreCommand.monitor.progressBar(false, normalExit)
	c.Assert(str != "", Equals, true)
	str = restoreCommand.monitor.progressBar(false, errExit)
	c.Assert(str != "", Equals, true)
	str = restoreCommand.monitor.progressBar(true, normalExit)
	c.Assert(str != "", Equals, true)
	restoreCommand.monitor.finish = false
	str = restoreCommand.monitor.progressBar(true, errExit)
	c.Assert(str != "", Equals, true)

	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	str = restoreCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := restoreCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(3))
	c.Assert(snap.errNum, Equals, int64(0))
	c.Assert(snap.dealNum, Equals, int64(3))

	restoreCommand.monitor.seekAheadEnd = true
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("%d", 3)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "succeed:"), Equals, true)
	c.Assert(strings.Contains(str, fmt.Sprintf("%d", 3)), Equals, true)
	c.Assert(strings.Contains(str, "err"), Equals, false)

	restoreCommand.monitor.seekAheadEnd = true
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
		c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")
	}

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
		c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchRestoreNotExistBucket(c *C) {
	// restore notexist bucket
	err := s.initRestoreObject([]string{CloudURLToString(bucketNamePrefix+randLowStr(10), "")}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestBatchRestoreErrorContinue(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("恢复object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	bucket, err := restoreCommand.command.ossBucket(bucketName)
	c.Assert(err, IsNil)
	c.Assert(bucket, NotNil)

	// restore prepare
	cloudURL, err := CloudURLFromString(CloudURLToString(bucketName, ""), "")
	c.Assert(err, IsNil)

	restoreCommand.monitor.init("Restored")
	restoreCommand.reOption.ctnu = true

	// init reporter
	restoreCommand.reOption.reporter, err = GetReporter(restoreCommand.reOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer restoreCommand.reOption.reporter.Clear()

	var routines int64
	routines = 3
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	chObjects <- objectNames[0]
	chObjects <- "notexistobject" + randStr(3)
	chObjects <- objectNames[1]
	chListError <- nil
	close(chObjects)

	for i := 0; int64(i) < routines; i++ {
		restoreCommand.restoreConsumer(bucket, cloudURL, chObjects, chError)
	}

	err = restoreCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, IsNil)

	str := restoreCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := restoreCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(2))
	c.Assert(snap.errNum, Equals, int64(1))
	c.Assert(snap.dealNum, Equals, int64(3))

	restoreCommand.monitor.seekAheadEnd = true
	restoreCommand.monitor.seekAheadError = nil
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	restoreCommand.monitor.seekAheadEnd = true
	restoreCommand.monitor.seekAheadError = nil
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
		c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestBatchRestoreErrorBreak(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	num := 2
	objectNames := []string{}
	for i := 0; i < num; i++ {
		object := fmt.Sprintf("恢复object:%d%s", i, randStr(5))
		s.putObject(bucketName, object, uploadFileName, c)
		objectNames = append(objectNames, object)
	}

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, "-rf", DefaultOutputDir)
	c.Assert(err, IsNil)

	// make error bucket with error id
	bucket := s.getErrorOSSBucket(bucketName, c)
	c.Assert(bucket, NotNil)

	// restore prepare
	cloudURL, err := CloudURLFromString(CloudURLToString(bucketName, ""), "")
	c.Assert(err, IsNil)

	restoreCommand.monitor.init("Restored")
	restoreCommand.reOption.ctnu = true

	// init reporter
	restoreCommand.reOption.reporter, err = GetReporter(restoreCommand.reOption.ctnu, DefaultOutputDir, commandLine)
	c.Assert(err, IsNil)

	defer restoreCommand.reOption.reporter.Clear()

	var routines int64
	routines = 3
	chObjects := make(chan string, ChannelBuf)
	chError := make(chan error, routines+1)
	chListError := make(chan error, 1)

	chObjects <- objectNames[0]
	chObjects <- objectNames[1]
	chListError <- nil
	close(chObjects)

	for i := 0; int64(i) < routines; i++ {
		restoreCommand.restoreConsumer(bucket, cloudURL, chObjects, chError)
	}

	err = restoreCommand.waitRoutinueComplete(chError, chListError, routines)
	c.Assert(err, NotNil)

	str := restoreCommand.monitor.progressBar(false, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(false, errExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, normalExit)
	c.Assert(str, Equals, "")
	str = restoreCommand.monitor.progressBar(true, errExit)
	c.Assert(str, Equals, "")

	snap := restoreCommand.monitor.getSnapshot()
	c.Assert(snap.okNum, Equals, int64(0))
	c.Assert(snap.errNum, Equals, int64(2))
	c.Assert(snap.dealNum, Equals, int64(2))

	restoreCommand.monitor.seekAheadEnd = true
	restoreCommand.monitor.seekAheadError = nil
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(normalExit))
	c.Assert(strings.Contains(str, "finishwitherror:"), Equals, true)
	c.Assert(strings.Contains(str, "succeed:"), Equals, false)
	c.Assert(strings.Contains(str, "error"), Equals, true)

	restoreCommand.monitor.seekAheadEnd = true
	restoreCommand.monitor.seekAheadError = nil
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "total"), Equals, true)
	restoreCommand.monitor.seekAheadEnd = false
	str = strings.ToLower(restoreCommand.monitor.getFinishBar(errExit))
	c.Assert(strings.Contains(str, "when error happens."), Equals, true)
	c.Assert(strings.Contains(str, "scanned"), Equals, true)

	for _, object := range objectNames {
		objectStat := s.getStat(bucketName, object, c)
		c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
		_, ok := objectStat["X-Oss-Restore"]
		c.Assert(ok, Equals, false)
	}

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithPayerError400(c *C) {
	s.createFile(uploadFileName, content, c)
	bucketName := payerBucket

	//put object, with --payer=requester
	args := []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err := s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// stat with payer
	command := "restore"
	args = []string{CloudURLToString(bucketName, uploadFileName)}
	str := ""
	requester := "requester"
	options := OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "StatusCode=400"), Equals, true)
}
