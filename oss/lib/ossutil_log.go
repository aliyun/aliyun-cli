package lib

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"path/filepath"

	oss "github.com/aliyun/aliyun-oss-go-sdk/oss"
)

var logName = "ossutil.log"
var logLevel = oss.LogOff
var utilLogger *log.Logger
var logFile *os.File

func openLogFile() (*os.File, error) {
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		dir = "."
	}
	absLogName := dir + string(os.PathSeparator) + logName
	f, err := os.OpenFile(absLogName, os.O_CREATE|os.O_APPEND|os.O_RDWR, 0660)
	if err != nil {
		fmt.Printf("open %s error,info:%s.\n", absLogName, err.Error())
	} else {
		fmt.Printf("log file is %s\n", absLogName)
	}
	return f, err
}

func InitLogger(level int, name string) {
	logLevel = level
	logName = name
	f, err := openLogFile()
	if err != nil {
		return
	}
	utilLogger = log.New(f, "", log.LstdFlags|log.Lmicroseconds)
	logFile = f
}

func UnInitLogger() {
	if logFile == nil {
		return
	}

	logFile.Close()
	logFile = nil
	utilLogger = nil
}

func writeLog(level int, format string, a ...interface{}) {
	if utilLogger == nil {
		return
	}

	var logBuffer bytes.Buffer
	logBuffer.WriteString(oss.LogTag[level-1])
	logBuffer.WriteString(fmt.Sprintf(format, a...))
	utilLogger.Printf("%s", logBuffer.String())
	return
}

func LogError(format string, a ...interface{}) {
	if logLevel < oss.Error {
		return
	}
	writeLog(oss.Error, format, a...)
}

func LogWarn(format string, a ...interface{}) {
	if logLevel < oss.Warn {
		return
	}
	writeLog(oss.Warn, format, a...)
}

func LogInfo(format string, a ...interface{}) {

	if logLevel < oss.Info {
		return
	}
	writeLog(oss.Info, format, a...)
}

func LogDebug(format string, a ...interface{}) {
	if logLevel < oss.Debug {
		return
	}
	writeLog(oss.Debug, format, a...)
}
