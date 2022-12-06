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
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
)

func lineInFile(name string, lookFor string) bool {
	f, err := os.Open(name)
	if err != nil {
		return false
	}
	defer f.Close()
	r := bufio.NewReader(f)
	prefix := []byte{}
	for {
		line, isPrefix, err := r.ReadLine()
		if err == io.EOF {
			return false
		}
		if err != nil {
			return false
		}
		if isPrefix {
			prefix = append(prefix, line...)
			continue
		}
		line = append(prefix, line...)
		if string(line) == lookFor {
			return true
		}
		prefix = prefix[:0]
	}
}

func createFile(name string, content string) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("%s\n", content))
	return err
}

func appendToFile(name string, content string) error {
	f, err := os.OpenFile(name, os.O_RDWR|os.O_APPEND, 0)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.WriteString(fmt.Sprintf("\n%s\n", content))
	return err
}

func removeFromFile(name string, content string) error {
	backup := name + ".bck"
	err := copyFile(name, backup)
	if err != nil {
		return err
	}
	temp, err := removeContentToTempFile(name, content)
	if err != nil {
		return err
	}

	err = copyFile(temp, name)
	if err != nil {
		return err
	}

	return os.Remove(backup)
}

func removeContentToTempFile(name, content string) (string, error) {

	rf, err := os.Open(name)
	if err != nil {
		return "", err
	}
	defer rf.Close()
	wf, err := ioutil.TempFile("", "complete-")
	if err != nil {
		return "", err
	}
	defer wf.Close()

	r := bufio.NewReader(rf)
	prefix := []byte{}
	for {
		line, isPrefix, err := r.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if isPrefix {
			prefix = append(prefix, line...)
			continue
		}
		line = append(prefix, line...)
		str := string(line)
		if str == content {
			continue
		}
		wf.WriteString(str + "\n")
		prefix = prefix[:0]
	}
	return wf.Name(), nil
}

func copyFile(src string, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func getBinaryPath() (string, error) {
	bin, err := os.Executable()
	if err != nil {
		return "", err
	}
	return filepath.Abs(bin)
}

func rcFile(name string) string {
	path := filepath.Join(getHomeDir(), name)
	fmt.Println(path)
	if _, err := os.Stat(path); err != nil {
		return ""
	}

	return path
}
