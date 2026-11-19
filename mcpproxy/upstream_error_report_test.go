package mcpproxy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

type timeoutError struct{}

func (timeoutError) Error() string   { return "i/o timeout" }
func (timeoutError) Timeout() bool   { return true }
func (timeoutError) Temporary() bool { return true }

func TestUpstreamErrorReportKeepsCategoryWithoutSecrets(t *testing.T) {
	plain := errors.New("dial tcp 10.0.0.1:443: connect: connection refused")
	urlWrapTransport := &url.Error{Op: "Post", URL: fakeUpstreamTarget(), Err: plain}
	urlWrapNetwork := &url.Error{Op: "Post", URL: fakeUpstreamTarget(), Err: &net.OpError{Op: "dial", Net: "tcp", Err: plain}}

	for _, tc := range []struct {
		name     string
		err      error
		category string
	}{
		{name: "nil error", err: nil, category: "unknown"},
		{name: "plain transport error", err: plain, category: "transport"},
		{name: "url error wrapping plain", err: urlWrapTransport, category: "transport"},
		{name: "url error wrapping network", err: urlWrapNetwork, category: "network"},
		{name: "multilevel wrap keeps network", err: fmt.Errorf("failed to send request: %w", urlWrapNetwork), category: "network"},
		{name: "deadline exceeded wrapped", err: fmt.Errorf("wrapped: %w", context.DeadlineExceeded), category: "timeout"},
		{name: "net timeout", err: timeoutError{}, category: "timeout"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			report := upstreamErrorReport("initial", tc.err, false)
			require.Equal(t, "phase=initial category="+tc.category, report)
			requireNoUpstreamSecrets(t, "error report", report)
		})
	}

	t.Run("unparsable target falls back to unknown", func(t *testing.T) {
		bad := &url.Error{Op: "Post", URL: "https://userinfo-secret-20261103@upstream.example.com/%zz", Err: plain}
		report := upstreamErrorReport("initial", bad, true)
		require.Contains(t, report, "target=<unknown>")
		requireNoUpstreamSecrets(t, "error report", report)
	})

	t.Run("sanitized target keeps origin and path", func(t *testing.T) {
		report := upstreamErrorReport("retry_after_refresh", urlWrapNetwork, true)
		require.Equal(t, "phase=retry_after_refresh category=network target=https://upstream.example.com/mcp/secret-server", report)
		requireNoUpstreamSecrets(t, "error report", report)
	})
}
