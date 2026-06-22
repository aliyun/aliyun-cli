package logutil

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogThrottlingRetryHiddenByDefault(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(Error)
	defer SetOutput(nil)

	LogThrottlingRetry("throttling, retrying in %dms (attempt %d/%d)", 15, 1, 3)
	assert.Empty(t, buf.String())
}

func TestLogThrottlingRetryVisibleAtInfo(t *testing.T) {
	var buf bytes.Buffer
	SetOutput(&buf)
	SetLevel(Info)
	defer func() {
		SetOutput(nil)
		SetLevel(Error)
	}()

	LogThrottlingRetry("throttling, retrying in %dms (attempt %d/%d)", 15, 1, 3)
	assert.Equal(t, "aliyun: throttling, retrying in 15ms (attempt 1/3)\n", buf.String())
}

func TestInitFromContextParsesInfo(t *testing.T) {
	SetLevel(Error)
	InitFromContext("INFO")
	assert.True(t, IsInfoEnabled())
	SetLevel(Error)
}

func TestInitFromContextNamedConfig(t *testing.T) {
	SetLevel(Error)
	InitFromContext("development")
	assert.True(t, IsInfoEnabled())
	SetLevel(Error)
}
