package telemetry

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUploadDiagnosticsResponseRequestIDs(t *testing.T) {
	for _, test := range []struct {
		name, body, requestID, telemetryID string
		headers                            map[string]string
		status                             int
	}{
		{
			name: "headers_on_success", status: http.StatusAccepted,
			headers:   map[string]string{"X-Acs-Request-Id": "Gateway-ABC-123", "X-Telemetry-Request-Id": "collector:ABC-123"},
			body:      `{"requestId":"body-gateway","telemetryRequestId":"body-collector","secret":"private-body"}`,
			requestID: "Gateway-ABC-123", telemetryID: "collector:ABC-123",
		},
		{
			name: "body_on_http_failure", status: http.StatusServiceUnavailable,
			body:      `{"requestId":"Gateway-Body-456","telemetryRequestId":"Collector-Body-456","message":"private-error"}`,
			requestID: "Gateway-Body-456", telemetryID: "Collector-Body-456",
		},
		{
			name: "mixed_header_and_body", status: http.StatusAccepted,
			headers:   map[string]string{"X-Telemetry-Request-Id": "collector-header"},
			body:      `{"RequestId":"gateway-body","telemetryRequestId":"collector-body"}`,
			requestID: "gateway-body", telemetryID: "collector-header",
		},
		{
			name: "generic_header_fallback", status: http.StatusServiceUnavailable,
			headers: map[string]string{"X-Request-Id": "generic-request-id"}, body: "private-non-json-response",
			requestID: "generic-request-id",
		},
		{
			name: "invalid_header_uses_valid_body", status: http.StatusAccepted,
			headers: map[string]string{"X-Acs-Request-Id": "https://private-url?token=secret"},
			body:    `{"requestId":"valid-body-id"}`, requestID: "valid-body-id",
		},
		{
			name: "unsafe_ids_omitted", status: http.StatusServiceUnavailable,
			headers: map[string]string{"X-Acs-Request-Id": "/private/customer-path", "X-Telemetry-Request-Id": "private value"},
			body:    `{"requestId":"https://private-url?token=secret","telemetryRequestId":"private\nforged-log"}`,
		},
		{
			name: "oversized_body_omitted", status: http.StatusAccepted,
			body: `{"requestId":"body-id"}` + strings.Repeat(" ", maxTelemetryResponseBytes),
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(TelemetryLogDisableEnv, "0")
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				for key, value := range test.headers {
					w.Header().Set(key, value)
				}
				w.WriteHeader(test.status)
				_, _ = io.WriteString(w, test.body)
			}))
			defer server.Close()
			t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
			dir := t.TempDir()
			cache, err := WriteEvent(GetPublicCacheDir(dir), Event{EventID: diagnosticUploadEventID, Pipeline: PipelinePublic})
			require.NoError(t, err)
			require.NoError(t, UploadFile(dir, PipelinePublic, cache))
			records, raw := readUploadDiagnostics(t, dir)
			last := records[len(records)-1]
			require.Equal(t, test.status, last.HTTPStatus)
			require.Equal(t, test.requestID, last.RequestID)
			require.Equal(t, test.telemetryID, last.TelemetryRequestID)
			if test.status < 300 {
				require.Equal(t, "succeeded", last.Status)
			} else {
				require.Equal(t, "failed", last.Status)
				require.Equal(t, "http_error", last.Reason)
			}
			for _, secret := range []string{"private", "secret", server.URL, "customer-path"} {
				require.NotContains(t, raw, secret)
			}
			require.NoFileExists(t, cache)
		})
	}
}

func TestUploadRetainsResponseIDsWhenRedirectFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-Acs-Request-Id", "gateway-redirect")
		w.Header().Set("X-Telemetry-Request-Id", "collector-redirect")
		w.Header().Set("Location", "/private-location")
		w.WriteHeader(http.StatusFound)
	}))
	defer server.Close()
	t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
	result := postTelemetryBatchResult(nil, &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error {
		return errors.New("private-redirect-error")
	}})
	require.False(t, result.success)
	require.Equal(t, http.StatusFound, result.httpStatus)
	require.Equal(t, "gateway-redirect", result.requestID)
	require.Equal(t, "collector-redirect", result.telemetryRequestID)
	require.Equal(t, "network_error", result.errorKind)
}

func TestNetworkFailureWithoutResponseHasNoRequestIDs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	server.Close()
	t.Setenv("ALIBABA_CLOUD_CLI_TELEMETRY_ENDPOINT", server.URL)
	result := postTelemetryBatchResult(nil, nil)
	require.False(t, result.success)
	require.Equal(t, "network_error", result.errorKind)
	require.Zero(t, result.httpStatus)
	require.Empty(t, result.requestID)
	require.Empty(t, result.telemetryRequestID)
}

type countedRequestIDBody struct {
	io.Reader
	read int
}

func (body *countedRequestIDBody) Read(p []byte) (int, error) {
	n, err := body.Reader.Read(p)
	body.read += n
	return n, err
}

func (*countedRequestIDBody) Close() error { return nil }

func TestRequestIDExtractionReadsBoundedResponse(t *testing.T) {
	body := &countedRequestIDBody{Reader: bytes.NewReader(bytes.Repeat([]byte{'x'}, maxTelemetryResponseBytes*2))}
	resp := &http.Response{Header: make(http.Header), Body: body}
	requestID, collectorID := telemetryResponseIDs(resp, true)
	require.Empty(t, requestID)
	require.Empty(t, collectorID)
	require.Equal(t, maxTelemetryResponseBytes+1, body.read)

	resp.Header.Set("X-Acs-Request-Id", "gateway-id")
	resp.Header.Set("X-Telemetry-Request-Id", "collector-id")
	requestID, collectorID = telemetryResponseIDs(resp, true)
	require.Equal(t, "gateway-id", requestID)
	require.Equal(t, "collector-id", collectorID)
	require.Equal(t, maxTelemetryResponseBytes+1, body.read, "complete response headers avoid reading the body")

	resp.Header.Del("X-Acs-Request-Id")
	_, collectorID = telemetryResponseIDs(resp, false)
	require.Equal(t, "collector-id", collectorID)
	require.Equal(t, maxTelemetryResponseBytes+1, body.read, "an errored response must not read the closed body")
}

func TestRequestIDResponseReadFailureKeepsHeader(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "closed-body")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	resp := &http.Response{Header: make(http.Header), Body: f}
	resp.Header.Set("X-Telemetry-Request-Id", "collector-id")
	requestID, collectorID := telemetryResponseIDs(resp, true)
	require.Empty(t, requestID)
	require.Equal(t, "collector-id", collectorID)
}
