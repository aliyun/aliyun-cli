package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetFromEnv(t *testing.T) {
	os.Setenv("test1", "test1")
	os.Setenv("test2", "test2")
	assert.Equal(t, "test1", GetFromEnv("test1", "test2"))
	assert.Equal(t, "test1", GetFromEnv("test3", "test1", "test2"))
	assert.Equal(t, "", GetFromEnv("test3"))
}
