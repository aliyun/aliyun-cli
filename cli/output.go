package cli

import (
	"io"
	"fmt"
	"os"
)

var DefaultOutput = NewOutput(os.Stdout)

func Print(a ...interface{}) (n int, err error){
	return DefaultOutput.Print(a...)
}

func Println(a ...interface{}) (n int, err error){
	return DefaultOutput.Println(a...)
}

func Printf(format string, args ...interface{}) (n int, err error){
	return DefaultOutput.Printf(format, args...)
}

func GetOutputWriter() io.Writer {
	return DefaultOutput.GetWriter()
}

func NewOutput(writer io.Writer) *Output{
	return &Output{writer: writer}
}

type Output struct {
	writer io.Writer
}

func (o *Output)GetWriter() io.Writer {
	return o.writer
}

func (o *Output)Print(a ...interface{}) (n int, err error) {
	return fmt.Fprint(o.writer, a...)
}

func (o *Output)Println(a ...interface{}) (n int, err error) {
	return fmt.Fprintln(o.writer, a...)
}

func (o *Output)Printf(format string, a ...interface{}) (n int, err error) {
	return fmt.Fprintf(o.writer, format , a...)
}