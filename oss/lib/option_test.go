package lib

import (
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestParseArgOptionsCommandDiagnostics(t *testing.T) {
	originalArgs := os.Args
	t.Cleanup(func() { os.Args = originalArgs })
	for _, tc := range []struct {
		args    []string
		command string
	}{
		{[]string{"oss", "ls", "--header=value"}, " ls"},
		{[]string{"oss", "cp", "--header=value"}, " cp"},
		{[]string{"oss", "unknown-command", "--header=value"}, ""},
		{[]string{"oss", "--header=value"}, ""},
		{[]string{"oss", "untrusted; command", "--header=value"}, ""},
		{[]string{"oss", "--region", "ls", "--header=value"}, ""},
	} {
		t.Run(strings.Join(tc.args[1:], " "), func(t *testing.T) {
			os.Args = append([]string(nil), tc.args...)
			_, _, err := ParseArgOptions()
			require.EqualError(t, err, "--header is not an oss command option.\nRun 'aliyun oss"+tc.command+" --help' for supported oss options.")
			require.Equal(t, tc.args, os.Args)
		})
	}
}
