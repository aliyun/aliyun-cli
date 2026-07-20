package openapi

import (
	"bytes"
	"errors"
	"testing"
	"time"

	sdkerrors "github.com/aliyun/alibaba-cloud-sdk-go/sdk/errors"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/responses"
	"github.com/aliyun/aliyun-cli/v3/logutil"
	"github.com/aliyun/aliyun-cli/v3/sysconfig/throttlingretry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCallWithThrottlingRetry(t *testing.T) {
	originSleep := throttlingRetrySleep
	throttlingRetrySleep = func(time.Duration) {}
	t.Cleanup(func() {
		throttlingRetrySleep = originSleep
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Info)
	t.Cleanup(func() {
		logutil.SetOutput(nil)
		logutil.SetLevel(logutil.Error)
	})

	invoker := &BasicInvoker{
		throttlingRetryConfig: &throttlingretry.Config{
			MaxAttempts: 2,
			MaxDelayMS:  50,
		},
	}

	calls := 0
	resp, err := invoker.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		calls++
		if calls == 1 {
			return nil, newThrottlingServerError("Throttling.User", "50")
		}
		return &responses.CommonResponse{}, nil
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 2, calls)
	assert.Equal(t, "aliyun: throttling, retrying in 50ms (attempt 1/2)\n", logBuf.String())
}

func TestCallWithThrottlingRetryHiddenByDefault(t *testing.T) {
	originSleep := throttlingRetrySleep
	throttlingRetrySleep = func(time.Duration) {}
	t.Cleanup(func() {
		throttlingRetrySleep = originSleep
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Error)
	t.Cleanup(func() {
		logutil.SetOutput(nil)
	})

	invoker := &BasicInvoker{
		throttlingRetryConfig: &throttlingretry.Config{
			MaxAttempts: 2,
			MaxDelayMS:  50,
		},
	}

	calls := 0
	_, err := invoker.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		calls++
		if calls == 1 {
			return nil, newThrottlingServerError("Throttling.User", "50")
		}
		return &responses.CommonResponse{}, nil
	})

	require.NoError(t, err)
	assert.Equal(t, 2, calls)
	assert.Empty(t, logBuf.String())
}

func TestCallWithThrottlingRetryPrintsEachRetryNotice(t *testing.T) {
	originSleep := throttlingRetrySleep
	throttlingRetrySleep = func(time.Duration) {}
	t.Cleanup(func() {
		throttlingRetrySleep = originSleep
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Info)
	t.Cleanup(func() {
		logutil.SetOutput(nil)
		logutil.SetLevel(logutil.Error)
	})

	invoker := &BasicInvoker{
		request: requests.NewCommonRequest(),
		throttlingRetryConfig: &throttlingretry.Config{
			MaxAttempts: 3,
			MaxDelayMS:  60000,
		},
	}

	type retryHeaders struct {
		attempts string
		delay    string
	}
	headersByCall := make([]retryHeaders, 0, 3)
	calls := 0
	resp, err := invoker.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		calls++
		headersByCall = append(headersByCall, retryHeaders{
			attempts: invoker.request.Headers["x-acs-retry-attempts"],
			delay:    invoker.request.Headers["x-acs-retry-delay"],
		})
		switch calls {
		case 1:
			return nil, newThrottlingServerError("Throttling.User", "100")
		case 2:
			return nil, newThrottlingServerError("Throttling.Api", "200")
		default:
			return &responses.CommonResponse{}, nil
		}
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 3, calls)
	assert.Equal(t, []retryHeaders{
		{attempts: "", delay: ""},
		{attempts: "1", delay: "100"},
		{attempts: "2", delay: "200"},
	}, headersByCall)
	assert.Equal(t, ""+
		"aliyun: throttling, retrying in 100ms (attempt 1/3)\n"+
		"aliyun: throttling, retrying in 200ms (attempt 2/3)\n",
		logBuf.String())
}

