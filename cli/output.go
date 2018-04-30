package cli

import (
	"fmt"
	"io"
	"os"
)

var (
	defaultOutput = NewOutput()
	defaultWriter = os.Stdout
)

func DefaultWriter() io.Writer {
	return defaultWriter
}

func Print(w io.Writer, a ...interface{}) (n int, err error) {
	return defaultOutput.Print(w, a...)
}

func Println(w io.Writer, a ...interface{}) (n int, err error) {
	return defaultOutput.Println(w, a...)
}

func Printf(w io.Writer, format string, args ...interface{}) (n int, err error) {
	return defaultOutput.Printf(w, format, args...)
}

func NewOutput() *Output {
	return &Output{}
}

type Output struct {
}

func (o *Output) Print(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprint(w, a...)
}

func (o *Output) Println(w io.Writer, a ...interface{}) (n int, err error) {
	return fmt.Fprintln(w, a...)
}

func (o *Output) Printf(w io.Writer, format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(w, format, a...)
}
