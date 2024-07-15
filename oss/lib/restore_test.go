package lib

import (
	"encoding/xml"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestRestoreObject(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	//put object to archive bucket
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

func (s *OssutilCommandSuite) TestRestoreObjectErrorObj(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageColdArchive, c)
	bucketNameIA := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketNameIA, StorageIA, c)

	//put object to cold archive bucket
	object := "恢复文件" + randStr(5)
	s.putObject(bucketName, object, uploadFileName, c)
	s.putObject(bucketNameIA, object, uploadFileName, c)

	objectStat := s.getStat(bucketName, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketNameIA, object, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageIA)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// not exist bucket
	err := s.initRestoreObject([]string{CloudURLToString("xx", object)}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// not exist object
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, object+"xx")}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "StatusCode=404"), Equals, true)

	// bucket is none
	err = s.initRestoreObject([]string{CloudURLToString("", "")}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// error days
	restoreXml := `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>99</Days>
        <JobParameters>
            <Tier>xxx</Tier>
        </JobParameters>
    </RestoreRequest>`
	restoreConfName := "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, object), restoreConfName}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// restore object(class ia)
	err = s.initRestoreObject([]string{CloudURLToString(bucketNameIA, object)}, "", DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
	s.removeBucket(bucketNameIA, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileBasic(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	data := randStr(20)
	s.createFile(uploadFileName, data, c)
	object1 := "restore" + randStr(5)
	object2 := "restore" + randStr(5)
	object3 := "restore" + randStr(5)
	object4 := "restore" + randStr(5)
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketName, object3, uploadFileName, c)
	s.putObject(bucketName, object4, uploadFileName, c)

	objectStat := s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	content := object1 + "\n" + object2 + "\n" + object3 + "\n" + object4
	s.createFile(objectFileName, content, c)

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	objectStat = s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object3, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object4, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(objectFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileErrorObjFile(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)
	bucketNameIA := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketNameIA, StorageIA, c)

	data := randStr(20)
	s.createFile(uploadFileName, data, c)
	object1 := "restore" + randStr(5)
	object2 := "restore" + randStr(5)
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketNameIA, object1, uploadFileName, c)

	objectStat := s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketNameIA, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageIA)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	// file size 0
	s.createFile(objectFileName, "", c)

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)

	// receive file but give dir
	err = os.Mkdir(fmt.Sprintf("./%s", objectFileName), os.ModePerm)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)

	// not exist file
	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, fmt.Sprintf("--object-file %s", objectFileName+"xxx"), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileErrVer(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	object1 := "restore" + randStr(5)
	object2 := "restore" + randStr(5)
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)

	content := object1 + "\n" + object2
	s.createFile(objectFileName, content, c)

	command := "restore"
	args := []string{CloudURLToString(bucketName, "")}
	versionId := "xxx"
	str := ""
	routines := strconv.Itoa(Routines)
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionId,
		"routines":        &routines,
		"objectFile":      &objectFileName,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	os.Remove(objectFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileWithConfCA(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageColdArchive, c)

	// put object to coldArchive bucket
	object1 := "restore" + randStr(5)
	object2 := "restore" + randStr(5)
	object3 := "restore" + randStr(5)
	object4 := "restore" + randStr(5)
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketName, object3, uploadFileName, c)
	s.putObject(bucketName, object4, uploadFileName, c)

	// get object status
	objectStat := s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	restoreXml := `<?xml version="1.0" encoding="UTF-8"?>
  <RestoreRequest>
      <Days>2</Days>
      <JobParameters>
          <Tier>Bulk</Tier>
      </JobParameters>
  </RestoreRequest>`

	rulesConfigSrc := oss.RestoreConfiguration{}
	err := xml.Unmarshal([]byte(restoreXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	restoreConfName := "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content := object1 + "\n" + object2
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(objectFileName)
	os.Remove(restoreConfName)

	// conf only with days
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
  <RestoreRequest>
      <Days>7</Days>
  </RestoreRequest>`
	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content = object3
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	objectStat = s.getStat(bucketName, object3, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(objectFileName)
	os.Remove(restoreConfName)

	// error Tier
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>99</Days>
        <JobParameters>
            <Tier>xxx</Tier>
        </JobParameters>
    </RestoreRequest>`
	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content = object4
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s --disable-ignore-error", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// error Days
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>399</Days>
        <JobParameters>
            <Tier>Bulk</Tier>
        </JobParameters>
    </RestoreRequest>`
	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content = object4
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s --disable-ignore-error", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	// error xml
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>4</Days>`
	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s --disable-ignore-error", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)
	os.Remove(restoreConfName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileWithConfAr(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	object1 := "restore" + randStr(5)
	object2 := "restore" + randStr(5)
	object3 := "restore" + randStr(5)
	object4 := "restore" + randStr(5)
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketName, object3, uploadFileName, c)
	s.putObject(bucketName, object4, uploadFileName, c)

	// get object status
	objectStat := s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	restoreXml := `<?xml version="1.0" encoding="UTF-8"?>
     <RestoreRequest>
   	     <Days>7</Days>
     </RestoreRequest>`

	rulesConfigSrc := oss.RestoreConfiguration{}
	err := xml.Unmarshal([]byte(restoreXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	restoreConfName := "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content := object1 + "\n" + object2
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(restoreConfName)

	// err conf
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
	  <RestoreRequest>
		  <Days>2</Days>
		  <JobParameters>
			  <Tier>Bulk</Tier>
		  </JobParameters>
	  </RestoreRequest>`

	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content = object3
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s --disable-ignore-error", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "StatusCode=400"), Equals, true)

	os.Remove(restoreConfName)

	// err days
	restoreXml = `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>99</Days>
    </RestoreRequest>`

	rulesConfigSrc = oss.RestoreConfiguration{}
	err = xml.Unmarshal([]byte(restoreXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	restoreConfName = "test-ossutil-" + randLowStr(12)
	s.createFile(restoreConfName, restoreXml, c)

	content = object4
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, ""), restoreConfName}, fmt.Sprintf("--object-file %s --disable-ignore-error", objectFileName), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "StatusCode=400"), Equals, true)

	os.Remove(objectFileName)
	os.Remove(restoreConfName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileSnap(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	object1 := "restore_1"
	object2 := "restore_2"
	object3 := "restore_3"
	object4 := "restore_4"
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketName, object3, uploadFileName, c)
	s.putObject(bucketName, object4, uploadFileName, c)

	objectStat := s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object3, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object4, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	content := object1 + "\n" + object2
	s.createFile(objectFileName, content, c)

	snapShotDir := "ossutil_test_snapshot" + randStr(5)
	cmd := fmt.Sprintf("--object-file %s --snapshot-path %s", objectFileName, snapShotDir)

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, cmd, DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object1, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object2, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object3, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	objectStat = s.getStat(bucketName, object4, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok = objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	os.Remove(objectFileName)

	content = object1 + "\n" + object2 + "\n" + object3 + "\n" + object4
	s.createFile(objectFileName, content, c)

	err = s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, cmd, DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, object3, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	objectStat = s.getStat(bucketName, object4, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(objectFileName)
	os.Remove(snapShotDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectFileErrorSnap(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)
	bucketNameIA := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketNameIA, StorageIA, c)

	object1 := "restore_1"
	object2 := "restore_2"
	s.putObject(bucketName, object1, uploadFileName, c)
	s.putObject(bucketName, object2, uploadFileName, c)
	s.putObject(bucketNameIA, object1, uploadFileName, c)

	content := object1 + "\n" + object2
	s.createFile(objectFileName, content, c)

	// create file which name same as snapshotPath
	snapShotDir := "ossutil_test_snapshot" + randStr(5)
	s.createFile(snapShotDir, content, c)
	cmd := fmt.Sprintf("--object-file %s --snapshot-path %s", objectFileName, snapShotDir)

	err := s.initRestoreObject([]string{CloudURLToString(bucketName, "")}, cmd, DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)
	os.Remove(snapShotDir)

	// restore object(class ia)
	content = object1
	s.createFile(objectFileName, content, c)

	snapShotDir = "ossutil_test_snapshot" + randStr(5)

	err = s.initRestoreObject([]string{CloudURLToString(bucketNameIA, "")}, fmt.Sprintf("--object-file %s --snapshot-path %s --disable-ignore-error", objectFileName, snapShotDir), DefaultOutputDir)
	c.Assert(err, IsNil)
	err = restoreCommand.RunCommand()
	c.Assert(err, NotNil)

	os.Remove(objectFileName)
	os.Remove(snapShotDir)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithVersionError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	//put object to archive bucket
	object := "恢复文件" + randStr(5)
	s.putObject(bucketName, object, uploadFileName, c)

	// -r & --version-id error
	command := "restore"
	args := []string{CloudURLToString(bucketName, object)}
	str := ""
	versionId := "xxx"
	r := true
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"recursive":       &r,
		"versionId":       &versionId,
	}
	_, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

	// only --version-id
	options = OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
		"versionId":       &versionId,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)

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
	c.Assert(ok, Equals, true)

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
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(strings.Contains(err.Error(), "StatusCode=400"), Equals, true)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithPayer(c *C) {
	bucketName := payerBucket + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)
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
	s.createFile(uploadFileName, content, c)

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
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)

	//put object, with --payer=requester
	args = []string{uploadFileName, CloudURLToString(bucketName, "")}
	showElapse, err = s.rawCPWithPayer(args, false, true, false, DefaultBigFileThreshold, "requester")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, true)

	// stat with payer
	requester = "request"
	options = OptionMapType{
		"endpoint":        &payerBucketEndPoint,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &payerConfigFile,
		"payer":           &requester,
	}
	_, err = cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithConfigColdArchiveSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageColdArchive, c)

	// put object to archive bucket
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)

	data := randStr(20)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)
	os.Remove(testFileName)

	// get object status
	objectStat := s.getStat(bucketName, objectName, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	restoreXml := `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>2</Days>
        <JobParameters>
            <Tier>Bulk</Tier>
        </JobParameters>
    </RestoreRequest>`

	rulesConfigSrc := oss.RestoreConfiguration{}
	err := xml.Unmarshal([]byte(restoreXml), &rulesConfigSrc)
	c.Assert(err, IsNil)

	restoreFileName := "test-ossutil-" + randLowStr(12)
	s.createFile(restoreFileName, restoreXml, c)

	//restore command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	restoreArgs := []string{CloudURLToString(bucketName, objectName), restoreFileName}
	_, err = cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, objectName, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageColdArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(restoreFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithConfigArchiveSuccess(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageArchive, c)

	// put object to archive bucket
	objectName := "ossutil_test_object" + randStr(5)
	testFileName := "ossutil_test_file" + randStr(5)

	data := randStr(20)
	s.createFile(testFileName, data, c)
	s.putObject(bucketName, objectName, testFileName, c)
	os.Remove(testFileName)

	// get object status
	objectStat := s.getStat(bucketName, objectName, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	_, ok := objectStat["X-Oss-Restore"]
	c.Assert(ok, Equals, false)

	restoreXml := `<?xml version="1.0" encoding="UTF-8"?>
    <RestoreRequest>
        <Days>2</Days>
    </RestoreRequest>`

	restoreFileName := "test-ossutil-" + randLowStr(12)
	s.createFile(restoreFileName, restoreXml, c)

	//restore command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	restoreArgs := []string{CloudURLToString(bucketName, objectName), restoreFileName}
	_, err := cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, IsNil)

	// get object status
	objectStat = s.getStat(bucketName, objectName, c)
	c.Assert(objectStat["X-Oss-Storage-Class"], Equals, StorageArchive)
	c.Assert(objectStat["X-Oss-Restore"], Equals, "ongoing-request=\"true\"")

	os.Remove(restoreFileName)
	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestRestoreObjectWithConfigError(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	s.putBucketWithStorageClass(bucketName, StorageColdArchive, c)

	// file is not exist
	objectName := "test-ossutil-" + randLowStr(12)
	restoreFileName := "test-ossutil-" + randLowStr(12)

	//restore command test
	var str string
	options := OptionMapType{
		"endpoint":        &str,
		"accessKeyID":     &str,
		"accessKeySecret": &str,
		"stsToken":        &str,
		"configFile":      &configFile,
	}

	restoreArgs := []string{CloudURLToString(bucketName, objectName), restoreFileName}
	_, err := cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, NotNil)

	// empty file
	s.createFile(restoreFileName, "", c)
	_, err = cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, NotNil)

	// invalid xml file
	os.Remove(restoreFileName)
	s.createFile(restoreFileName, "abc", c)
	_, err = cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, NotNil)

	// is dir
	os.Remove(restoreFileName)
	os.MkdirAll(restoreFileName, 0755)
	_, err = cm.RunCommand("restore", restoreArgs, options)
	c.Assert(err, NotNil)
	os.RemoveAll(restoreFileName)
	s.removeBucket(bucketName, true, c)
}

// TestRestoreProducer test restoreProducer
func (s *OssutilCommandSuite) TestRestoreProducer(c *C) {
	chObjects := make(chan string, ChannelBuf)
	chListError := make(chan error, 1)
	var filters []filterOptionType
	restoreCommand.restoreProducer("no_exist_file", chObjects, chListError, filters)
	err := <-chListError
	c.Assert(err, NotNil)
	select {
	case _, ok := <-chObjects:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}

	emptyContentFileName := "empty.txt"
	os.Remove(emptyContentFileName)
	s.createFile(emptyContentFileName, "     ", c)
	chObjects2 := make(chan string, ChannelBuf)
	chListError2 := make(chan error, 1)
	restoreCommand.restoreProducer(emptyContentFileName, chObjects2, chListError2, filters)
	err = <-chListError2
	c.Assert(err, NotNil)
	select {
	case _, ok := <-chObjects2:
		testLogger.Printf("chObjects channel has closed")
		c.Assert(ok, Equals, false)
	default:
		testLogger.Printf("chObjects no data")
		c.Assert(true, Equals, false)
	}

	os.Remove(emptyContentFileName)
}
