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
	os.Remove(configFile)
}

// Run once after all tests or benchmarks have finished running
func (s *OssutilConfigSuite) TearDownTest(c *C) {
	fmt.Printf("tear down test:%s\n", c.TestName())
	os.Remove(configFile)
}

// test "config"
func (s *OssutilConfigSuite) TestConfigNonInteractive(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
	endpoint := "oss-cn-hangzhou.aliyuncs.com"
	accessKeyID := "ak"
	accessKeySecret := "sk"
	stsToken := "token"
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
	os.Remove(configFile)
}

func (s *OssutilConfigSuite) TestConfigNonInteractiveWithAgent(c *C) {
	command := "config"
	var args []string
	userAgent := "demo-walker"
	stsToken := "token"
	options := OptionMapType{
		"endpoint":        &endpoint,
		"accessKeyID":     &accessKeyID,
		"accessKeySecret": &accessKeySecret,
		"userAgent":       &userAgent,
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
	c.Assert(opts[OptionUserAgent], Equals, nil)
	c.Assert(opts[OptionSTSToken], Equals, stsToken)
}

func (s *OssutilConfigSuite) TestConfigNonInteractiveLanguage(c *C) {
	command := "config"
	var args []string
	for _, language := range []string{DefaultLanguage, EnglishLanguage, LEnglishLanguage} {
		configFile := randStr(10)
		endpoint := "oss-cn-hangzhou.aliyuncs.com"
		stsToken := "token"
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
		os.Remove(configFile)
	}
}

func (s *OssutilConfigSuite) TestConfigInteractive(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
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
	os.Remove(configFile)
}

func (s *OssutilConfigSuite) TestConfigInteractiveLanguage(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
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
	os.Remove(configFile)
}

func (s *OssutilConfigSuite) TestConfigLanguageEN(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
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
	os.Remove(configFile)
}

func (s *OssutilConfigSuite) TestConfigLanguageCH(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
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
	os.Remove(configFile)
}

// test option empty value
func (s *OssutilConfigSuite) TestConfigOptionEmptyValue(c *C) {
	command := "config"
	var args []string
	configFile := randStr(10)
	endp := ""
	id := ""
	accessKeySecret := "sk"
	stsToken := "token"
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
	os.Remove(configFile)
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

func (s *OssutilConfigSuite) TestConfigNonInteractiveWithCommonOption(c *C) {
	cfile := randStr(10)
	userAgent := "demo-walker"
	stsToken := "stsToken"
	logLevel := "info"
	proxyHost = "http://192.168.0.1:8085"
	proxyUser = "test"
	proxyPwd = "test1234"
	mode := "AK"
	ecsRoleName = "ossTest"
	tokenTimeout := "300"
	ramRoleArn := "acs:ram::123*******123:role/ramosssts"
	roleSessionName := "roleTest"
	readTimeout := "10"
	connectTimeOut := "10"
	stsRegion = "sts.cn-qingdao.aliyuncs.com"
	signVersion := "v4"
	region := "oss-cn-chengdu.aliyuncs.com"
	cloudboxId := "12124123"
	retryTimes := "400"
	data := "[Credentials]" + "\n" +
		"language=" + DefaultLanguage + "\n" +
		"accessKeyID=" + accessKeyID + "\n" +
		"accessKeySecret=" + accessKeySecret + "\n" +
		"endpoint=" + endpoint + "\n" +
		"stsToken=" + stsToken + "\n" +
		"[Default]" + "\n" +
		"loglevel=" + logLevel + "\n" +
		"userAgent=" + userAgent + "\n" +
		"proxyHost=" + proxyHost + "\n" +
		"proxyUser=" + proxyUser + "\n" +
		"proxyPwd=" + proxyPwd + "\n" +
		"readTimeOut=" + readTimeout + "\n" +
		"connectTimeOut=" + connectTimeOut + "\n" +
		"retryTimes=" + retryTimes + "\n" +
		"mode=" + mode + "\n" +
		"ecsRoleName=" + ecsRoleName
	s.createFile(cfile, data, c)

	f, err := os.Stat(cfile)
	c.Assert(err, IsNil)
	c.Assert(f.Size() > 0, Equals, true)

	opts, err := LoadConfig(cfile)
	testLogger.Print(opts)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 13)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
	c.Assert(opts[OptionEndpoint], Equals, endpoint)
	c.Assert(opts[OptionAccessKeyID], Equals, accessKeyID)
	c.Assert(opts[OptionAccessKeySecret], Equals, accessKeySecret)

	c.Assert(opts[OptionSTSToken], Equals, stsToken)

	c.Assert(opts[OptionLogLevel], Equals, logLevel)
	c.Assert(opts[OptionUserAgent], Equals, userAgent)
	c.Assert(opts[OptionProxyHost], Equals, proxyHost)
	c.Assert(opts[OptionProxyUser], Equals, proxyUser)
	c.Assert(opts[OptionProxyPwd], Equals, proxyPwd)
	c.Assert(opts[OptionReadTimeout], Equals, readTimeout)
	c.Assert(opts[OptionConnectTimeout], Equals, connectTimeOut)

	c.Assert(opts[OptionMode], Equals, nil)
	c.Assert(opts[OptionRamRoleArn], Equals, nil)
	c.Assert(opts[OptionTokenTimeout], Equals, nil)
	c.Assert(opts[OptionRoleSessionName], Equals, nil)
	c.Assert(opts[OptionSTSRegion], Equals, nil)
	c.Assert(opts[OptionECSRoleName], Equals, nil)
	c.Assert(opts[OptionSignVersion], Equals, nil)
	c.Assert(opts[OptionRegion], Equals, nil)
	c.Assert(opts[OptionCloudBoxID], Equals, nil)

	os.Remove(cfile)

	tokenTimeout1 := "301"
	ramRoleArn1 := "acs:ram::123*******123:role/ramosssts1"
	roleSessionName1 := "roleTest1"
	ecsRoleName1 := "ossTest1"
	stsRegion1 := "sts.cn-hangzhou.aliyuncs.com"
	data = "[Credentials]" + "\n" +
		"language=" + DefaultLanguage + "\n" +
		"accessKeyID=" + accessKeyID + "\n" +
		"accessKeySecret=" + accessKeySecret + "\n" +
		"endpoint=" + endpoint + "\n" +
		"stsToken=" + stsToken + "\n" +
		"ecsRoleName=" + ecsRoleName1 + "\n" +
		"tokenTimeout=" + tokenTimeout1 + "\n" +
		"ramRoleArn=" + ramRoleArn1 + "\n" +
		"roleSessionName=" + roleSessionName1 + "\n" +
		"mode=" + mode + "\n" +
		"stsRegion=" + stsRegion1 + "\n" +
		"signVersion=" + signVersion + "\n" +
		"region=" + region + "\n" +
		"cloudBoxID=" + cloudboxId + "\n" +
		"[Default]" + "\n" +
		"loglevel=" + logLevel + "\n" +
		"userAgent=" + userAgent + "\n" +
		"proxyHost=" + proxyHost + "\n" +
		"proxyUser=" + proxyUser + "\n" +
		"proxyPwd=" + proxyPwd + "\n" +
		"ecsRoleName=" + ecsRoleName + "\n" +
		"tokenTimeout=" + tokenTimeout + "\n" +
		"ramRoleArn=" + ramRoleArn + "\n" +
		"roleSessionName=" + roleSessionName + "\n" +
		"readTimeOut=" + readTimeout + "\n" +
		"connectTimeOut=" + connectTimeOut
	s.createFile(cfile, data, c)

	opts, err = LoadConfig(cfile)
	testLogger.Print(opts)
	c.Assert(err, IsNil)
	c.Assert(len(opts), Equals, 21)
	c.Assert(opts[OptionLanguage], Equals, DefaultLanguage)
	c.Assert(opts[OptionEndpoint], Equals, endpoint)
	c.Assert(opts[OptionAccessKeyID], Equals, accessKeyID)
	c.Assert(opts[OptionAccessKeySecret], Equals, accessKeySecret)

	c.Assert(opts[OptionSTSToken], Equals, stsToken)

	c.Assert(opts[OptionUserAgent], Equals, userAgent)
	c.Assert(opts[OptionLogLevel], Equals, logLevel)
	c.Assert(opts[OptionProxyHost], Equals, proxyHost)
	c.Assert(opts[OptionProxyUser], Equals, proxyUser)
	c.Assert(opts[OptionProxyPwd], Equals, proxyPwd)
	c.Assert(opts[OptionReadTimeout], Equals, readTimeout)
	c.Assert(opts[OptionConnectTimeout], Equals, connectTimeOut)

	c.Assert(opts[OptionMode], Equals, mode)
	c.Assert(opts[OptionRamRoleArn], Equals, ramRoleArn1)
	c.Assert(opts[OptionTokenTimeout], Equals, tokenTimeout1)
	c.Assert(opts[OptionRoleSessionName], Equals, roleSessionName1)
	c.Assert(opts[OptionSTSRegion], Equals, stsRegion1)
	c.Assert(opts[OptionECSRoleName], Equals, ecsRoleName1)
	c.Assert(opts[OptionSignVersion], Equals, signVersion)
	c.Assert(opts[OptionRegion], Equals, region)
	c.Assert(opts[OptionCloudBoxID], Equals, cloudboxId)

	os.Remove(cfile)
}

func (s *OssutilConfigSuite) createFile(fileName, content string, c *C) {
	fout, err := os.Create(fileName)
	defer fout.Close()
	c.Assert(err, IsNil)
	_, err = fout.WriteString(content)
	c.Assert(err, IsNil)
}
