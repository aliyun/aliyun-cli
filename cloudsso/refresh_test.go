package cloudsso

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
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
				Body: io.NopCloser(bytes.NewReader(func() []byte {
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
				Body:       io.NopCloser(bytes.NewReader([]byte(`{"ErrorCode": "Forbidden", "ErrorMessage": "Access Denied", "RequestId": "12345"}`))),
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

func TestTryRefreshStsTokenRejectsNilInputs(t *testing.T) {
	signInURL := strPtr("https://example.com")
	accessToken := strPtr("mockAccessToken")
	accessConfig := strPtr("mockAccessConfig")
	accountID := strPtr("mockAccountId")

	tests := []struct {
		name         string
		accessToken  *string
		accessConfig *string
		accountID    *string
		wantError    string
	}{
		{name: "nil access token", accessConfig: accessConfig, accountID: accountID, wantError: "accessToken is nil"},
		{name: "nil access config", accessToken: accessToken, accountID: accountID, wantError: "accessConfig is nil"},
		{name: "nil account ID", accessToken: accessToken, accessConfig: accessConfig, wantError: "accountId is nil"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := TryRefreshStsToken(signInURL, tt.accessToken, tt.accessConfig, tt.accountID, nil)
			if err == nil || err.Error() != tt.wantError {
				t.Fatalf("TryRefreshStsToken() error = %v, want %q", err, tt.wantError)
			}
		})
	}
}

func TestCreateCloudCredentialResponseShapes(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		wantErr   string
		wantKeyID string
	}{
		{
			name:      "nested current response",
			body:      `{"CloudCredential":{"AccessKeyId":"nested-key","Expiration":"2015-04-09T11:52:19Z"},"RequestId":"request-id"}`,
			wantKeyID: "nested-key",
		},
		{
			name:      "top-level legacy response",
			body:      `{"AccessKeyId":"legacy-key","Expiration":"2015-04-09T11:52:19Z"}`,
			wantKeyID: "legacy-key",
		},
		{
			name:    "missing credential",
			body:    `{"RequestId":"request-id"}`,
			wantErr: "cloud credential is missing from response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &http.Client{Transport: &MockHttpClient{DoFunc: func(*http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(tt.body)),
				}, nil
			}}}

			credential, err := CreateCloudCredential("https://example.com", "token", CloudCredentialOptions{}, client)
			if tt.wantErr != "" {
				if err == nil || err.Error() != tt.wantErr {
					t.Fatalf("CreateCloudCredential() error = %v, want %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("CreateCloudCredential() error = %v", err)
			}
			if credential.AccessKeyId != tt.wantKeyID {
				t.Fatalf("AccessKeyId = %q, want %q", credential.AccessKeyId, tt.wantKeyID)
			}
			if credential.ExpirationInt64 == 0 {
				t.Fatal("ExpirationInt64 was not populated")
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
