package lib

import (
	"fmt"
	"os"

	. "gopkg.in/check.v1"
)

type OssutilConfigSuite struct{}

var _ = Suite(&OssutilConfigSuite{})

var token = stsToken

// Run once when the suite starts running
func (s *OssutilConfigSuite) SetUpSuite(c *C) {
	os.Stdout = testLogFile
	os.Stderr = testLogFile
	testLogger.Println("test config started")
	SetUpCredential()
	cm.Init()
}

// Run before each test or benchmark starts running
func (s *OssutilConfigSuite) TearDownSuite(c *C) {
	testLogger.Println("test config completed")
	os.Stdout = out
	os.Stderr = errout
}

// Run after each test or benchmark runs
func (s *OssutilConfigSuite) SetUpTest(c *C) {
	if stsToken == "" {
		stsToken = "ststoken"
	}
	os.Remove(configFile)
}

// Run once after all tests or benchmarks have finished running
func (s *OssutilConfigSuite) TearDownTest(c *C) {
	stsToken = token
	os.Remove(configFile)
}

// test "config"
func (s *OssutilConfigSuite) TestConfigNonInteractive(c *C) {
	command := "config"
	var args []string
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &stsToken,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 5)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
	c.Assert(opts[OptionEndpoint], Equals, endpoint)
	c.Assert(opts[OptionAccessKeyID], Equals, accessKeyID)
	c.Assert(opts[OptionAccessKeySecret], Equals, accessKeySecret)
	c.Assert(opts[OptionSTSToken], Equals, stsToken)
}

func (s *OssutilConfigSuite) TestConfigNonInteractiveLanguage(c *C) {
	command := "config"
	var args []string
	for _, language := range []string{DefaultLanguage, EnglishLanguage, LEnglishLanguage} {
		options := OptionMapType{
			"endpoint":   &endpoint,
			"stsToken":   &stsToken,
			"configFile": &configFile,
			"language":   &language,
		}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, IsNil)

		f, err := os.Stat(configFile)
		c.Assert(err, IsNil)
		c.Assert(f.Size() > 0, Equals, true)

		opts, err := LoadConfig(configFile)
		c.Assert(err, IsNil)
		c.Assert(len(opts), Equals, 3)
		c.Assert(opts[OptionEndpoint], Equals, endpoint)
		c.Assert(opts[OptionSTSToken], Equals, stsToken)
		c.Assert(opts[OptionLanguage], Equals, language)
	}
}

func (s *OssutilConfigSuite) TestConfigInteractive(c *C) {
	command := "config"
	var args []string
	options := OptionMapType{
		"configFile": &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 1)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
}

func (s *OssutilConfigSuite) TestConfigInteractiveLanguage(c *C) {
	command := "config"
	var args []string
	for _, language := range []string{DefaultLanguage, EnglishLanguage, LEnglishLanguage} {
		options := OptionMapType{
			"configFile": &configFile,
			"language":   &language,
		}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, IsNil)
	}

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 1)
}

func (s *OssutilConfigSuite) TestConfigLanguageEN(c *C) {
	command := "config"
	var args []string
	language := "En"
	options := OptionMapType{
		"configFile": &configFile,
		"language":   &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 1)
	c.Assert(opts[OptionLanguage], Equals, language)
}

func (s *OssutilConfigSuite) TestConfigLanguageCH(c *C) {
	command := "config"
	var args []string
	language := "CH"
	options := OptionMapType{
		"configFile": &configFile,
		"language":   &language,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 1)
	c.Assert(opts[OptionLanguage], Equals, language)
}

// test option empty value
func (s *OssutilConfigSuite) TestConfigOptionEmptyValue(c *C) {
	command := "config"
	var args []string
	endp := ""
	id := ""
	options := OptionMapType{
		"endpoint":        &endp,
		"accessKeyID":     &id,
		"accessKeySecret": &accessKeySecret,
		"stsToken":        &stsToken,
		"configFile":      &configFile,
	}
	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, IsNil)

	f, err := os.Stat(configFile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(configFile)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 3)
	c.Assert(opts[OptionEndpoint], IsNil)
	c.Assert(opts[OptionAccessKeyID], IsNil)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
	c.Assert(opts[OptionAccessKeySecret], Equals, accessKeySecret)
	c.Assert(opts[OptionSTSToken], Equals, stsToken)
}

// test invalid option
func (s *OssutilConfigSuite) TestConfigInvalidOptions(c *C) {
	command := "config"
	var args []string
	opts := []string{"acl", "bigfileThreshold", "checkpointDir", "retryTimes", "routines", "notexist"}
	for _, name := range opts {
		str := "private"
		options := OptionMapType{name: &str}
		showElapse, err := cm.RunCommand(command, args, options)
		c.Assert(showElapse, Equals, false)
		c.Assert(err, Equals, CommandError{command: "config", reason: fmt.Sprintf("the command does not support option: \"%s\"", name)})
	}
}

func (s *OssutilConfigSuite) TestConfigInvalidOption(c *C) {
	command := "config"
	var args []string
	shortFormat := true
	options := OptionMapType{
		"endpoint":    &endpoint,
		"shortFormat": &shortFormat,
	}

	showElapse, err := cm.RunCommand(command, args, options)
	c.Assert(showElapse, Equals, false)
	c.Assert(err, Equals, CommandError{command: "config", reason: fmt.Sprintf("the command does not support option: \"%s\"", "shortFormat")})
}
