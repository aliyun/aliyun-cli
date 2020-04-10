package lib

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strings"

	. "gopkg.in/check.v1"
)

type OssutilConfigSuite struct{}

var _ = Suite(&OssutilConfigSuite{})

var token = stsToken

// Run once when the suite starts running
func (s *OssutilConfigSuite) SetUpSuite(c *C) {
	fmt.Printf("set up OssutilConfigSuite\n")
	os.Stdout = testLogFile
	os.Stderr = testLogFile
	testLogger.Println("test config started")
	SetUpCredential()
	cm.Init()
}

// Run before each test or benchmark starts running
func (s *OssutilConfigSuite) TearDownSuite(c *C) {
	fmt.Printf("tear down OssutilConfigSuite\n")
	testLogger.Println("test config completed")
	os.Stdout = out
	os.Stderr = errout
}

// Run after each test or benchmark runs
func (s *OssutilConfigSuite) SetUpTest(c *C) {
	fmt.Printf("set up test:%s\n", c.TestName())
	if stsToken == "" {
		stsToken = "ststoken"
	}
	os.Remove(configFile)
}

// Run once after all tests or benchmarks have finished running
func (s *OssutilConfigSuite) TearDownTest(c *C) {
	fmt.Printf("tear down test:%s\n", c.TestName())
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

func (s *OssutilConfigSuite) TestConfigNotConfigFile(c *C) {
	configCommand.runCommandInteractive("", LEnglishLanguage)
	contents, _ := ioutil.ReadFile(logPath)
	LogContent := string(contents)
	c.Assert(strings.Contains(LogContent, "Please enter the config file name"), Equals, true)

	configCommand.runCommandInteractive("", ChineseLanguage)
	contents, _ = ioutil.ReadFile(logPath)
	LogContent = string(contents)
	c.Assert(strings.Contains(LogContent, "请输入配置文件名"), Equals, true)
}

func (s *OssutilConfigSuite) TestConfigConfigInteractive(c *C) {
	inputFileName := "test-util-" + randLowStr(6)
	configFileName := "test-util-config" + randLowStr(6)
	cf, _ := os.Create(configFileName)
	cf.Close()

	// prepare input config
	inputFile, _ := os.OpenFile(inputFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, 0664)
	strEndPoint := randLowStr(1024) // over 256
	io.WriteString(inputFile, strEndPoint)
	inputFile.Seek(0, 0)

	oldStdin := os.Stdin
	os.Stdin = inputFile

	err := configCommand.configInteractive(configFileName, LEnglishLanguage)
	c.Assert(err, IsNil)

	fileData, err := ioutil.ReadFile(configFileName)
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(string(fileData), strEndPoint), Equals, true)

	os.Stdin = oldStdin
	inputFile.Close()

	os.Remove(inputFileName)
	os.Remove(configFileName)
	os.Remove(configFileName + ".bak")
}

func (s *OssutilCommandSuite) TestConfigHelpInfo(c *C) {
	options := OptionMapType{}

	mkArgs := []string{"config"}
	_, err := cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

	mkArgs = []string{}
	_, err = cm.RunCommand("help", mkArgs, options)
	c.Assert(err, IsNil)

}
