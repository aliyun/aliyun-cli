/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package cli

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPlatformCompatible(t *testing.T) {
	PlatformCompatible()
	if runtime.GOOS == "windows" {
		assert.False(t, withColor)
	} else {
		assert.True(t, withColor)
	}
	EnableColor()
}