func TestCallWithThrottlingRetryExhaustedNotice(t *testing.T) {
	originSleep := throttlingRetrySleep
	throttlingRetrySleep = func(time.Duration) {}
	t.Cleanup(func() {
		throttlingRetrySleep = originSleep
	})

	var logBuf bytes.Buffer
	logutil.SetOutput(&logBuf)
	logutil.SetLevel(logutil.Info)
	t.Cleanup(func() {
		logutil.SetOutput(nil)
		logutil.SetLevel(logutil.Error)
	})

	invoker := &BasicInvoker{
		throttlingRetryConfig: &throttlingretry.Config{
			MaxAttempts: 2,
			MaxDelayMS:  60000,
		},
	}

	calls := 0
	_, err := invoker.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		calls++
		return nil, newThrottlingServerError("Throttling.User", "100")
	})

	require.Error(t, err)
	assert.Equal(t, 3, calls)
	assert.Equal(t, ""+
		"aliyun: throttling, retrying in 100ms (attempt 1/2)\n"+
		"aliyun: throttling, retrying in 100ms (attempt 2/2)\n"+
		"aliyun: still throttled after 2 attempts, giving up\n",
		logBuf.String())
}

func TestCallWithThrottlingRetryStopsWhenDisabled(t *testing.T) {
	enabled := false
	invoker := &BasicInvoker{
		throttlingRetryConfig: &throttlingretry.Config{Enabled: &enabled},
	}

	calls := 0
	_, err := invoker.callWithThrottlingRetry(func() (*responses.CommonResponse, error) {
		calls++
		return nil, newThrottlingServerError("Throttling.User", "100")
	})

	require.Error(t, err)
	assert.Equal(t, 1, calls)
}

func TestThrottlingRetryDelayRequiresRetryAfterHeader(t *testing.T) {
	invoker := &BasicInvoker{}

	delay, ok := invoker.throttlingRetryDelay(newThrottlingServerError("Throttling.Api", "123"))
	assert.True(t, ok)
	assert.Equal(t, int64(123), delay)

	delay, ok = invoker.throttlingRetryDelay(newThrottlingServerError("InternalError", "123"))
	assert.True(t, ok)
	assert.Equal(t, int64(123), delay)

	_, ok = invoker.throttlingRetryDelay(newThrottlingServerError("Throttling.Api", "not-number"))
	assert.False(t, ok)

	_, ok = invoker.throttlingRetryDelay(errors.New("plain error"))
	assert.False(t, ok)
}

func newThrottlingServerError(code string, retryAfter string) error {
	err := sdkerrors.NewServerError(400, `{"Code":"`+code+`","Message":"too many requests"}`, "")
	serverErr := err.(*sdkerrors.ServerError)
	serverErr.RespHeaders = map[string][]string{
		"x-acs-retry-after": {retryAfter},
	}
	return serverErr
}

func TestRetryAfterFromHeaders(t *testing.T) {
	delay, ok := retryAfterFromHeaders(map[string][]string{
		"X-Acs-Retry-After": {" 42 "},
	})
	assert.True(t, ok)
	assert.Equal(t, int64(42), delay)

	_, ok = retryAfterFromHeaders(map[string][]string{
		"x-acs-retry-after": {},
	})
	assert.False(t, ok)

	_, ok = retryAfterFromHeaders(map[string][]string{
		"x-acs-retry-after": {"bad"},
	})
	assert.False(t, ok)

	_, ok = retryAfterFromHeaders(map[string][]string{
		"x-acs-retry-after": {"-1"},
	})
	assert.False(t, ok)
}

func TestApplyRetryRequestHeadersEdgeCases(t *testing.T) {
	var invoker *BasicInvoker
	invoker.applyRetryRequestHeaders(1, 100)

	invoker = &BasicInvoker{}
	invoker.applyRetryRequestHeaders(0, 100)
	invoker.applyRetryRequestHeaders(1, 100)

	invoker.request = requests.NewCommonRequest()
	invoker.request.Headers = nil
	invoker.applyRetryRequestHeaders(2, 250)
	assert.Equal(t, "2", invoker.request.Headers["x-acs-retry-attempts"])
	assert.Equal(t, "250", invoker.request.Headers["x-acs-retry-delay"])
}

func TestThrottlingRetryMaxDelayCapsDelay(t *testing.T) {
	invoker := &BasicInvoker{
		throttlingRetryConfig: &throttlingretry.Config{MaxDelayMS: 80},
	}
	delay, ok := invoker.throttlingRetryDelay(newThrottlingServerError("Throttling.User", "200"))
	assert.True(t, ok)
	assert.Equal(t, int64(80), delay)
}
