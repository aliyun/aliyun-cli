package openapi

import (
	"bytes"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/logutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInitThrottlingLogFromFlag(t *testing.T) {
	t.Cleanup(func() {
		logutil.SetLevel(logutil.Error)
	})

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	AddFlags(ctx.Flags())
	flag := LogLevelFlag(ctx.Flags())
	require.NotNil(t, flag)
	flag.SetAssigned(true)
	flag.SetValue("INFO")

	initThrottlingLog(ctx)
	assert.True(t, logutil.IsInfoEnabled())
}

func TestInitThrottlingLogWithoutFlagValue(t *testing.T) {
	t.Cleanup(func() {
		logutil.SetLevel(logutil.Error)
	})

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	AddFlags(ctx.Flags())

	initThrottlingLog(ctx)
	assert.False(t, logutil.IsInfoEnabled())
}

func TestInitThrottlingLogNilFlags(t *testing.T) {
	t.Cleanup(func() {
		logutil.SetLevel(logutil.Error)
	})

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	initThrottlingLog(ctx)
	assert.False(t, logutil.IsInfoEnabled())
}
