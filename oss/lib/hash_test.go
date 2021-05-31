package lib

import (
	. "gopkg.in/check.v1"
	"os"
)

func (s *OssutilCommandSuite) TestErrorInputFile(c *C) {
	command := "hash"
	fakeFileName := randStr(10)
	args := []string{fakeFileName}
	hashType := DefaultHashType
	options := OptionMapType{
		"hash": &hashType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestErrorHashType(c *C) {
	command := "hash"
	hashType := "crc256"
	fakeFileName := randStr(10)
	args := []string{fakeFileName}
	options := OptionMapType{
		"hashType": &hashType,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, NotNil)
	c.Assert(showElapse, Equals, false)
}

func (s *OssutilCommandSuite) TestCrc64(c *C) {
	command := "hash"
	content = "this is content"
	s.createFile(inputFileName, content, c)

	args := []string{inputFileName}
	hashType := DefaultHashType
	options := OptionMapType{
		"hashType": &hashType,
	}

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	saveOut := os.Stdout
	os.Stdout = testResultFile

	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, false)

	hashStat := s.getHashResults(c)
	c.Assert(hashStat[HashCRC64], Equals, "2863152195715871371")

	os.Stdout = saveOut

	os.Remove(inputFileName)
}

func (s *OssutilCommandSuite) TestMd5(c *C) {
	command := "hash"
	content = "this is content"
	s.createFile(inputFileName, content, c)

	args := []string{inputFileName}
	hashType := MD5HashType
	options := OptionMapType{
		"hashType": &hashType,
	}

	testResultFile, _ = os.OpenFile(resultPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)

	saveOut := os.Stdout
	os.Stdout = testResultFile

	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(err, IsNil)
	c.Assert(showElapse, Equals, false)

	hashStat := s.getHashResults(c)
	c.Assert(hashStat[HashMD5], Equals, "B7FCEF7FE745F2A95560FF5F550E3B8F")
	c.Assert(hashStat[HashContentMD5], Equals, "t/zvf+dF8qlVYP9fVQ47jw==")

	os.Stdout = saveOut

	os.Remove(inputFileName)
}
