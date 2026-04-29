package mock

import (
	"fmt"
	"io"
	"os"
)

type Options struct {
	Args     []string
	Stdout   io.Writer
	Stderr   io.Writer
	MockPath string
}

type Result struct {
	Handled  bool
	ExitCode int
}

var (
	loadRecords = Load
	saveRecords = Save
)

func Intercept(opts Options) Result {
	if os.Getenv(EnvMockEnabled) != "true" {
		return Result{}
	}

	records, err := loadRecords(opts.MockPath)
	if err != nil {
		writef(opts.Stderr, "ERROR: load mock data failed %s\n", err)
		return Result{Handled: true, ExitCode: 1}
	}

	index, ok := FindMatch(records, opts.Args)
	if !ok {
		return Result{}
	}

	record := records[index]
	records = Consume(records, index)
	if err := saveRecords(opts.MockPath, records); err != nil {
		writef(opts.Stderr, "ERROR: save mock data failed %s\n", err)
		return Result{Handled: true, ExitCode: 1}
	}

	if record.Stdout != "" {
		writes(opts.Stdout, record.Stdout)
	}
	if record.Stderr != "" {
		writes(opts.Stderr, record.Stderr)
	}

	return Result{Handled: true, ExitCode: record.ExitCode}
}

func writes(writer io.Writer, text string) {
	if writer == nil {
		return
	}
	_, _ = io.WriteString(writer, text)
}

func writef(writer io.Writer, format string, args ...interface{}) {
	if writer == nil {
		return
	}
	_, _ = fmt.Fprintf(writer, format, args...)
}
