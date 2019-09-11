package lib

import (
	"fmt"
	"os"

	. "gopkg.in/check.v1"
)

type OssutilHelpSuite struct{}

var _ = Suite(&OssutilHelpSuite{})

// Run once when the suite starts running
func (s *OssutilHelpSuite) SetUpSuite(c *C) {
	fmt.Printf("set up suite OssutilHelpSuite\n")
	testLogger.Println("test help started")
	os.Stdout = testLogFile
	os.Stderr = testLogFile
	SetUpCredential()
	cm.Init()
}

// Run before each test or benchmark starts running
func (s *OssutilHelpSuite) TearDownSuite(c *C) {
	fmt.Printf("tear down suite OssutilHelpSuite\n")
	testLogger.Println("test help completed")
	os.Remove(configFile)
	os.Stdout = out
	os.Stderr = errout
}

// Run after each test or benchmark runs
func (s *OssutilHelpSuite) SetUpTest(c *C) {
	fmt.Printf("set up test:%s\n", c.TestName())
}

// Run once after all tests or benchmarks have finished running
func (s *OssutilHelpSuite) TearDownTest(c *C) {
	fmt.Printf("tear down test:%s\n", c.TestName())
}

// test "help"
func (s *OssutilHelpSuite) TestHelp(c *C) {
	command := "help"
	var args []string
	var options OptionMapType
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)
}

// test "help"
func (s *OssutilHelpSuite) TestHelpChinese(c *C) {
	command := "help"
	var args []string
	language := DefaultLanguage
	options := OptionMapType{
		"language": &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)
}

// test "help"
func (s *OssutilHelpSuite) TestHelpEnglish(c *C) {
	command := "help"
	var args []string
	language := EnglishLanguage
	options := OptionMapType{
		"language": &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)
}

func (s *OssutilHelpSuite) TestHelpLEnglish(c *C) {
	command := "help"
	var args []string
	language := LEnglishLanguage
	options := OptionMapType{
		"language": &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)
}

// test "help options"
func (s *OssutilHelpSuite) TestHelpOption(c *C) {
	command := "help"
	var args []string
	for _, option := range OptionMap {
		str := "abc"
		options := OptionMapType{option.name: &str}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, Equals, CommandError{command: "help", reason: fmt.Sprintf("the command does not support option: \"%s\"", option.name)})
	}
}

// test "help options" with not exist options
var notExistOptions = []string{"notexist", "abc", "CONFIG_FILE", "ACCESSKEY_ID", "accesskey_id"}

func (s *OssutilHelpSuite) TestHelpNotExistOption(c *C) {
	command := "help"
	var args []string
	for _, option := range notExistOptions {
		str := "abc"
		options := OptionMapType{option: &str}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, Equals, CommandError{command: "help", reason: fmt.Sprintf("the command does not support option: \"%s\"", option)})
	}
}

// test "help command"
type helpCommandTestCase struct {
	subCommand string
	showElapse bool
	err        error
}

var subCommands = []string{"help", "config", "hash", "update", "mb", "ls", "rm", "stat", "set-acl", "set-meta", "cp", "restore", "create-symlink", "read-symlink"}

func (s *OssutilHelpSuite) TestHelpCommand(c *C) {
	command := "help"
	for _, language := range []string{DefaultLanguage, EnglishLanguage, LEnglishLanguage} {
		options := OptionMapType{
			"language": &language,
		}
		for _, subCmd := range subCommands {
			args := []string{subCmd}
			showElapse, err := cm.RunCommand(command, args, options)
			c.Assert(showElapse, Equals, false)
			c.Assert(err, IsNil)
		}
	}
}

var notExistSubCommands = []string{"hp", "pb", "list", "remove", "delete", "info", "set_acl", "set_meta", "copy", "notexit"}

func (s *OssutilHelpSuite) TestHelpNotExistCommand(c *C) {
	command := "help"
	var options OptionMapType
	for _, subCmd := range notExistSubCommands {
		args := []string{subCmd}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, NotNil)
	}
}

// test "help command options"
func (s *OssutilHelpSuite) TestHelpCommandOption(c *C) {
	command := "help"
	for _, subCmd := range subCommands {
		for _, option := range OptionMap {
			args := []string{subCmd}
			str := "abc"
			options := OptionMapType{option.name: &str}
			showElapse, err := cm.RunCommand(command, args, options)
			c.Assert(showElapse, Equals, false)
			c.Assert(err, NotNil)
		}
	}
}

func (s *OssutilHelpSuite) TestHelpLoadConfig(c *C) {
	fileName := "notexistconfigfile"
	err := helpCommand.rewriteLoadConfig(fileName)
	c.Assert(err, IsNil)

	err = helpCommand.rewriteLoadConfig(configFile)
	c.Assert(err, IsNil)
}
