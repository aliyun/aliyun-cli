package lib

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseArgOptions(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	os.Args = []string{"oss", "ls", "--region", "cn-hangzhou"}
	args, maps, err := ParseArgOptions()
	assert.Equal(t, []string{"ls"}, args)
	assert.Nil(t, err)
	assert.NotNil(t, maps)
	region, ok := maps["region"].(*string)
	assert.True(t, ok)
	assert.Equal(t, "cn-hangzhou", *region)
}
