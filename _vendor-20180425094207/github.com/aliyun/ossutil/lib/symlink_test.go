package lib

import (
	"fmt"
	"net/url"

	. "gopkg.in/check.v1"
)

func (s *OssutilCommandSuite) TestCreateSymlink(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)
	notExistBucketName := bucketNamePrefix + randLowStr(10)

	symObject := bucketNamePrefix + randStr(5) + "符号链接"
	targetObject := bucketNamePrefix + randStr(5) + "符号链接目标"
	targetObject1 := bucketNamePrefix + randStr(5) + "target"

	// put symlink to not exist bucket
	cmdline := fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject)
	err := s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	s.putBucket(bucketName, c)

	data := "中文内容"
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, targetObject, uploadFileName, c)

	data1 := "english"
	s.createFile(uploadFileName, data1, c)
	s.putObject(bucketName, targetObject1, uploadFileName, c)

	// put symlink to different bucket
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), CloudURLToString(notExistBucketName, targetObject))
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	s.getObject(bucketName, symObject, downloadFileName, c)
	str := s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	// put symlink again
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject1)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	s.getObject(bucketName, symObject, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data1)

	// error put symlink
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject1, "abc")
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, NotNil)

	// put symlink with urlencoding
	urlTarget := url.QueryEscape(targetObject)
	c.Assert(urlTarget != targetObject, Equals, true)

	cmdline = fmt.Sprintf("%s %s --encoding-type url", CloudURLToString(bucketName, url.QueryEscape(symObject)), urlTarget)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	s.getObject(bucketName, symObject, downloadFileName, c)
	str = s.readFile(downloadFileName, c)
	c.Assert(str, Equals, data)

	// error create symlink error args
	cmdline = fmt.Sprintf("%s %s --encoding-type url", CloudURLToString("", url.QueryEscape(symObject)), urlTarget)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// error create symlink error args
	cmdline = fmt.Sprintf("%s %s --encoding-type url", CloudURLToString(bucketName, ""), urlTarget)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// error create symlink error args
	cmdline = fmt.Sprintf("%s %s --encoding-type url", "oss://", urlTarget)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// error create symlink error args
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), CloudURLToString("", urlTarget))
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// error create symlink error args
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), CloudURLToString(bucketName, ""))
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	s.removeBucket(bucketName, true, c)
}

func (s *OssutilCommandSuite) TestReadSymlink(c *C) {
	bucketName := bucketNamePrefix + randLowStr(10)

	symObject := bucketNamePrefix + randStr(5) + "符号链接"
	targetObject := bucketNamePrefix + randStr(5) + "目标文件"

	// read symlink of not exist bucket
	cmdline := CloudURLToString(bucketName, symObject)
	err := s.initReadSymlink(cmdline)
	c.Assert(err, IsNil)
	err = readSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	s.putBucket(bucketName, c)

	// read symlink of not exist object
	cmdline = CloudURLToString(bucketName, symObject)
	err = s.initReadSymlink(cmdline)
	c.Assert(err, IsNil)
	err = readSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// create target object
	data := "中文内容"
	s.createFile(uploadFileName, data, c)
	s.putObject(bucketName, targetObject, uploadFileName, c)

	// put symlink
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	// read symlink
	cmdline = CloudURLToString(bucketName, symObject)
	linkStat := s.readSymlink(cmdline, c)
	c.Assert(linkStat["X-Oss-Symlink-Target"], Equals, targetObject)

	// put symObject again
	s.putObject(bucketName, symObject, uploadFileName, c)

	// read symlink of normal object
	cmdline = CloudURLToString(bucketName, symObject)
	err = s.initReadSymlink(cmdline)
	c.Assert(err, IsNil)
	err = readSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	// put symlink again
	cmdline = fmt.Sprintf("%s %s", CloudURLToString(bucketName, symObject), targetObject)
	err = s.initCreateSymlink(cmdline)
	c.Assert(err, IsNil)
	err = createSymlinkCommand.RunCommand()
	c.Assert(err, IsNil)

	// read symlink with encoding-type
	urlObject := url.QueryEscape(symObject)
	c.Assert(urlObject != symObject, Equals, true)

	cmdline = fmt.Sprintf("%s", CloudURLToString(bucketName, urlObject))
	err = s.initReadSymlink(cmdline)
	c.Assert(err, IsNil)
	err = readSymlinkCommand.RunCommand()
	c.Assert(err, NotNil)

	cmdline = fmt.Sprintf("%s --encoding-type url", CloudURLToString(bucketName, urlObject))
	linkStat = s.readSymlink(cmdline, c)
	c.Assert(linkStat["X-Oss-Symlink-Target"], Equals, targetObject)

	// read link error cloudURL
	for _, line := range []string{"oss://", CloudURLToString("", symObject), CloudURLToString(bucketName, "")} {
		cmdline = line
		err = s.initReadSymlink(cmdline)
		c.Assert(err, IsNil)
		err = readSymlinkCommand.RunCommand()
		c.Assert(err, NotNil)
	}

	s.removeBucket(bucketName, true, c)
}
