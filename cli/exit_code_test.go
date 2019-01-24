/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEnableExitCode(t *testing.T) {
	assert.True(t, withExitCode)
	DisableExitCode()
	assert.False(t, withExitCode)
	EnableExitCode()
	assert.True(t, withExitCode)
}
