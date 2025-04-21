package cloudsso

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"testing"
)

func TestParseUrl(t *testing.T) {
	tests := []struct {
		name      string
		input     *string
		want      []string
		expectErr bool
	}{
		{
			name:      "Valid URL",
			input:     strPtr("https://example.com"),
			want:      []string{"https", "example.com"},
			expectErr: false,
		},
		{
			name:      "Invalid URL - Missing scheme",
			input:     strPtr("example.com"),
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Invalid URL - Empty input",
			input:     strPtr(""),
			want:      nil,
			expectErr: true,
		},
		{
			name:      "Nil input",
			input:     nil,
			want:      nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseUrl(tt.input)
			if (err != nil) != tt.expectErr {
				t.Errorf("ParseUrl() error = %v, expectErr %v", err, tt.expectErr)
				return
			}
			if !equalSlices(got, tt.want) {
				t.Errorf("ParseUrl() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTryRefreshStsToken(t *testing.T) {
	tests := []struct {
		name          string
		signInUrl     *string
		accessToken   *string
		accessConfig  *string
		accountId     *string
		mockResponse  *http.Response
		mockError     error
		expectErr     bool
		expectedToken *CloudCredentialResponse
	}{
		{
			name:         "Valid response",
			signInUrl:    strPtr("https://example.com"),
			accessToken:  strPtr("mockAccessToken"),
			accessConfig: strPtr("mockAccessConfig"),
			accountId:    strPtr("mockAccountId"),
			mockResponse: &http.Response{
				StatusCode: 200,
				Body: ioutil.NopCloser(bytes.NewReader(func() []byte {
					resp := CloudCredentialResponse{
						AccessKeyId:     "mockKeyId",
						AccessKeySecret: "mockKeySecret",
						SecurityToken:   "mockToken",
						Expiration:      "2015-04-09T11:52:19Z",
					}
					data, _ := json.Marshal(resp)
					return data
				}())),
			},
			mockError: nil,
			expectErr: false,
			expectedToken: &CloudCredentialResponse{
				AccessKeyId:     "mockKeyId",
				AccessKeySecret: "mockKeySecret",
				SecurityToken:   "mockToken",
				Expiration:      "2015-04-09T11:52:19Z",
			},
		},
		{
			name:         "Invalid URL",
			signInUrl:    strPtr(""),
			accessToken:  strPtr("mockAccessToken"),
			accessConfig: strPtr("mockAccessConfig"),
			accountId:    strPtr("mockAccountId"),
			mockResponse: nil,
			mockError:    nil,
			expectErr:    true,
		},
		{
			name:         "HTTP error",
			signInUrl:    strPtr("https://example.com"),
			accessToken:  strPtr("mockAccessToken"),
			accessConfig: strPtr("mockAccessConfig"),
			accountId:    strPtr("mockAccountId"),
			mockResponse: nil,
			mockError:    errors.New("mock HTTP error"),
			expectErr:    true,
		},
		{
			name:         "HTTP 403 error",
			signInUrl:    strPtr("https://example.com"),
			accessToken:  strPtr("mockAccessToken"),
			accessConfig: strPtr("mockAccessConfig"),
			accountId:    strPtr("mockAccountId"),
			mockResponse: &http.Response{
				StatusCode: 403,
				Body:       ioutil.NopCloser(bytes.NewReader([]byte(`{"ErrorCode": "Forbidden", "ErrorMessage": "Access Denied", "RequestId": "12345"}`))),
			},
			mockError: nil,
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 创建一个模拟的Transport
			mockTransport := &MockHttpClient{
				DoFunc: func(req *http.Request) (*http.Response, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			// 创建一个使用模拟Transport的HTTP客户端
			mockClient := &http.Client{
				Transport: mockTransport,
			}

			// 使用自定义的HTTP客户端调用函数
			got, err := TryRefreshStsToken(tt.signInUrl, tt.accessToken, tt.accessConfig, tt.accountId, mockClient)
			if (err != nil) != tt.expectErr {
				t.Errorf("TryRefreshStsToken() error = %v, expectErr %v", err, tt.expectErr)
				return
			}

			if !tt.expectErr {
				if !equalCloudCredentialResponse(got, tt.expectedToken) {
					t.Errorf("TryRefreshStsToken() = %v, want %v", got, tt.expectedToken)
				}
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func equalSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

type MockHttpClient struct {
	DoFunc func(req *http.Request) (*http.Response, error)
}

func (m *MockHttpClient) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.DoFunc != nil {
		return m.DoFunc(req)
	}
	return nil, errors.New("no mock function provided")
}

func equalCloudCredentialResponse(a, b *CloudCredentialResponse) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.AccessKeyId == b.AccessKeyId &&
		a.AccessKeySecret == b.AccessKeySecret &&
		a.SecurityToken == b.SecurityToken &&
		a.Expiration == b.Expiration
}
