package upgrade

import (
	"context"
	"encoding/json"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestReadOnlyCheckRespectsBridgeAndTimeout(t *testing.T) {
	previous := httpClient
	t.Cleanup(func() { httpClient = previous })
	httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) {
		body := "3.9.0"
		if req.URL.Path == "/versions" {
			body = `{"format":"aliyun-cli-versions v1","versions":[{"version":"3.6.0","public_keys":["1"]},{"version":"3.9.0","public_keys":["2"]}]}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body))}, nil
	})}
	r, err := CheckUpdate(context.Background(), "3.5.3")
	require.NoError(t, err)
	require.Equal(t, "3.9.0", r.Latest)
	require.Equal(t, "3.6.0", r.Target)
	body, err := json.Marshal(r)
	require.NoError(t, err)
	var fields map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(body, &fields))
	require.Len(t, fields, 7)
	for _, key := range []string{"current", "latest", "target", "update_available", "status", "command", "checked_at"} {
		require.Contains(t, fields, key)
	}
	httpClient = &http.Client{Transport: roundTripperFunc(func(req *http.Request) (*http.Response, error) { return nil, req.Context().Err() })}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = CheckUpdate(ctx, "3.5.3")
	require.Error(t, err)
}
