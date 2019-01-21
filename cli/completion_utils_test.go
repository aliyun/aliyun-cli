/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
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
