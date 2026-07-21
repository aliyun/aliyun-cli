package openapi

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	openapiClient "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	openapiutil "github.com/alibabacloud-go/darabonba-openapi/v2/utils"
	openapiTeaUtils "github.com/alibabacloud-go/tea-utils/v2/service"
	"github.com/alibabacloud-go/tea/tea"
	"github.com/aliyun/aliyun-cli/v3/cli"
	"github.com/aliyun/aliyun-cli/v3/config"
	"github.com/aliyun/aliyun-cli/v3/logutil"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenapiThrottlingRetryHelpers(t *testing.T) {
	disabled := false
	cfg := &throttlingretry.Config{
		Enabled:     &disabled,
		MaxAttempts: 5,
		MaxDelayMS:  2000,
	}

	assert.False(t, openapiThrottlingRetryEnabled(cfg))
	enabled := true
	cfg.Enabled = &enabled
	assert.True(t, openapiThrottlingRetryEnabled(cfg))
	assert.True(t, openapiThrottlingRetryEnabled(&throttlingretry.Config{}))

	assert.Equal(t, 5, openapiThrottlingRetryMaxAttempts(cfg))
	assert.Equal(t, defaultThrottlingRetryMaxAttempts, openapiThrottlingRetryMaxAttempts(&throttlingretry.Config{}))

	assert.Equal(t, int64(2000), openapiThrottlingRetryMaxDelayMS(cfg))
	assert.Equal(t, defaultThrottlingRetryMaxDelayMS, openapiThrottlingRetryMaxDelayMS(&throttlingretry.Config{}))
}

func TestOpenapiThrottlingRetryDelay(t *testing.T) {
	cfg := &throttlingretry.Config{MaxDelayMS: 1000}

	delay, ok := openapiThrottlingRetryDelay(newOpenAPThrottlingError("Throttling.User", 500), cfg)
	assert.True(t, ok)
	assert.Equal(t, int64(500), delay)

	delay, ok = openapiThrottlingRetryDelay(newOpenAPThrottlingError("Throttling.Api", 1500), cfg)
	assert.True(t, ok)
	assert.Equal(t, int64(1000), delay)

	disabled := false
	disabledCfg := &throttlingretry.Config{Enabled: &disabled}
	_, ok = openapiThrottlingRetryDelay(newOpenAPThrottlingError("Throttling", 100), disabledCfg)
	assert.False(t, ok)

	_, ok = openapiThrottlingRetryDelay(errors.New("plain"), cfg)
	assert.False(t, ok)

	// Non-Throttling code with retryAfter still counts as throttling.
	delay, ok = openapiThrottlingRetryDelay(newOpenAPThrottlingError("ServiceUnavailable", 100), cfg)
	assert.True(t, ok)
	assert.Equal(t, int64(100), delay)

	err := newOpenAPThrottlingError("Throttling", 100)
	throttlingErr := err.(*openapiClient.ThrottlingError)
	throttlingErr.RetryAfter = nil
	_, ok = openapiThrottlingRetryDelay(err, cfg)
	assert.False(t, ok)

	negative := int64(-1)
	throttlingErr.RetryAfter = &negative
	_, ok = openapiThrottlingRetryDelay(err, cfg)
	assert.False(t, ok)
}

func TestApplyOpenAPIRetryHeaders(t *testing.T) {
	headers := map[string]*string{}
	applyOpenAPIRetryHeaders(nil, 1, 100)
	applyOpenAPIRetryHeaders(headers, 0, 100)
	assert.Empty(t, headers)

	applyOpenAPIRetryHeaders(headers, 2, 250)
	assert.Equal(t, "2", tea.StringValue(headers["x-acs-retry-attempts"]))
	assert.Equal(t, "250", tea.StringValue(headers["x-acs-retry-delay"]))
}

