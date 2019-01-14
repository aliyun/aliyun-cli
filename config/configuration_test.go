package config

import (
	"bufio"
	"github.com/stretchr/testify/assert"
	"os"

	"testing"
)

func TestGetConfigPath(t *testing.T) {
	homepath := GetHomePath()
	expath := homepath + "/.aliyun"
	path := GetConfigPath()
	if ok := assert.Equal(t, expath, path); ok {
		file1, err := os.Open(path)
		ok = assert.NoError(t, err)
		if ok {
			file1.Close()
		}

	}

}

func TestLoadProfile(t *testing.T) {
	w := new(bufio.Writer)
	assert.Nil(t, MigrateLegacyConfiguration(w))

	//test target reset to legacy_test.go
}
