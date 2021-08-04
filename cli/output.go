// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"fmt"
	"io"
	"os"
)

var (
	defaultOutput       = NewOutput()
	defaultWriter       = os.Stdout
	defaultStderrWriter = os.Stderr
)

func DefaultWriter() io.Writer {
	return defaultWriter
}

func DefaultStderrWriter() io.Writer {
	return defaultStderrWriter
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
