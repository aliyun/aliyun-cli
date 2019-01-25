/*
 * Copyright (C) 1999-2019 Alibaba Group Holding Limited
 */
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserAgent(t *testing.T) {
	SetUserAgent("aliyun-cli-v4.0.0")
	assert.Equal(t, "aliyun-cli-v4.0.0", userAgent)
	assert.Equal(t, "aliyun-cli-v4.0.0", GetUserAgent())
}
