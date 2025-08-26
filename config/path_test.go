package config

import (
	"os"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHomePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		assert.Equal(t, os.Getenv("USERPROFILE"), GetHomePath())
	} else {
		assert.Equal(t, os.Getenv("HOME"), GetHomePath())
	}
}

func TestGetXDGConfigHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		return
	}

	assert.Equal(t, os.Getenv("HOME")+"/.config", GetXDGConfigHome())
	os.Setenv("XDG_CONFIG_HOME", "/tmp/config")
	assert.Equal(t, "/tmp/config", GetXDGConfigHome())
	os.Setenv("XDG_CONFIG_HOME", "")
	assert.Equal(t, os.Getenv("HOME")+"/.config", GetXDGConfigHome())
}
