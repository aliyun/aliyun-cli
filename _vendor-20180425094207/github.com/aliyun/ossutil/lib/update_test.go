package lib

import (
	"os"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) rawUpdate(force bool, language string) (bool, error) {
	command := "update"
	var args []string
	options := OptionMapType{
		OptionForce:    &force,
		OptionLanguage: &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	return showElapse, err
}

func (s *OssutilCommandSuite) TestUpdate(c *C) {
	showElapse, err := s.rawUpdate(false, "ch")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, false)

	showElapse, err = s.rawUpdate(false, "En")
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, false)

	cmdline := []string{"ossutil", "update"}
	os.Args = cmdline

	showElapse, err = s.rawUpdate(true, "ch")

	showElapse, err = s.rawUpdate(true, "En")

	err = updateCommand.updateVersion(Version, "ch")

	err = updateCommand.updateVersion("1.0.0.Beta", "ch")

	fileName := "ossutil_test_not_exist"
	err = updateCommand.rewriteLoadConfig(fileName)
	c.Assert(err, IsNil)
}

func (s *OssutilCommandSuite) TestUpdateDiffVersion(c *C) {
	// error get lastest version
	ue := vUpdateBucket
	vUpdateBucket = "abc"
	version, err := updateCommand.getLastestVersion()
	c.Assert(err, NotNil)

	vUpdateBucket = ue
	version, err = updateCommand.getLastestVersion()
	c.Assert(err, IsNil)
	vVersion = version

	cmdline := []string{"ossutil", "update"}
	os.Args = cmdline

	err = updateCommand.RunCommand()
	c.Assert(err, IsNil)
	vVersion = version + "123"
	updateCommand.RunCommand()
	vVersion = Version
}

func (s *OssutilCommandSuite) TestRevertRename(c *C) {
	filePath := ".ossutil_tempf" + randStr(5)
	renameFilePath := ".ossutil_tempr" + randStr(5)

	s.createFile(filePath, filePath+"i", c)
	s.createFile(renameFilePath, renameFilePath+"i", c)

	updateCommand.revertRename(filePath, renameFilePath)
	_, err := os.Stat(renameFilePath)
	c.Assert(err, NotNil)

	str := s.readFile(filePath, c)
	c.Assert(str, Equals, renameFilePath+"i")

	os.Remove(filePath)
	os.Remove(renameFilePath)

	renameFilePath = ".ossutil_notexist"
	err = updateCommand.revertRename(filePath, renameFilePath)
	c.Assert(err, NotNil)
}

func (s *OssutilCommandSuite) TestDownloadLastestBinary(c *C) {
	tempBinaryFile := ".ossutil_test_update.temp"
	err := updateCommand.getBinary(tempBinaryFile, "1.0.0.Beta")
	c.Assert(err, IsNil)

	os.Remove(tempBinaryFile)
}

func (s *OssutilCommandSuite) TestAnonymousGetToFileError(c *C) {
	bucket := bucketNameNotExist
	object := "TestAnonymousGetToFileError"
	err := updateCommand.anonymousGetToFileRetry(bucket, object, object)
	c.Assert(err, NotNil)

	bucket = bucketNameDest
	s.putObject(bucket, object, uploadFileName, c)
	fileName := "*"
	err = updateCommand.anonymousGetToFileRetry(bucket, object, fileName)
	c.Assert(err, NotNil)
}
