// Copyright (c) 2009-present, Alibaba Cloud All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package cli

import (
	"bufio"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func readfile(t *testing.T, name string) string {
	file, err := os.Open(name)
	assert.Nil(t, err)

	reader := bufio.NewReader(file)
	var str string
	buf := make([]byte, 1024)
	for {
		n, err := reader.Read(buf)
		if n < 1024 || err != nil {
			str += string(buf[:n])
			break
		}
		str += string(buf)
	}
	file.Close()
	return str
}

func TestCreateFile(t *testing.T) {
	err := createFile("test.txt", "oss")
	assert.Nil(t, err)
	err1 := createFile(" /?&^%$#*/***/./.", "OOO")
	assert.NotNil(t, err1)
	str := readfile(t, "test.txt")
	assert.Equal(t, "oss\n", str)
	assert.Nil(t, os.Remove("test.txt"))
}

func TestOpToFile(t *testing.T) {
	err := createFile("test.txt", "oss")
	assert.Nil(t, err)

	//appendToFile
	err = appendToFile("test.txt", "cdn")
	assert.Nil(t, err)
	str := readfile(t, "test.txt")
	assert.Equal(t, "oss\n\ncdn\n", str)

	//removeFromFile
	err = removeFromFile("test.txt", "cdn")
	assert.Nil(t, err)
	str = readfile(t, "test.txt")
	assert.Equal(t, "oss\n\n", str)

	assert.Nil(t, os.Remove("test.txt"))
}

func TestRemoveContentToTempFile(t *testing.T) {
	err := createFile("test.txt", "oss")
	assert.Nil(t, err)
	_, err = removeContentToTempFile("test.txt", "oss")
	assert.Nil(t, err)
	assert.Nil(t, os.Remove("test.txt"))
}

func TestCopyFile(t *testing.T) {
	err := createFile("test.txt", "oss")
	assert.Nil(t, err)
	err = copyFile("test.txt", "testcp.txt")
	assert.Nil(t, err)

	defer func() {
		if _, err = os.Stat("test.txt"); err == nil {
			os.Remove("test.txt")
		}
		if _, err = os.Stat("testcp.txt"); err == nil {
			os.Remove("testcp.txt")
		}
	}()

	test, err := os.Open("test.txt")
	assert.Nil(t, err)
	var teststr string
	test.WriteString(teststr)
	test.Close()
	testcp, err := os.Open("testcp.txt")
	assert.Nil(t, err)
	var testcpstr string
	testcp.WriteString(testcpstr)
	testcp.Close()
	assert.Equal(t, teststr, testcpstr)

}

func TestGetBinaryPath(t *testing.T) {
	bpath, err := getBinaryPath()
	assert.Nil(t, err)
	assert.NotNil(t, bpath)
}

func TestRCFile(t *testing.T) {
	path := rcFile("aa")
	assert.Empty(t, path)
	name := getHomeDir() + "/hh"
	file, err := os.Create(name)
	defer func() {
		err := file.Close()
		assert.Nil(t, err)
	}()
	assert.Nil(t, err)
	path = rcFile("hh")
	assert.NotNil(t, path)
	os.Remove(name)
}
