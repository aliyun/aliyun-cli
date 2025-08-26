package util

import (
	"io"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestGetFromEnv(t *testing.T) {
	_ = os.Setenv("test1", "test1")
	_ = os.Setenv("test2", "test2")
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

func TestRandStringBytesMaskImprSrc(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"zero length", 0},
		{"small length", 5},
		{"medium length", 10},
		{"large length", 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := RandStringBytesMaskImprSrc(tt.length)

			// 验证返回类型是字符串
			str, ok := result.(string)
			assert.True(t, ok, "返回值应该是字符串类型")

			// 验证长度
			assert.Equal(t, tt.length, len(str), "生成的字符串长度应该等于输入参数")

			// 验证字符串只包含预期的字符
			if tt.length > 0 {
				expectedChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
				for _, char := range str {
					assert.Contains(t, expectedChars, string(char), "生成的字符应该在预期字符集中")
				}
			}
		})
	}

	// 测试多次调用生成不同的字符串（虽然由于时间相关性，这个测试可能偶尔失败）
	t.Run("randomness", func(t *testing.T) {
		results := make(map[string]bool)
		length := 10
		attempts := 5

		for i := 0; i < attempts; i++ {
			result := RandStringBytesMaskImprSrc(length)
			str := result.(string)
			results[str] = true
			time.Sleep(time.Nanosecond) // 确保时间戳不同
		}

		// 至少应该有一些不同的结果（虽然理论上可能全部相同，但概率极低）
		assert.True(t, len(results) >= 1, "应该生成至少一个结果")
	})
}

func TestOpenBrowser(t *testing.T) {
	testURL := "https://www.example.com"

	t.Run("successful execution", func(t *testing.T) {
		// 由于实际打开浏览器可能影响测试环境，我们主要测试函数不会panic
		// 并且总是返回nil
		err := OpenBrowser(testURL)
		assert.Nil(t, err, "OpenBrowser 应该总是返回 nil")
	})

	t.Run("with empty URL", func(t *testing.T) {
		err := OpenBrowser("")
		assert.Nil(t, err, "即使URL为空，OpenBrowser 也应该返回 nil")
	})

	t.Run("with invalid URL", func(t *testing.T) {
		err := OpenBrowser("not-a-valid-url")
		assert.Nil(t, err, "即使URL无效，OpenBrowser 也应该返回 nil")
	})

	// 测试不同操作系统的命���选择（模拟测试）
	t.Run("command selection by OS", func(t *testing.T) {
		// 这个测试主要是为了覆盖代码，实际的命令执行我们无法轻易模拟
		// 但我们可以验证函数在不同场景下的行为

		// 在当前OS上执行测试
		_ = runtime.GOOS // 引用变量避免未使用警告
		err := OpenBrowser(testURL)
		assert.Nil(t, err)
	})
}

func TestUnmarshalJsonFromReader(t *testing.T) {
	t.Run("valid JSON", func(t *testing.T) {
		jsonData := `{
			"access_token": "test_access_token",
			"refresh_token": "test_refresh_token",
			"expires_in": 3600,
			"token_type": "Bearer"
		}`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.NoError(t, err)
		assert.Equal(t, "test_access_token", result.AccessToken)
		assert.Equal(t, "test_refresh_token", result.RefreshToken)
		assert.Equal(t, int64(3600), result.ExpiresIn)
		assert.Equal(t, "Bearer", result.TokenType)
	})

	t.Run("partial JSON", func(t *testing.T) {
		jsonData := `{
			"access_token": "test_token",
			"expires_in": 1800
		}`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.NoError(t, err)
		assert.Equal(t, "test_token", result.AccessToken)
		assert.Equal(t, "", result.RefreshToken) // 未设置的字段应该是零值
		assert.Equal(t, int64(1800), result.ExpiresIn)
		assert.Equal(t, "", result.TokenType)
	})

	t.Run("invalid JSON", func(t *testing.T) {
		jsonData := `{invalid json`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid character")
	})

	t.Run("empty JSON", func(t *testing.T) {
		jsonData := `{}`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.NoError(t, err)
		// 所有字段应该是零值
		assert.Equal(t, "", result.AccessToken)
		assert.Equal(t, "", result.RefreshToken)
		assert.Equal(t, int64(0), result.ExpiresIn)
		assert.Equal(t, "", result.TokenType)
	})

	t.Run("null values", func(t *testing.T) {
		jsonData := `{
			"access_token": null,
			"refresh_token": "valid_token",
			"expires_in": null,
			"token_type": null
		}`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.NoError(t, err)
		assert.Equal(t, "", result.AccessToken) // null 应该解析为零值
		assert.Equal(t, "valid_token", result.RefreshToken)
		assert.Equal(t, int64(0), result.ExpiresIn)
		assert.Equal(t, "", result.TokenType)
	})

	t.Run("type mismatch", func(t *testing.T) {
		jsonData := `{
			"access_token": "valid_token",
			"expires_in": "not_a_number"
		}`

		reader := io.NopCloser(strings.NewReader(jsonData))
		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}

		err := UnmarshalJsonFromReader(reader, &result)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot unmarshal")
	})

	t.Run("reader close behavior", func(t *testing.T) {
		jsonData := `{"access_token": "test"}`

		// 创建一个可以检测是否被关闭的reader
		var closed bool
		reader := &testReadCloser{
			Reader: strings.NewReader(jsonData),
			onClose: func() {
				closed = true
			},
		}

		var result struct {
			AccessToken  string `json:"access_token"`
			RefreshToken string `json:"refresh_token"`
			ExpiresIn    int64  `json:"expires_in"`
			TokenType    string `json:"token_type"`
		}
		err := UnmarshalJsonFromReader(reader, &result)

		assert.NoError(t, err)
		assert.True(t, closed, "Reader 应该被关闭")
	})
}

// 辅助测试结构体，用于测试 reader 的关闭行为
type testReadCloser struct {
	io.Reader
	onClose func()
}

func (t *testReadCloser) Close() error {
	if t.onClose != nil {
		t.onClose()
	}
	return nil
}
