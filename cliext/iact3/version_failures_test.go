package iact3

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/aliyun/aliyun-cli/v3/cli"
)

func TestLatestIact3VersionFailures(t *testing.T) {
	oldTransport, oldURL := http.DefaultTransport, iact3BaseUrl
	t.Cleanup(func() { http.DefaultTransport = oldTransport; iact3BaseUrl = oldURL })
	for _, tc := range []struct {
		name, body            string
		status                int
		readFail, networkFail bool
		want                  string
	}{
		{name: "valid", body: " 1.2.3 \n", status: 200, want: "1.2.3"}, {name: "empty", body: " ", status: 200}, {name: "status", body: "missing", status: 404}, {name: "body truncated", status: 200, readFail: true}, {name: "network", networkFail: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			http.DefaultTransport = credentialTransport(func(r *http.Request) (*http.Response, error) {
				if r.Header.Get("User-Agent") != "aliyun-cli/"+cli.Version {
					t.Errorf("user agent=%q", r.Header.Get("User-Agent"))
				}
				if tc.networkFail {
					return nil, errors.New("offline")
				}
				var body io.ReadCloser = io.NopCloser(strings.NewReader(tc.body))
				if tc.readFail {
					body = archiveReadError{}
				}
				return &http.Response{StatusCode: tc.status, Body: body}, nil
			})
			got, err := GetLatestIact3Version()
			if got != tc.want || (err == nil) != (tc.want != "") {
				t.Fatalf("version=%q,error=%v", got, err)
			}
		})
	}
	iact3BaseUrl = "://invalid"
	if _, err := GetLatestIact3Version(); err == nil {
		t.Fatal("invalid request URL accepted")
	}
}
