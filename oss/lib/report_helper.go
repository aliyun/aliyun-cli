package lib

import (
	"fmt"
	"log"
	"os"
	"time"
)

type Reporter struct {
	rlogger    *log.Logger
	written    bool
	prompted   bool
	path       string
	comment    string
	outputDir  string
	createDir  bool
	fileHandle *os.File
}

func (re *Reporter) Init(outputDir, comment string) error {
	if outputDir == "" {
		outputDir = DefaultOutputDir
	}
	re.outputDir = outputDir
	re.createDir = false
	if _, err := os.Stat(outputDir); err != nil && os.IsNotExist(err) {
		re.createDir = true
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return err
	}
	re.path = re.outputDir + string(os.PathSeparator) + ReportPrefix + time.Now().Format("20060102_150405") + ReportSuffix
	re.comment = comment
	re.written = false
	re.prompted = false
	f, err := os.OpenFile(re.path, os.O_RDWR|os.O_APPEND|os.O_CREATE, 0664)
	if err != nil {
		return fmt.Errorf("Create reporter file error: %s", err.Error())
	}
	re.fileHandle = f
	re.rlogger = log.New(f, "", log.Ldate|log.Ltime)
	re.Comment()
	re.rlogger.SetFlags(log.Ldate | log.Ltime)
	return nil
}

func (re *Reporter) Clear() {
	if re != nil && re.fileHandle != nil {
		re.fileHandle.Close()
	}

	if re != nil && !re.written {
		os.Remove(re.path)
		if re.createDir {
			os.RemoveAll(re.outputDir)
		}
	}
}

func (re *Reporter) HasPrompt() bool {
	if re == nil {
		return false
	}
	return re.prompted == false
}

func (re *Reporter) Comment() {
	if re != nil && !re.written {
		re.rlogger.SetFlags(0)
		re.rlogger.SetPrefix("# ")
		re.rlogger.Println(re.comment)
	}
}

func (re *Reporter) ReportError(msg string) {
	if re != nil && re.rlogger != nil {
		re.written = true
		re.rlogger.SetPrefix("[Error] ")
		re.rlogger.Println(msg)
	}
}

func (re *Reporter) Prompt(err error) {
	if re != nil && re.written && re.HasPrompt() {
		re.prompted = true
		fmt.Printf("\r%s\rError occurs, message: %s. See more information in file: %s\n", clearStr, err.Error(), re.path)
	}
}

func GetReporter(need bool, outputDir, comment string) (*Reporter, error) {
	if need {
		var reporter Reporter
		if err := reporter.Init(outputDir, comment); err != nil {
			return nil, err
		}
		return &reporter, nil
	}
	return nil, nil
}
