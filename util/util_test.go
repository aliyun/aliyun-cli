package util

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFromEnv(t *testing.T) {
	os.Setenv("test1", "test1")
	os.Setenv("test2", "test2")
	assert.Equal(t, "test1", GetFromEnv("test1", "test2"))
	assert.Equal(t, "test1", GetFromEnv("test3", "test1", "test2"))
	assert.Equal(t, "", GetFromEnv("test3"))
}

func TestGetCurrentUnixTime(t *testing.T) {
	// 获取函数返回的时间戳
	timestamp := GetCurrentUnixTime()

	// 验证时间戳不为0
	assert.NotEqual(t, int64(0), timestamp)

	// 验证时间戳是近期的时间（在过去一分钟内）
	now := time.Now().Unix()
	assert.True(t, timestamp <= now)
	assert.True(t, timestamp >= now-60, "时间戳应该在当前时间的一分钟内")
}

func TestNewHttpClient(t *testing.T) {
	// 获取HTTP客户端
	client := NewHttpClient()

	// 验证客户端不为nil
	assert.NotNil(t, client)

	// 验证超时设置为10秒
	assert.Equal(t, time.Second*10, client.Timeout)
}
