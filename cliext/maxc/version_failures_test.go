package maxc

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestLatestVersionRequestFailures(t *testing.T) {
	old := httpGetFunc
	t.Cleanup(func() { httpGetFunc = old })
	c := withCtx(t, "linux-amd64")
	for _, tc := range []struct {
		name, body            string
		status                int
		requestFail, bodyFail bool
	}{
		{name: "network", requestFail: true}, {name: "status", status: 403, body: "denied"}, {name: "body", status: 200, bodyFail: true}, {name: "empty", status: 200, body: " "},
	} {
		t.Run(tc.name, func(t *testing.T) {
			httpGetFunc = func(string) (*http.Response, error) {
				if tc.requestFail {
					return nil, errors.New("offline")
				}
				var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.body))
				if tc.bodyFail {
					body = maxcBodyError{}
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			}
			if _, err := c.GetLatestVersion(); err == nil {
				t.Fatal("invalid version response accepted")
			}
		})
	}
}
