package cloudsso

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"testing/iotest"
)

func TestAccountAndConfigurationReadFailures(t *testing.T) {
	for _, operation := range []string{"accounts", "configurations"} {
		for _, tc := range []struct {
			name, body string
			status     int
			err        error
			readError  bool
		}{
			{name: "transport", err: errors.New("offline")},
			{name: "body read", readError: true},
			{name: "bad error JSON", status: 403, body: "{"},
			{name: "bad success JSON", status: 200, body: "{"},
		} {
			t.Run(operation+"/"+tc.name, func(t *testing.T) {
				client := &http.Client{Transport: &MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
					if tc.err != nil {
						return nil, tc.err
					}
					if tc.readError {
						return &http.Response{StatusCode: 200, Body: io.NopCloser(iotest.ErrReader(errors.New("read failed")))}, nil
					}
					return mockHTTPResponse(tc.status, tc.body), nil
				}}}
				var err error
				if operation == "accounts" {
					_, err = (&ListUserParameter{BaseUrl: "https://sso.example", HttpClient: client}).ListUsers("next", 0)
				} else {
					_, err = (&AccessConfigurationsParameter{UrlPrefix: "https://sso.example", HttpClient: client}).ListAccessConfigurationsForAccount(AccessConfigurationsRequest{AccountId: "account", NextToken: "next"})
				}
				if err == nil {
					t.Fatal("read failure was accepted")
				}
			})
		}
	}
	if _, err := (&ListUserParameter{BaseUrl: ":invalid"}).ListUsers("", 0); err == nil {
		t.Fatal("invalid account base URL accepted")
	}
	if _, err := (&AccessConfigurationsParameter{UrlPrefix: ":invalid"}).ListAccessConfigurationsForAccount(AccessConfigurationsRequest{}); err == nil {
		t.Fatal("invalid configuration base URL accepted")
	}
}

func TestPaginationReturnsLaterPageErrors(t *testing.T) {
	for _, operation := range []string{"accounts", "configurations"} {
		t.Run(operation, func(t *testing.T) {
			requests := 0
			client := &http.Client{Transport: &MockHTTPClient{DoFunc: func(req *http.Request) (*http.Response, error) {
				requests++
				if requests == 1 {
					return mockHTTPResponse(200, `{"IsTruncated":true,"NextToken":"page-two"}`), nil
				}
				if req.URL.Query().Get("NextToken") != "page-two" {
					t.Errorf("next token = %q", req.URL.Query().Get("NextToken"))
				}
				return nil, errors.New("second page failed")
			}}}
			var err error
			if operation == "accounts" {
				_, err = (&ListUserParameter{BaseUrl: "https://sso.example", HttpClient: client}).ListAllUsers()
			} else {
				_, err = (&AccessConfigurationsParameter{UrlPrefix: "https://sso.example", HttpClient: client}).ListAllAccessConfigurations(AccessConfigurationsRequest{})
			}
			if err == nil || !strings.Contains(err.Error(), "second page failed") || requests != 2 {
				t.Fatalf("requests %d, error %v", requests, err)
			}
		})
	}
}

func TestCloudCredentialReadAndRequestErrors(t *testing.T) {
	if _, err := CreateCloudCredential(":invalid", "token", CloudCredentialOptions{}, nil); err == nil {
		t.Fatal("invalid credential URL accepted")
	}
	for _, tc := range []struct {
		name, body string
		status     int
		readError  bool
	}{
		{name: "body read", readError: true},
		{name: "non JSON error", status: 403, body: "forbidden"},
		{name: "bad JSON", status: 200, body: "{"},
		{name: "bad expiration", status: 200, body: `{"CloudCredential":{"AccessKeyId":"id","Expiration":"tomorrow"}}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := &http.Client{Transport: &MockHTTPClient{DoFunc: func(*http.Request) (*http.Response, error) {
				if tc.readError {
					return &http.Response{StatusCode: 200, Body: io.NopCloser(iotest.ErrReader(errors.New("broken")))}, nil
				}
				return mockHTTPResponse(tc.status, tc.body), nil
			}}}
			if _, err := CreateCloudCredential("https://sso.example", "token", CloudCredentialOptions{}, client); err == nil {
				t.Fatal("invalid credential response accepted")
			}
		})
	}
	bad := ":invalid"
	if _, err := ParseUrl(&bad); err == nil {
		t.Fatal("invalid URL accepted")
	}
	if _, err := TryRefreshStsToken(&bad, nil, nil, nil, nil); err == nil {
		t.Fatal("invalid login URL accepted")
	}
}
