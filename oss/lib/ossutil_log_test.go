package lib

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
	. "gopkg.in/check.v1"
)

type OssUtilLogSuite struct {
	testLogName  string
	testLogLevel int
}

var _ = Suite(&OssUtilLogSuite{})

// Run once when the suite starts running
func (s *OssUtilLogSuite) SetUpSuite(c *C) {
	fmt.Printf("set up suite OssUtilLogSuite\n")
}

// Run before each test or benchmark starts running
func (s *OssUtilLogSuite) TearDownSuite(c *C) {
	fmt.Printf("tear down OssUtilLogSuite\n")
}

// Run after each test or benchmark runs
func (s *OssUtilLogSuite) SetUpTest(c *C) {
	fmt.Printf("set up test:%s\n", c.TestName())
	s.testLogName = logName
	s.testLogLevel = logLevel

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = ""
	}
	absLogName := dir + string(os.PathSeparator) + logName
	os.Remove(absLogName)
}

// Run once after all tests or benchmarks have finished running
func (s *OssUtilLogSuite) TearDownTest(c *C) {
	fmt.Printf("tear down test:%s\n", c.TestName())
	logName = s.testLogName
	logLevel = s.testLogLevel

	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = ""
	}
	absLogName := dir + string(os.PathSeparator) + logName
	os.Remove(absLogName)
}

// test "config"
func (s *OssUtilLogSuite) TestLogLevel(c *C) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = ""
	}
	absLogName := dir + string(os.PathSeparator) + logName

	fmt.Printf("absLogName:%s.\n", absLogName)

	// nologLevel
	logLevel = oss.LogOff
	InitLogger(logLevel, logName)

	errorContext := "i am error log.\n"
	LogError(errorContext)
	LogWarn(errorContext)
	LogInfo(errorContext)
	LogDebug(errorContext)

	contents, err := ioutil.ReadFile(absLogName)
	LogContent := string(contents)
	c.Assert(strings.Contains(LogContent, "[error]"+errorContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[warn]"+errorContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[info]"+errorContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[debug]"+errorContext), Equals, false)
	UnInitLogger()
	os.Remove(absLogName)

	// errorLevel
	logLevel = oss.Error
	InitLogger(logLevel, logName)
	LogError(errorContext)
	LogWarn(errorContext)
	LogInfo(errorContext)
	LogDebug(errorContext)

	contents, err = ioutil.ReadFile(absLogName)
	LogContent = string(contents)
	c.Assert(strings.Contains(LogContent, "[error]"+errorContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[warn]"+errorContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[info]"+errorContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[debug]"+errorContext), Equals, false)
	UnInitLogger()
	os.Remove(absLogName)

	// normalLevel
	logLevel = oss.Warn
	InitLogger(logLevel, logName)
	normalContext := "i am normal log.\n"
	LogError(normalContext)
	LogWarn(normalContext)
	LogInfo(normalContext)
	LogDebug(normalContext)

	contents, err = ioutil.ReadFile(absLogName)
	LogContent = string(contents)
	c.Assert(strings.Contains(LogContent, "[error]"+normalContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[warn]"+normalContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[info]"+normalContext), Equals, false)
	c.Assert(strings.Contains(LogContent, "[debug]"+normalContext), Equals, false)
	UnInitLogger()
	os.Remove(absLogName)

	// infolevel
	logLevel = oss.Info
	InitLogger(logLevel, logName)
	infoContext := "i am info log.\n"
	LogError(infoContext)
	LogWarn(infoContext)
	LogInfo(infoContext)
	LogDebug(infoContext)

	contents, err = ioutil.ReadFile(absLogName)
	LogContent = string(contents)
	c.Assert(strings.Contains(LogContent, "[error]"+infoContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[warn]"+infoContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[info]"+infoContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[debug]"+infoContext), Equals, false)
	UnInitLogger()
	os.Remove(absLogName)

	// debuglevel
	logLevel = oss.Debug
	InitLogger(logLevel, logName)
	debugContext := "i am debug log.\n"
	LogError(debugContext)
	LogWarn(debugContext)
	LogInfo(debugContext)
	LogDebug(debugContext)

	contents, err = ioutil.ReadFile(absLogName)
	LogContent = string(contents)
	c.Assert(strings.Contains(LogContent, "[error]"+debugContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[warn]"+debugContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[info]"+debugContext), Equals, true)
	c.Assert(strings.Contains(LogContent, "[debug]"+debugContext), Equals, true)
	UnInitLogger()
	os.Remove(absLogName)
}

func (s *OssUtilLogSuite) TestOpenFileError(c *C) {
	oldLogName := logName
	logName = ""
	_, err := openLogFile()
	c.Assert(err, NotNil)
	logName = oldLogName
}

func (s *OssUtilLogSuite) TestInitLoggerError(c *C) {
	oldLogName := logName
	oldLogFile := logFile
	InitLogger(oss.Info, "")
	c.Assert(logFile, IsNil)

	logName = oldLogName
	logFile = oldLogFile
}