func TestOpenapiThrottlingRetryConfigLoadError(t *testing.T) {
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "config.json")
	require.NoError(t, os.WriteFile(configFile, []byte(`{"current":"default","profiles":[]}`), 0600))
	require.NoError(t, os.Mkdir(filepath.Join(configDir, throttlingretry.ConfigFileName), 0755))

	ctx := cli.NewCommandContext(new(bytes.Buffer), new(bytes.Buffer))
	config.AddFlags(ctx.Flags())
	config.ConfigurePathFlag(ctx.Flags()).SetAssigned(true)
	config.ConfigurePathFlag(ctx.Flags()).SetValue(configFile)

	cfg := openapiThrottlingRetryConfig(ctx)
	assert.NotNil(t, cfg)
}

func TestHttpContextCallThrottlingRetry(t *testing.T) {
	origExecute := httpContextExecuteFunc
	t.Cleanup(func() {
		httpContextExecuteFunc = origExecute
		logutil.SetOutput(nil)
		logutil.SetLevel(logutil.Error)
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Info)

	calls := 0
	httpContextExecuteFunc = func(a *HttpContext) (map[string]interface{}, error) {
		calls++
		if calls < 3 {
			return nil, newOpenAPThrottlingError("Throttling.User", 1)
		}
		return map[string]interface{}{"body": "ok"}, nil
	}

	ctx := &HttpContext{
		throttlingRetryConfig: &throttlingretry.Config{MaxAttempts: 3, MaxDelayMS: 60000},
		openapiRequest: &openapiutil.OpenApiRequest{
			Headers: map[string]*string{},
		},
		openapiParams:  &openapiClient.Params{},
		openapiRuntime: &openapiTeaUtils.RuntimeOptions{},
	}

	err := ctx.Call()
	require.NoError(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, "2", tea.StringValue(ctx.openapiRequest.Headers["x-acs-retry-attempts"]))
	assert.Equal(t, "1", tea.StringValue(ctx.openapiRequest.Headers["x-acs-retry-delay"]))
	assert.Contains(t, logBuf.String(), "retrying in 1ms (attempt 1/3)")
}

func TestHttpContextCallThrottlingExhausted(t *testing.T) {
	origExecute := httpContextExecuteFunc
	t.Cleanup(func() {
		httpContextExecuteFunc = origExecute
		logutil.SetOutput(nil)
		logutil.SetLevel(logutil.Error)
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Info)

	httpContextExecuteFunc = func(a *HttpContext) (map[string]interface{}, error) {
		return nil, newOpenAPThrottlingError("Throttling.User", 1)
	}

	ctx := &HttpContext{
		throttlingRetryConfig: &throttlingretry.Config{MaxAttempts: 2},
		openapiRequest: &openapiutil.OpenApiRequest{
			Headers: map[string]*string{},
		},
		openapiParams:  &openapiClient.Params{},
		openapiRuntime: &openapiTeaUtils.RuntimeOptions{},
	}

	err := ctx.Call()
	require.Error(t, err)
	assert.Contains(t, logBuf.String(), "still throttled after 2 attempts, giving up")
}

func TestHttpContextCallNonThrottlingError(t *testing.T) {
	origExecute := httpContextExecuteFunc
	httpContextExecuteFunc = func(a *HttpContext) (map[string]interface{}, error) {
		return nil, errors.New("plain error")
	}
	t.Cleanup(func() {
		httpContextExecuteFunc = origExecute
	})

	ctx := &HttpContext{
		throttlingRetryConfig: &throttlingretry.Config{MaxAttempts: 2},
		openapiRequest: &openapiutil.OpenApiRequest{
			Headers: map[string]*string{},
		},
		openapiParams:  &openapiClient.Params{},
		openapiRuntime: &openapiTeaUtils.RuntimeOptions{},
	}

	err := ctx.Call()
	require.Error(t, err)
}

func newOpenAPThrottlingError(code string, retryAfterMS int64) error {
	return &openapiClient.ThrottlingError{
		Code:       tea.String(code),
		RetryAfter: tea.Int64(retryAfterMS),
	}
}
